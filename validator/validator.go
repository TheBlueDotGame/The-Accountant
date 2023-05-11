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

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/webhooks"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
)

const wsConnectionTimeout = 5 * time.Second

const (
	Header = "Computantis-Validator"
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
	Index     int64       `json:"index"      bson:"index"         db:"index"`
	Block     block.Block `json:"block"      bson:"block"         db:"-"`
	Valid     bool        `json:"valid"      bson:"valid"         db:"valid"`
	CreatedAt time.Time   `json:"created_at" bson:"created_at"    db:"created_at"`
}

// StatusReadWriter provides methods to bulk read and single write validator status.
type StatusReadWriter interface {
	WriteValidatorStatus(ctx context.Context, vs *Status) error
	ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]Status, error)
}

// WebhookCreateRemovePoster provides methods to create, remove webhooks and post messages to webhooks.
type WebhookCreateRemovePoster interface {
	CreateWebhook(trigger string, h webhooks.Hook) error
	RemoveWebhook(trigger string, h webhooks.Hook) error
	PostWebhookBlock(blc *block.Block)
}

// Verifier provides methods to verify the signature of the message.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Config contains configuration of the validator.
type Config struct {
	Token     string `yaml:"token"`     // token is used to authenticate validator in the central server
	Websocket string `yaml:"websocket"` // websocket address of the central server
	Port      int    `yaml:"port"`      // port on which validator will listen for http requests
}

type app struct {
	mux       sync.RWMutex
	lastBlock block.Block
	cfg       Config
	srw       StatusReadWriter
	log       logger.Logger
	conns     map[string]socket
	ver       Verifier
	wh        WebhookCreateRemovePoster
	wallet    *wallet.Wallet
	cancel    context.CancelFunc
}

func (a *app) blocks(c *fiber.Ctx) error {
	return nil
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// Validator connects to the central server via websocket and listens for new blocks.
// It will block until the context is canceled.
func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet *wallet.Wallet) error {
	ctxx, cancel := context.WithCancel(ctx)
	a := &app{
		mux:    sync.RWMutex{},
		cfg:    cfg,
		srw:    srw,
		log:    log,
		conns:  make(map[string]socket),
		ver:    ver,
		wh:     wh,
		wallet: wallet,
		cancel: cancel,
	}

	if err := a.connectToSocket(ctxx, cfg.Websocket); err != nil {
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
	go a.pushPump(ctxx, c, address)
	a.log.Info(fmt.Sprintf("validator connected to central node on address: %s", address))

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
				return
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
					return
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

func (a *app) pushPump(ctx context.Context, conn *websocket.Conn, address string) {
	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		if err := a.disconnectFromSocket(address); err != nil {
			a.log.Error(fmt.Sprintf("validator disconnect on close, %s", err.Error()))
		}
	}()
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				a.log.Error(fmt.Sprintf("validator write msg error, %s", err.Error()))
				return
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

	router.Get("/blocks", a.blocks)

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
	case server.CommandNewTransaction:
		a.log.Warn("not implemented")
	case server.CommandSocketList:
		a.processSocketList(ctx, m.Sockets)
	default:
		a.log.Error(fmt.Sprintf("validator received unknown command, %s", m.Command))
	}
}

func (a *app) processBlock(_ context.Context, b *block.Block, remoteAddress string) {
	a.mux.Lock()
	defer a.mux.Unlock()
	err := a.validateBlock(b)
	if err != nil {
		a.log.Error(fmt.Sprintf("remote address: %s => validator received invalid:  %s ", remoteAddress, err.Error()))
	}
	a.log.Info(fmt.Sprintf("remote address: %s => last block idx: %v | new block idx %v \n", remoteAddress, a.lastBlock.Index, b.Index))
	a.lastBlock = *b

	go a.wh.PostWebhookBlock(b) // post concurrently
}

func (a *app) processSocketList(ctx context.Context, sockets []string) {
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
		a.connectToSocket(ctx, socket)
	}
	for _, socket := range remove {
		a.disconnectFromSocket(socket)
	}
}
