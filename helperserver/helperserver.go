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
	"github.com/bartossh/Computantis/webhooks"
)

const (
	Header = "Computantis-Helper"
)

const (
	AliveURL           = notaryserver.AliveURL   // URL to check is service alive
	MetricsURL         = notaryserver.MetricsURL // URL to serve service metrics over http.
	BlockHookURL       = "/block/new"            // URL allows to create block hook.
	TransactionHookURL = "/transaction/new"      // URL allows to create transaction hook.
)

var (
	ErrProofBlockIsInvalid    = errors.New("block proof is invalid")
	ErrBlockIndexIsInvalid    = errors.New("block index is invalid")
	ErrBlockPrevHashIsInvalid = errors.New("block previous hash is invalid")
	ErrBlockIsNil             = errors.New("block is nil")
)

// Config contains configuration of the validator.
type Config struct {
	Port int `yaml:"port"` // port on which validator will listen for http requests
}

// Status is a status of each received block by the validator.
// It keeps track of invalid blocks in case of blockchain corruption.
type Status struct {
	ID        any         `json:"-"          bson:"_id,omitempty" db:"id"`
	CreatedAt time.Time   `json:"created_at" bson:"created_at"    db:"created_at"`
	Block     block.Block `json:"block"      bson:"block"         db:"-"`
	Index     int64       `json:"index"      bson:"index"         db:"index"`
	Valid     bool        `json:"valid"      bson:"valid"         db:"valid"`
}

// StatusReadWriter provides methods to bulk read and single write validator status.
type StatusReadWriter interface {
	WriteValidatorStatus(ctx context.Context, vs *Status) error
	ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]Status, error)
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
	SubscribeNewBlock(call block.BlockSubscriberCallback, log logger.Logger) error
	SubscribeNewTransactionsForAddresses(call transaction.TrxAddressesSubscriberCallback, log logger.Logger) error
}

// Verifier provides methods to verify the signature of the message.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type app struct {
	mux          sync.RWMutex
	cancel       context.CancelFunc
	srw          StatusReadWriter
	ver          Verifier
	wh           WebhookCreateRemovePoster
	randDataProv notaryserver.RandomDataProvideValidator
	log          logger.Logger
	lastBlock    block.Block
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// It will block until the context is canceled.
func Run(
	ctx context.Context, cfg Config,
	sub NodesComunicationSubscriber, srw StatusReadWriter,
	log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster,
	rdp notaryserver.RandomDataProvideValidator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	a := &app{
		mux:          sync.RWMutex{},
		srw:          srw,
		log:          log,
		ver:          ver,
		wh:           wh,
		randDataProv: rdp,
		cancel:       cancel,
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		return errors.New("port out of range 0 - 65535")
	}

	if err := sub.SubscribeNewBlock(a.processBlock, log); err != nil {
		return err
	}

	if err := sub.SubscribeNewTransactionsForAddresses(a.processNewTrxIssued, log); err != nil {
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
		ServerHeader:  notaryserver.Header,
		AppName:       notaryserver.ApiVersion,
		Concurrency:   4096,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: "Validator Node"}))
	router.Get(AliveURL, a.alive)

	router.Post(notaryserver.DataToValidateURL, a.data)
	router.Post(BlockHookURL, a.blocks)
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

func (a *app) processBlock(b *block.Block, notaryNodeURL string) {
	lastBlockIndex := a.lastBlock.Index
	err := a.validateBlock(b)
	if err != nil {
		a.log.Error(fmt.Sprintf("notary node URL: [ %s ], block hash: [ %v ], %s", notaryNodeURL, b.Hash, err.Error()))
		return
	}
	a.log.Info(fmt.Sprintf("last block index: [ %v ] current block index: [ %v ] from notary node URL: [ %s ] \n", lastBlockIndex, b.Index, notaryNodeURL))
	go a.wh.PostWebhookBlock(b) // post concurrently
}

func (a *app) processNewTrxIssued(receivers []string, storingNodeURL string) {
	go a.wh.PostWebhookNewTransaction(receivers, storingNodeURL) // post concurrently
}

func (s *app) alive(c *fiber.Ctx) error {
	return c.JSON(
		notaryserver.AliveResponse{
			Alive:      true,
			APIVersion: notaryserver.ApiVersion,
			APIHeader:  notaryserver.Header,
		})
}

func (a *app) data(c *fiber.Ctx) error {
	var req notaryserver.DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("/data endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	d := a.randDataProv.ProvideData(req.Address)
	return c.JSON(notaryserver.DataToSignResponse{Data: d})
}

func (a *app) blocks(c *fiber.Ctx) error {
	var req CreateRemoveUpdateHookRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, failed to parse request body: %s", BlockHookURL, err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := a.randDataProv.ValidateData(req.Address, req.Data); !ok {
		a.log.Error("%s endpoint, corrupted data")
		return fiber.ErrForbidden
	}

	buf := make([]byte, 0, len(req.Data)+len(req.URL))
	buf = append(buf, append(req.Data, []byte(req.URL)...)...)

	if err := a.ver.Verify(buf, req.Signature, [32]byte(req.Digest), req.Address); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, invalid signature: %s", BlockHookURL, err.Error()))
		return fiber.ErrForbidden
	}

	h := webhooks.Hook{
		URL:   req.URL,
		Token: string(req.Data),
	}
	if err := a.wh.CreateWebhook(webhooks.TriggerNewBlock, req.Address, h); err != nil {
		a.log.Error(fmt.Sprintf("%s failed to create webhook: %s", BlockHookURL, err.Error()))
		return c.JSON(CreateRemoveUpdateHookResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(CreateRemoveUpdateHookResponse{Ok: true})
}

func (a *app) transactions(c *fiber.Ctx) error {
	var req CreateRemoveUpdateHookRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, failed to parse request body: %s", TransactionHookURL, err.Error()))
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
