package validator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/repo"
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

// StatusReadWriter provides methods to bulk read and single write validator status.
type StatusReadWriter interface {
	WriteValidatorStatus(ctx context.Context, vs *repo.ValidatorStatus) error
	ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]repo.ValidatorStatus, error)
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
	lastBlock block.Block
	cfg       Config
	srw       StatusReadWriter
	log       logger.Logger
	conn      *websocket.Conn
	ver       Verifier
	wh        WebhookCreateRemovePoster
}

func (a *app) blocks(c *fiber.Ctx) error {
	return nil
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// Validator connects to the central server via websocket and listens for new blocks.
// It will block until the context is canceled.
func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger, ver Verifier, wh WebhookCreateRemovePoster, wallet *wallet.Wallet) error {
	hash, signature := wallet.Sign([]byte(cfg.Token))

	header := make(http.Header)
	header.Add("Token", cfg.Token)
	header.Add("Address", wallet.Address())
	header.Add("Signature", hex.EncodeToString(signature[:]))
	header.Add("Hash", hex.EncodeToString(hash[:]))

	ctxTimeout, cancel := context.WithTimeout(ctx, wsConnectionTimeout)
	c, _, err := websocket.DefaultDialer.DialContext(ctxTimeout, cfg.Websocket, header)
	cancel()
	if err != nil {
		return err
	}
	defer c.Close()

	ctxx, cancel := context.WithCancel(ctx)

	a := &app{
		cfg:  cfg,
		srw:  srw,
		log:  log,
		conn: c,
		ver:  ver,
		wh:   wh,
	}

	go a.pullPump(ctxx, cancel)
	go a.pushPump(ctxx, cancel)

	return a.runServer(ctxx, cancel)
}

func (a app) pullPump(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	defer cancel()
listener:
	for {
		select {
		case <-ctx.Done():
			break listener
		case <-ticker.C:
		}
		msgType, raw, err := a.conn.ReadMessage()
		if err != nil {
			a.log.Error(fmt.Sprintf("validator read msg error, %s", err.Error()))
			continue
		}
		switch msgType {
		case websocket.PingMessage, websocket.PongMessage:
			continue
		default:
			var msg server.Message
			if err := json.Unmarshal(raw, &msg); err != nil {
				a.log.Error(fmt.Sprintf("validator unmarshal msg error, %s", err.Error()))
				continue
			}
			if msg.Error != "" {
				a.log.Error(fmt.Sprintf("validator msg error, %s", msg.Error))
				continue
			}
			a.processMessage(&msg)
		}
	}
}

func (a app) pushPump(ctx context.Context, cancel context.CancelFunc) {
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		defer cancel()
	connectionPingCloser:
		for {
			select {
			case <-ticker.C:
				err := a.conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					a.log.Error(fmt.Sprintf("validator write msg error, %s", err.Error()))
					break connectionPingCloser
				}
			case <-ctx.Done():
				a.log.Info("validator closing connection")
				err := a.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					a.log.Error(fmt.Sprintf("validator write closing msg error, %s", err.Error()))
				}
				break connectionPingCloser
			}
		}
	}()
}

func (a app) runServer(ctx context.Context, cancel context.CancelFunc) error {
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

func (a app) processMessage(m *server.Message) {
	switch m.Command {
	case server.CommandNewBlock:
		a.processBlock(&m.Block)
	case server.CommandNewTransaction:
	default:
		a.log.Error(fmt.Sprintf("validator received unknown command, %s", m.Command))
	}
}

func (a app) processBlock(b *block.Block) {
	err := a.validateBlock(b)

	switch err {
	case nil:
		a.lastBlock = *b
	default:
		// TODO: trigger webhook alert about invalid block
		// TODO: implement strategy to handle invalid blocks
		a.log.Error(fmt.Sprintf("validator received invalid block, %s", err.Error()))
	}

	go a.wh.PostWebhookBlock(b) // post concurrently
}
