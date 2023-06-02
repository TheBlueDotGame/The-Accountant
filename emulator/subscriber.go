package emulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bartossh/Computantis/httpclient"
	"github.com/bartossh/Computantis/walletapi"
	"github.com/bartossh/Computantis/webhooks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pterm/pterm"
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
		timeout:   time.Second * time.Duration(config.TimeoutSeconds),
		clientURL: config.ClientURL,
		random:    config.Random,
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

	var res walletapi.CreateWebhookResponse
	req := walletapi.CreateWebHookRequest{
		URL: fmt.Sprintf("%s%s", config.PublicURL, WebHookEndpoint),
	}
	url := fmt.Sprintf("%s%s", s.pub.clientURL, walletapi.CreateUpdateWebhook)
	if err := httpclient.MakePost(s.pub.timeout, url, req, &res); err != nil {
		return err
	}

	if !res.Ok {
		if res.Err != "" {
			return errors.New(res.Err)
		}
		return errors.New("unexpected error when creating the webkhook")
	}

	<-ctx.Done()
	return err
}

func (sub *subscriber) hook(ctx *fiber.Ctx) error {
	hookRes := make(map[string]bool)

	var res webhooks.NewTransactionMessage
	if err := ctx.BodyParser(&res); err != nil {
		pterm.Error.Println(err.Error())
		hookRes["ack"] = false
		hookRes["ok"] = false
		return ctx.JSON(hookRes)
	}

	sub.mux.Lock()
	defer sub.mux.Unlock()

	if res.Time.Before(sub.lastTransactionTime) {
		pterm.Error.Println("time is corrupted")
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

	var resReceivedTransactions walletapi.ReceivedTransactionResponse
	url := fmt.Sprintf("%s%s", sub.pub.clientURL, walletapi.GetReceivedTransactions)
	if err := httpclient.MakeGet(sub.pub.timeout, url, &resReceivedTransactions); err != nil {
		pterm.Error.Println(err.Error())
		return
	}

	if !resReceivedTransactions.Ok {
		if resReceivedTransactions.Err != "" {
			pterm.Error.Println(resReceivedTransactions.Err)
		}
		return
	}

	var counter int
	var confirmRes walletapi.ConfirmTransactionResponse

	for _, transaction := range resReceivedTransactions.Transactions {

		pterm.Info.Printf("Trx data: [ %s ]\n", string(transaction.Data))

		confirmReq := walletapi.ConfirmTransactionRequest{
			Transaction: transaction,
		}

		url := fmt.Sprintf("%s%s", sub.pub.clientURL, walletapi.ConfirmTransaction)
		if err := httpclient.MakePost(sub.pub.timeout, url, confirmReq, &confirmRes); err != nil {
			pterm.Error.Printf("Transaction cannot be signed, %s\n", err)
			continue
		}

		if !confirmRes.Ok {
			if confirmRes.Err != "" {
				pterm.Error.Printf("Transaction cannot be signed, %s\n", confirmRes.Err)
				continue
			}
			pterm.Error.Println("Transaction cannot be signed.")
			continue
		}
		counter++
	}

	if counter == len(resReceivedTransactions.Transactions) {
		pterm.Info.Printf("Signed all of [ %v ] received transactions\n", counter)
		return
	}
	pterm.Warning.Printf("Signed [ %v ] of [ %v ] received transactions\n", counter, len(resReceivedTransactions.Transactions))
}
