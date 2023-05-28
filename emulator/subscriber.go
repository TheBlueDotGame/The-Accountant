package emulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/signerservice"
	"github.com/bartossh/Computantis/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

const (
	header     = "SubscriberEmulator"
	apiVersion = "1.0"
)

const WebHookEndpoint = "/hook/transaction"

var (
	ErrFailedHook = errors.New("failed to create web hook")
)

type subscriber struct {
	mux                  sync.Mutex
	pub                  publisher
	lastTransactionTime  time.Time
	allowedIssuerAddress string
}

// RunSubscriber runs subscriber emulator.
// To stop the subscriber cancel the context.
func RunSubscriber(ctx context.Context, cancel context.CancelFunc, config Config) error {
	defer cancel()

	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 20 {
		return fmt.Errorf("wrong timeout_seconds parameter, expected value between 1 and 20 inclusive")
	}

	p := publisher{
		timeout:       time.Second * time.Duration(config.TimeoutSeconds),
		signerAPIRoot: config.SignerServiceURL,
		random:        config.Random,
	}

	s := subscriber{
		mux:                 sync.Mutex{},
		pub:                 p,
		lastTransactionTime: time.Now(),
	}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * time.Duration(config.TimeoutSeconds),
		WriteTimeout:  time.Second * time.Duration(config.TimeoutSeconds),
		ServerHeader:  header,
		AppName:       apiVersion,
		Concurrency:   16,
	})

	router.Use(recover.New())
	router.Post(WebHookEndpoint, s.hook)

	var err error
	isServerRunning := true
	go func() {
		err = router.Listen(fmt.Sprintf("0.0.0.0:%v", config.Port))
		if err != nil {
			isServerRunning = false
			cancel()
		}
	}()

	defer func() {
		er := router.Shutdown()
		if er != nil {
			err = errors.Join(err, er)
		}
	}()

	time.Sleep(time.Second)

	if !isServerRunning {
		return err
	}

	var resAddress signerservice.AddressResponse

	err = p.makeGet(signerservice.Address, &resAddress)
	if err != nil {
		err = errors.Join(ErrFailedHook, err)
		return err
	}

	var resData server.DataToSignResponse
	reqData := server.DataToSignRequest{
		Address: resAddress.Address,
	}
	err = p.makePost(validator.DataEdnpoint, reqData, &resData)
	if err != nil {
		err = errors.Join(ErrFailedHook, err)
		return err
	}

	buf := make([]byte, 0, len(resData.Data)+len(config.SignerServiceURL)+len(WebHookEndpoint))
	buf = append(resData.Data, append([]byte(config.SignerServiceURL), []byte(WebHookEndpoint)...)...)
	var resSign signerservice.SignDataResponse
	reqSign := signerservice.SignDataRequest{
		Data: buf,
	}

	if err := p.makePost(signerservice.SignData, reqSign, &resSign); err != nil {
		err = errors.Join(ErrFailedHook, err)
		return err
	}

	var resHook validator.CreateRemoveUpdateHookResponse
	reqHook := validator.CreateRemoveUpdateHookRequest{
		URL:       config.SignerServiceURL + WebHookEndpoint,
		Address:   reqData.Address,
		Data:      resSign.Data,
		Signature: resSign.Signature,
		Digest:    resSign.Digest,
	}

	err = p.makePost(validator.NewTransactionEndpointHook, reqHook, &resHook)
	if err != nil {
		err = errors.Join(ErrFailedHook, err)
		return err
	}

	if !resHook.Ok {
		if resHook.Err != "" {
			err = errors.Join(ErrFailedHook, errors.New(resHook.Err))
			return err
		}
		err = ErrFailedHook
		return err
	}

	<-ctx.Done()
	return err
}

func (sub *subscriber) hook(ctx *fiber.Ctx) error {
	hookRes := make(map[string]bool)

	var res validator.NewTransactionMessage
	if err := ctx.BodyParser(&res); err != nil {
		// TODO: log
		hookRes["ack"] = false
		hookRes["ok"] = false
		return ctx.JSON(hookRes)
	}

	sub.mux.Lock()
	defer sub.mux.Unlock()

	if res.Time.Before(sub.lastTransactionTime) {
		// TODO: log
		hookRes["ack"] = true
		hookRes["ok"] = false
		return ctx.JSON(hookRes)
	}

	sub.lastTransactionTime = res.Time

	go sub.actOnTransactions() // make actions concurrently

	hookRes["ack"] = true
	hookRes["ok"] = true
	return ctx.JSON(hookRes)
}

func (sub *subscriber) actOnTransactions() {
	sub.mux.Lock()
	defer sub.mux.Unlock()

	var resReceivedTransactions signerservice.ReceivedTransactionResponse
	if err := sub.pub.makeGet(signerservice.GetReceivedTransactions, &resReceivedTransactions); err != nil {
		// TODO: log
		return
	}

	if !resReceivedTransactions.Ok {
		if resReceivedTransactions.Err != "" {
			// TODO: log err
		}
		return
	}

	var confirmRes signerservice.ConfirmTransactionResponse
	for _, transaction := range resReceivedTransactions.Transactions {
		confirmReq := signerservice.ConfirmTransactionRequest{
			Transaction: transaction,
		}

		if transaction.IssuerAddress != sub.allowedIssuerAddress {
			// TODO: log
			continue
		}
		if err := sub.pub.makePost(signerservice.ConfirmTransaction, confirmReq, &confirmRes); err != nil {
			// TODO: log
			continue
		}

		if !confirmRes.Ok {
			if confirmRes.Err != "" {
				// TODO: log err
			}
			// TODO: log
			continue
		}

		// TODO: log and do something with data inside
	}
}
