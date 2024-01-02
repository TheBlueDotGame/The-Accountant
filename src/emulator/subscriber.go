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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/webhooks"
)

const (
	maxTrxInBuffer           = 25
	hashesBuffLen            = 10000
	tickerSaveReadMultiplier = 100
)

const (
	header     = "SubscriberEmulator"
	apiVersion = "1.0"
)

const (
	WebHookEndpointTransaction = "/hook/transaction"
	MessageEndpoint            = "/message"
)

var ErrFailedHook = errors.New("failed to create web hook")

type hashToValidate struct {
	hash          [32]byte
	notaryNodeUrl string
}

// Message holds timestamp info.
type Message struct {
	Status      string                  `json:"status"`
	Transaction transaction.Transaction `json:"transaction"`
	Timestamp   int64                   `json:"timestamp"`
	Volts       int64                   `json:"volts"`
	MiliAmps    int64                   `json:"mili_amps"`
	Power       int64                   `json:"power"`
}

type subscriber struct {
	mux                  sync.Mutex
	lastTransactionTime  time.Time
	pub                  publisher
	allowedIssuerAddress string
	buffer               []Message
	allowdMeasurements   [2]Measurement
	ticker               time.Duration
	validateCh           chan hashToValidate
}

// RunSubscriber runs subscriber emulator.
// To stop the subscriber cancel the context.
func RunSubscriber(ctx context.Context, cancel context.CancelFunc, config Config, data []byte) error {
	defer cancel()
	var m [2]Measurement
	var err error
	err = json.Unmarshal(data, &m)
	if err != nil {
		return fmt.Errorf("cannot unmarshal data, %s", err)
	}

	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	var conn *grpc.ClientConn
	conn, err = grpc.Dial(config.ClientURL, opts)
	if err != nil {
		return fmt.Errorf("dial failed, %s", err)
	}
	defer conn.Close()
	client := protobufcompiled.NewWalletClientAPIClient(conn)
	_, err = client.WebHook(ctx, &protobufcompiled.CreateWebHook{Url: fmt.Sprintf("%s%s", config.PublicURL, WebHookEndpointTransaction)})
	if err != nil {
		return err
	}
	p := publisher{
		conn:       conn,
		client:     client,
		random:     config.Random,
		knownNodes: config.NotaryNodes,
	}

	s := subscriber{
		mux:                 sync.Mutex{},
		pub:                 p,
		lastTransactionTime: time.Now(),
		allowdMeasurements:  m,
		ticker:              time.Duration(config.TickMillisecond) * time.Millisecond * tickerSaveReadMultiplier,
		validateCh:          make(chan hashToValidate, hashesBuffLen),
	}
	defer close(s.validateCh)
	go s.runCheckSaved(ctx, 1000, uint64(config.SpicePerTransaction), config.ReceiverPublicAddr)

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second,
		WriteTimeout:  time.Second,
		ServerHeader:  header,
		AppName:       apiVersion,
		Concurrency:   16,
	})
	router.Use(cors.New())
	router.Use(recover.New())
	router.Post(WebHookEndpointTransaction, s.hookTransaction)
	router.Get(MessageEndpoint, s.messages)

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

	<-ctx.Done()
	return err
}

func (sub *subscriber) messages(c *fiber.Ctx) error {
	sub.mux.Lock()
	defer sub.mux.Unlock()
	c.Set("Content-Type", "application/json")
	return c.JSON(sub.buffer)
}

func (sub *subscriber) hookTransaction(ctx *fiber.Ctx) error {
	hookRes := make(map[string]bool)

	var res webhooks.AwaitingTransactionsMessage
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

	go sub.actOnTransactions(res.NotaryNodeURL) // make actions concurrently

	hookRes["ack"] = true
	hookRes["ok"] = true
	return ctx.JSON(hookRes)
}

func (sub *subscriber) actOnTransactions(notaryNodeURL string) {
	sub.mux.Lock()
	defer sub.mux.Unlock()

	protoTrxs, err := sub.pub.client.Waiting(context.Background(), &protobufcompiled.NotaryNode{Url: sub.pub.getRandomNodeURLFromList(notaryNodeURL)})
	if err != nil || protoTrxs == nil {
		return
	}
	if len(protoTrxs.Array) == 0 {
		return
	}

	set := make(map[[32]byte]struct{}, len(protoTrxs.Array))
	for _, trx := range protoTrxs.Array {
		set[[32]byte(trx.Hash)] = struct{}{}
	}

	if len(set) != len(protoTrxs.Array) {
		pterm.Error.Printf("Waiting trxs array contains %v trxs where %v is unique.\n", len(protoTrxs.Array), len(set))
	}

	var counter int

	for i := range protoTrxs.Array {
		notaryNodeURL = sub.pub.getRandomNodeURLFromList(notaryNodeURL)
		if err := sub.validateData(protoTrxs.Array[i].Data); err != nil {
			pterm.Warning.Printf("Trx [ %x ] data [ %s ] rejected, %s.\n", protoTrxs.Array[i].Hash, protoTrxs.Array[i].Data, err)

			go sub.pub.client.Reject(context.Background(), &protobufcompiled.TrxHash{Hash: protoTrxs.Array[i].Hash, Url: notaryNodeURL})
			go sub.sendToValidationQueue([32]byte(protoTrxs.Array[i].Hash), notaryNodeURL)

			continue
		}

		pterm.Info.Printf("Trx [ %x ] data [ %s ] accepted.\n", protoTrxs.Array[i].Hash, protoTrxs.Array[i].Data)

		go sub.pub.client.Approve(context.Background(), &protobufcompiled.TransactionApproved{Transaction: protoTrxs.Array[i], Url: notaryNodeURL})
		go sub.sendToValidationQueue([32]byte(protoTrxs.Array[i].Hash), notaryNodeURL)

		counter++
	}

	if counter == int(protoTrxs.Len) {
		pterm.Info.Printf("Signed all of [ %v ] received transactions.\n", counter)
		return
	}
	pterm.Info.Printf("Signed [ %v ] of [ %v ] received transactions.\n", counter, protoTrxs.Len)
}

func (sub *subscriber) sendToValidationQueue(h [32]byte, notaryNodeURL string) {
	notaryNodeURL = sub.pub.getRandomNodeURLFromList(notaryNodeURL)

	sub.validateCh <- hashToValidate{h, notaryNodeURL}
}

func (sub *subscriber) runCheckSaved(ctx context.Context, paymentThreshold int, spice uint64, receiver string) {
	t := time.NewTicker(sub.ticker)
	defer t.Stop()
	buffer := make([]hashToValidate, 0, hashesBuffLen)
	var accepted int
	for {
		select {
		case <-ctx.Done():
			return
		case h := <-sub.validateCh:
			if h.hash == [32]byte{} {
				return
			}
			buffer = append(buffer, h)
		case <-t.C:
			for _, h := range buffer {
				accepted += sub.getAcceptedEnergyTrx(h.hash, h.notaryNodeUrl)
			}
			buffer = make([]hashToValidate, 0, hashesBuffLen)
			if accepted > paymentThreshold {
				accepted = 0
				if err := sub.sendSpice(ctx, spice, receiver); err != nil {
					pterm.Error.Printf("Failed to send %v_spice to [ %s ]. %s \n", spice, receiver, err)
				}
			}
		}
	}
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

func (sub *subscriber) getAcceptedEnergyTrx(hash [32]byte, notaryNodeURL string) int {
	trx, err := sub.pub.client.Saved(context.Background(), &protobufcompiled.TrxHash{Hash: []byte(hash[:]), Url: notaryNodeURL})
	if err != nil {
		pterm.Error.Printf("Transaction with hash: [ %x ] not saved in DAG node URL [ %s ], %s\n", hash, notaryNodeURL, err)
		return 0
	}
	if trx == nil {
		pterm.Error.Printf("Transaction with hash: [ %x ] not saved in node URL [ %s ], transaction is nil\n", hash, notaryNodeURL)
		return 0
	}

	if trx.Spice.Currency != 0 || trx.Spice.SuplementaryCurrency != 0 {
		pterm.Info.Printf(
			"Transaction with hash [ %x ] is secured in DAG node URL [ %s ] for SPICE TRANSFER: [ %s ].\n",
			trx.Hash, notaryNodeURL, trx.Spice,
		)
		return 0
	}

	switch len(trx.ReceiverSignature) != 0 {
	case true:
		pterm.Info.Printf(
			"Transaction with hash [ %x ] is secured in DAG node URL [ %s ] and <-ACCEPTED-> by the receiver [ %s ] for data %s .\n",
			trx.Hash, notaryNodeURL, trx.ReceiverAddress, string(trx.Data),
		)
		return 1
	default:
		pterm.Warning.Printf(
			"Transaction with hash [ %x ] is secured in DAG node URL [ %s ] and <-REJECTED-> by the receiver [ %s ] for data %s .\n",
			trx.Hash, notaryNodeURL, trx.ReceiverAddress, string(trx.Data),
		)
		return 0
	}
}

func (sub *subscriber) sendSpice(ctx context.Context, spice uint64, receiver string) error {
	_, err := sub.pub.client.Issue(ctx, &protobufcompiled.IssueTrx{
		Subject:         fmt.Sprintf("Spice transfer %v for %s", spice, receiver),
		ReceiverAddress: receiver,
		Data:            []byte{},
		Spice: &protobufcompiled.Spice{
			Currency:             spice,
			SuplementaryCurrency: 0,
		},
	})
	return err
}
