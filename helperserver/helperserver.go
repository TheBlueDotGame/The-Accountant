package helperserver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/versioning"
	"github.com/bartossh/Computantis/webhooks"
)

const (
	Header = "Computantis-Web-Hooks"
)

const (
	AliveURL           = notaryserver.AliveURL   // URL to check is service alive
	MetricsURL         = notaryserver.MetricsURL // URL to serve service metrics over http.
	TransactionHookURL = "/transaction/new"      // URL allows to create transaction hook.
)

// Config contains configuration of the validator.
type Config struct {
	Port int `yaml:"port"` // port on which validator will listen for http requests
}

// WebhookCreateRemovePoster provides methods to create, remove webhooks and post messages to webhooks.
type WebhookCreateRemovePoster interface {
	CreateWebhook(trigger byte, address string, h webhooks.Hook) error
	RemoveWebhook(trigger byte, address string, h webhooks.Hook) error
	PostWebhookBlock(blc *block.Block)
	PostWebhookNewTransaction(publicAddresses []string, storingNodeURL string)
}

// NodesComunicationSubscriber provides facade access to communication between nodes publisher endpoint.
type NodesComunicationSubscriber interface {
	SubscribeNewTransactionsForAddresses(call transaction.TrxAddressesSubscriberCallback, log logger.Logger) error
}

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type app struct {
	cancel       context.CancelFunc
	ver          verifier
	wh           WebhookCreateRemovePoster
	randDataProv notaryserver.RandomDataProvideValidator
	log          logger.Logger
	mux          sync.RWMutex
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// It will block until the context is canceled.
func Run(
	ctx context.Context, cfg Config, sub NodesComunicationSubscriber, log logger.Logger, ver verifier, wh WebhookCreateRemovePoster,
	rdp notaryserver.RandomDataProvideValidator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	a := &app{
		log:          log,
		ver:          ver,
		wh:           wh,
		randDataProv: rdp,
		cancel:       cancel,
		mux:          sync.RWMutex{},
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		return errors.New("port out of range 0 - 65535")
	}

	if err := sub.SubscribeNewTransactionsForAddresses(a.processNewTrxIssuedByAddresses, log); err != nil {
		return err
	}

	return a.runServer(ctxx, cancel, cfg.Port)
}

func (a *app) runServer(ctx context.Context, cancel context.CancelFunc, port int) error {
	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  versioning.Header,
		AppName:       versioning.ApiVersion,
		Concurrency:   4096,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: Header}))
	router.Get(AliveURL, a.alive)

	router.Post(notaryserver.DataToValidateURL, a.data)
	router.Post(TransactionHookURL, a.transactions)

	go func() {
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", port))
		if err != nil {
			cancel()
		}
	}()

	<-ctx.Done()

	if err := router.Shutdown(); err != nil {
		return err
	}

	return nil
}

func (a *app) processNewTrxIssuedByAddresses(receivers []string, storingNodeURL string) {
	go a.wh.PostWebhookNewTransaction(receivers, storingNodeURL) // post concurrently
}

func (s *app) alive(c *fiber.Ctx) error {
	return c.JSON(
		notaryserver.AliveResponse{
			Alive:      true,
			APIVersion: versioning.ApiVersion,
			APIHeader:  Header,
		})
}

func (a *app) data(c *fiber.Ctx) error {
	var req notaryserver.DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("/data endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if req.Address == "" {
		a.log.Error("wrong JSON format for requesting data to sing")
		return fiber.ErrBadRequest
	}

	d := a.randDataProv.ProvideData(req.Address)
	return c.JSON(notaryserver.DataToSignResponse{Data: d})
}

func (a *app) transactions(c *fiber.Ctx) error {
	var req CreateRemoveUpdateHookRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, failed to parse request body: %s", TransactionHookURL, err.Error()))
		return fiber.ErrBadRequest
	}

	if req.Address == "" || req.Data == nil || req.Signature == nil || req.URL == "" || req.Digest == [32]byte{} {
		a.log.Error("wrong JSON format when requesting blocks")
		return fiber.ErrBadRequest
	}

	if ok := a.randDataProv.ValidateData(req.Address, req.Data); !ok {
		a.log.Error("%s endpoint, corrupted data")
		return fiber.ErrForbidden
	}

	buf := make([]byte, 0, len(req.Data)+len(req.URL))
	buf = append(buf, append(req.Data, []byte(req.URL)...)...)

	if err := a.ver.Verify(buf, req.Signature, [32]byte(req.Digest), req.Address); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, invalid signature: %s", TransactionHookURL, err.Error()))
		return fiber.ErrForbidden
	}

	h := webhooks.Hook{
		URL:   req.URL,
		Token: string(req.Data),
	}
	if err := a.wh.CreateWebhook(webhooks.TriggerNewTransaction, req.Address, h); err != nil {
		a.log.Error(fmt.Sprintf("%s failed to create webhook: %s", TransactionHookURL, err.Error()))
		return c.JSON(CreateRemoveUpdateHookResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(CreateRemoveUpdateHookResponse{Ok: true})
}
