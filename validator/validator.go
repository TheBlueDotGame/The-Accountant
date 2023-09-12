package validator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/httpclient"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/webhooks"
)

const wsConnectionTimeout = 5 * time.Second

const (
	Header = "Computantis-Validator"
)

const (
	AliveURL           = server.AliveURL          // URL to check is service alive
	MetricsURL         = server.MetricsURL        // URL to serve service metrics over http.
	DataEndpointURL    = server.DataToValidateURL // URL to serve data to sign to prove identity.
	BloclHookURL       = "/block/new"             // URL allows to create block hook.
	TransactionHookURL = "/transaction/new"       // URL allows to create transaction hook.
)

var (
	ErrProofBlockIsInvalid    = fmt.Errorf("block proof is invalid")
	ErrBlockIndexIsInvalid    = fmt.Errorf("block index is invalid")
	ErrBlockPrevHashIsInvalid = fmt.Errorf("block previous hash is invalid")
)

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
	PostWebhookNewTransaction(publicAddresses []string)
}

// Verifier provides methods to verify the signature of the message.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Config contains configuration of the validator.
type Config struct {
	Token              string `yaml:"token"`                // token is used to authenticate validator in the central server
	CentralNodeAddress string `yaml:"central_node_address"` // address of the central server
	Port               int    `yaml:"port"`                 // port on which validator will listen for http requests
}

type app struct {
	conns        map[string]socket
	cancel       context.CancelFunc
	srw          StatusReadWriter
	ver          Verifier
	wh           WebhookCreateRemovePoster
	randDataProv server.RandomDataProvideValidator
	log          logger.Logger
	wallet       *wallet.Wallet
	mux          sync.RWMutex
	cfg          Config
	lastBlock    block.Block
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// Validator connects to the central server via websocket and listens for new blocks.
// It will block until the context is canceled.
func Run(
	ctx context.Context, cfg Config,
	srw StatusReadWriter, log logger.Logger,
	ver Verifier, wh WebhookCreateRemovePoster,
	wallet *wallet.Wallet, rdp server.RandomDataProvideValidator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	a := &app{
		mux:          sync.RWMutex{},
		cfg:          cfg,
		srw:          srw,
		log:          log,
		conns:        make(map[string]socket),
		ver:          ver,
		wh:           wh,
		wallet:       wallet,
		randDataProv: rdp,
		cancel:       cancel,
	}

	log.Info(fmt.Sprintf("validator [ %s ] is starting on port: %d", a.wallet.Address(), cfg.Port))

	deadline, ok := ctxx.Deadline()
	timeout := time.Until(deadline)
	if !ok {
		timeout = time.Second * 5
	}

	var res server.DiscoverResponse
	if err := httpclient.MakeGet(timeout, fmt.Sprintf("%s%s", cfg.CentralNodeAddress, server.DiscoverCentralNodesURL), &res); err != nil {
		cancel()
		return err
	}

	if err := a.processSocketList(ctxx, res.Sockets); err != nil {
		cancel()
		return err
	}

	return a.runServer(ctxx, cancel)
}

func (a *app) connectToSocket(ctx context.Context, address string) error {
	a.mux.RLock()
	if _, ok := a.conns[address]; ok {
		a.mux.RUnlock()
		return nil
	}
	a.mux.RUnlock()

	hash, signature := a.wallet.Sign([]byte(a.cfg.Token))
	header := make(http.Header)
	header.Add("Token", a.cfg.Token)
	header.Add("Address", a.wallet.Address())
	header.Add("Signature", hex.EncodeToString(signature[:]))
	header.Add("Hash", hex.EncodeToString(hash[:]))

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, wsConnectionTimeout)
	defer cancelTimeout()
	c, _, err := websocket.DefaultDialer.DialContext(ctxTimeout, address, header)
	if err != nil {
		return err
	}

	ctxx, cancelx := context.WithCancel(ctx)

	a.mux.Lock()
	defer a.mux.Unlock()

	a.conns[address] = socket{
		conn:   c,
		cancel: cancelx,
	}

	go a.pullPump(ctxx, c, address)
	go a.pushPump(ctxx, cancelx, c, address)
	a.log.Info(fmt.Sprintf("validator [ %s ] connected to central node on address: %s", a.wallet.Address(), address))

	return nil
}

func (a *app) disconnectFromSocket(address string) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	conn, ok := a.conns[address]
	if !ok {
		if len(a.conns) == 0 {
			a.cancel()
			return errors.New("no connections left")
		}
		return nil
	}
	err := conn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		a.log.Error(fmt.Sprintf("validator write closing msg error, %s", err.Error()))
	}
	conn.cancel()
	delete(a.conns, address)
	if len(a.conns) == 0 {
		a.cancel()
		return errors.New("no connections left")
	}
	a.log.Info(fmt.Sprintf("disconnected from %s", address))

	return nil
}

func (a *app) pullPump(ctx context.Context, conn *websocket.Conn, address string) {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer func() {
		if err := a.disconnectFromSocket(address); err != nil {
			a.log.Error(fmt.Sprintf("validator disconnect on close, %s", err.Error()))
		}
	}()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msgType, raw, err := conn.ReadMessage()
			if err != nil {
				a.log.Error(fmt.Sprintf("validator read msg error, %s", err.Error()))
				continue
			}
			switch msgType {
			case websocket.PingMessage, websocket.PongMessage:
				continue
			case websocket.CloseMessage:
				return
			default:
				var msg server.Message
				if err := json.Unmarshal(raw, &msg); err != nil {
					a.log.Error(fmt.Sprintf("validator unmarshal msg error, %s", err.Error()))
					continue
				}
				if msg.Error != "" {
					a.log.Info(fmt.Sprintf("validator msg error, %s", msg.Error))
					continue
				}
				a.processMessage(ctx, &msg, conn.RemoteAddr().String())
			}
		}
	}
}

func (a *app) pushPump(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, address string) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				a.log.Error(fmt.Sprintf("validator write msg error, %s", err.Error()))
				cancel()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *app) runServer(ctx context.Context, cancel context.CancelFunc) error {
	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  server.Header,
		AppName:       server.ApiVersion,
		Concurrency:   4096,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: "Validator Node"}))
	router.Get(AliveURL, a.alive)

	router.Post(DataEndpointURL, a.data)
	router.Post(BloclHookURL, a.blocks)
	router.Post(TransactionHookURL, a.transactions)

	go func() {
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", a.cfg.Port))
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

func (a *app) processMessage(ctx context.Context, m *server.Message, remoteAddress string) {
	switch m.Command {
	case server.CommandNewBlock:
		a.processBlock(ctx, &m.Block, remoteAddress)
	case server.CommandNewTrxIssued:
		a.processNewTrxIssued(ctx, m.IssuedTrxForAddresses)
	case server.CommandSocketList:
		if err := a.processSocketList(ctx, m.Sockets); err != nil {
			a.log.Error(err.Error())
		}
	default:
		a.log.Error(fmt.Sprintf("validator received unknown command, %s", m.Command))
	}
}

func (a *app) processBlock(_ context.Context, b *block.Block, remoteAddress string) {
	lastBlockIndex := a.lastBlock.Index
	err := a.validateBlock(b)
	if err != nil {
		a.log.Error(fmt.Sprintf("remote node address [ %s ], %s ", remoteAddress, err.Error()))
	}
	a.log.Info(fmt.Sprintf("block from [ %s ] :: last idx [ %v ] :: new idx [ %v ] \n", remoteAddress, lastBlockIndex, b.Index))
	go a.wh.PostWebhookBlock(b) // post concurrently
}

func (a *app) processNewTrxIssued(_ context.Context, receivers []string) {
	go a.wh.PostWebhookNewTransaction(receivers) // post concurrently
}

func (a *app) processSocketList(ctx context.Context, sockets []string) error {
	var connect, remove []string
	a.mux.RLock()
	uniqueSockets := make(map[string]struct{})
	for _, socket := range sockets {
		if _, ok := a.conns[socket]; !ok {
			connect = append(connect, socket)
		}
		uniqueSockets[socket] = struct{}{}
	}
	for socket := range a.conns {
		if _, ok := uniqueSockets[socket]; !ok {
			remove = append(remove, socket)
		}
	}
	a.mux.RUnlock()

	for _, socket := range connect {
		if err := a.connectToSocket(ctx, socket); err != nil {
			return err
		}
	}
	for _, socket := range remove {
		if err := a.disconnectFromSocket(socket); err != nil {
			return err
		}
	}

	return nil
}

func (s *app) alive(c *fiber.Ctx) error {
	return c.JSON(
		server.AliveResponse{
			Alive:      true,
			APIVersion: server.ApiVersion,
			APIHeader:  server.Header,
		})
}

func (a *app) data(c *fiber.Ctx) error {
	var req server.DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("/data endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	d := a.randDataProv.ProvideData(req.Address)
	return c.JSON(server.DataToSignResponse{Data: d})
}

func (a *app) blocks(c *fiber.Ctx) error {
	var req CreateRemoveUpdateHookRequest
	if err := c.BodyParser(&req); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, failed to parse request body: %s", BloclHookURL, err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := a.randDataProv.ValidateData(req.Address, req.Data); !ok {
		a.log.Error("%s endpoint, corrupted data")
		return fiber.ErrForbidden
	}

	buf := make([]byte, 0, len(req.Data)+len(req.URL))
	buf = append(buf, append(req.Data, []byte(req.URL)...)...)

	if err := a.ver.Verify(buf, req.Signature, [32]byte(req.Digest), req.Address); err != nil {
		a.log.Error(fmt.Sprintf("%s endpoint, invalid signature: %s", BloclHookURL, err.Error()))
		return fiber.ErrForbidden
	}

	h := webhooks.Hook{
		URL:   req.URL,
		Token: string(req.Data),
	}
	if err := a.wh.CreateWebhook(webhooks.TriggerNewBlock, req.Address, h); err != nil {
		a.log.Error(fmt.Sprintf("%s failed to create webhook: %s", BloclHookURL, err.Error()))
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
