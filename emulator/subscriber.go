package emulator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pterm/pterm"

	"github.com/bartossh/Computantis/httpclient"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/walletapi"
	"github.com/bartossh/Computantis/webhooks"
)

const maxTrxInBuffer = 25

const (
	header     = "SubscriberEmulator"
	apiVersion = "1.0"
)

const (
	WebHookEndpointTransaction = "/hook/transaction"
	WebHookEndpointBlock       = "hook/block"
	MessageEndpoint            = "/message"
)

var ErrFailedHook = errors.New("failed to create web hook")

// Message holds timestamp info.
type Message struct {
	Status      string                  `json:"status"`
	Transaction transaction.Transaction `json:"transaction"`
	Timestamp   int64                   `json:"timestamp"`
	Volts       int                     `json:"volts"`
	MiliAmps    int                     `json:"mili_amps"`
	Power       int                     `json:"power"`
}

type subscriber struct {
	buffer               []Message
	mux                  sync.Mutex
	lastTransactionTime  time.Time
	allowedIssuerAddress string
	pub                  publisher
	allowdMeasurements   [2]Measurement
}

// RunSubscriber runs subscriber emulator.
// To stop the subscriber cancel the context.
func RunSubscriber(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error {
	defer cancel()
	var m [2]Measurement
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("cannot unmarshal data, %s", err)
	}

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
		allowdMeasurements:  m,
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
	router.Use(cors.New())
	router.Use(recover.New())
	router.Post(WebHookEndpointTransaction, s.hookTransaction)
	router.Post(WebHookEndpointTransaction, s.hookTransaction)
	router.Get(MessageEndpoint, s.messages)

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

	var resT walletapi.CreateWebhookResponse
	reqT := walletapi.CreateWebHookRequest{
		URL: fmt.Sprintf("%s%s", config.PublicURL, WebHookEndpointTransaction),
	}
	url := fmt.Sprintf("%s%s", s.pub.clientURL, walletapi.CreateUpdateWebhook)
	if err := httpclient.MakePost(s.pub.timeout, url, reqT, &resT); err != nil {
		return err
	}

	if !resT.Ok {
		if resT.Err != "" {
			return errors.New(resT.Err)
		}
		return errors.New("unexpected error when creating the webkhook")
	}

	<-ctx.Done()
	return err
}

func (sub *subscriber) messages(c *fiber.Ctx) error {
	sub.mux.Lock()
	defer sub.mux.Unlock()
	c.Set("Content-Type", "application/json")
	return c.JSON(sub.buffer)
}

func (sub *subscriber) hookBlock(ctx *fiber.Ctx) error {
	hookRes := make(map[string]bool)

	var res webhooks.WebHookNewBlockMessage
	if err := ctx.BodyParser(&res); err != nil {
		pterm.Error.Println(err.Error())
		hookRes["ack"] = false
		hookRes["ok"] = false
		return ctx.JSON(hookRes)
	}

	sub.mux.Lock()
	defer sub.mux.Unlock()

	hookRes["ack"] = true
	hookRes["ok"] = true
	return ctx.JSON(hookRes)
}

func (sub *subscriber) hookTransaction(ctx *fiber.Ctx) error {
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
		pterm.Error.Println("Time is corrupted.")
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

	for _, trx := range resReceivedTransactions.Transactions {
		if err := sub.validateData(trx.Data); err != nil {
			pterm.Warning.Printf("Trx [ %x ] data [ %s ] rejected, %s.\n", trx.Hash[:], trx.Data, err)

			rejectReq := walletapi.RejectTransactionsRequest{
				Transactions: []transaction.Transaction{trx},
			}
			var rejectRes walletapi.RejectedTransactionResponse
			url := fmt.Sprintf("%s%s", sub.pub.clientURL, walletapi.RejectTransactions)
			if err := httpclient.MakePost(sub.pub.timeout, url, rejectReq, &rejectRes); err != nil {
				pterm.Error.Printf("Transaction faild to be rejected due to: %s.\n", err)
			}

			sub.appendToBuffer("rejected", trx)
			continue
		}

		pterm.Info.Printf("Trx [ %x ] data [ %s ] accepted.\n", trx.Hash[:], string(trx.Data))

		confirmReq := walletapi.ConfirmTransactionRequest{
			Transaction: trx,
		}

		url := fmt.Sprintf("%s%s", sub.pub.clientURL, walletapi.ConfirmTransaction)
		if err := httpclient.MakePost(sub.pub.timeout, url, confirmReq, &confirmRes); err != nil {
			pterm.Error.Printf("Transaction failed to be signed due to: %s.\n", err)
			continue
		}

		if !confirmRes.Ok {
			if confirmRes.Err != "" {
				pterm.Error.Printf("Transaction cannot be signed, %.s\n", confirmRes.Err)
				continue
			}
			pterm.Error.Println("Transaction cannot be signed.")
			continue
		}

		sub.appendToBuffer("accepted", trx)

		counter++
	}

	if counter == len(resReceivedTransactions.Transactions) {
		pterm.Info.Printf("Signed all of [ %v ] received transactions.\n", counter)
		return
	}
	pterm.Warning.Printf("Signed [ %v ] of [ %v ] received transactions.\n", counter, len(resReceivedTransactions.Transactions))
}

func (sub *subscriber) validateData(data []byte) error {
	var m Measurement
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	dMamps := m.Mamps > sub.allowdMeasurements[1].Mamps || m.Mamps < sub.allowdMeasurements[0].Mamps
	dPower := m.Power > sub.allowdMeasurements[1].Power || m.Power < sub.allowdMeasurements[0].Power
	dVolts := m.Volts > sub.allowdMeasurements[1].Volts || m.Volts < sub.allowdMeasurements[0].Volts

	if dMamps || dPower || dVolts {
		return errors.New("value out of range")
	}
	return nil
}

func (sub *subscriber) appendToBuffer(status string, trx transaction.Transaction) {
	var m Measurement
	if err := json.Unmarshal(trx.Data, &m); err != nil {
		pterm.Error.Printf("Failed to marshal mesuremet due to: %s", err)
		return
	}
	sub.buffer = append(sub.buffer, Message{
		Timestamp:   time.Now().UnixNano(),
		Status:      status,
		Transaction: trx,
		Volts:       m.Volts,
		MiliAmps:    m.Mamps,
		Power:       m.Power,
	})

	if len(sub.buffer) > maxTrxInBuffer {
		sub.buffer = sub.buffer[len(sub.buffer)-maxTrxInBuffer:]
	}
}
