package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/repo"
	"github.com/bartossh/Computantis/server"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
)

const (
	Header = "Computantis-Validator"
)

type StatusReadWriter interface {
	WriteValidatorStatus(ctx context.Context, vs *repo.ValidatorStatus) error
	ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]repo.ValidatorStatus, error)
}

// Config contains configuration of the validator.
type Config struct {
	Token      string `yaml:"token"`
	Address    string `yaml:"address"`
	Websocket  string `yaml:"websocket"`
	Port       int    `yaml:"port"`
	WalletPath string `yaml:"wallet_path"`
}

type app struct {
	srw StatusReadWriter
	log logger.Logger
}

func (a *app) blocks(c *fiber.Ctx) error {
	return nil
}

// Run initializes routing and runs the validator. To stop the validator cancel the context.
// Validator connects to the central server via websocket and listens for new blocks.
func Run(ctx context.Context, cfg Config, srw StatusReadWriter, log logger.Logger) error {
	header := make(http.Header)
	header.Add("token", cfg.Token)
	header.Add("address", cfg.Address)
	c, _, err := websocket.DefaultDialer.DialContext(ctx, cfg.Websocket, header)
	if err != nil {
		return err
	}
	defer c.Close()

	ctxx, cancel := context.WithCancel(ctx)

	go pullPump(ctxx, cancel, c, log, srw)
	go pushPump(ctxx, cancel, c, log)

	return runServer(ctxx, cancel, cfg, log, srw)
}

func pullPump(ctx context.Context, cancel context.CancelFunc, c *websocket.Conn, log logger.Logger, srw StatusReadWriter) {
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
		msgType, raw, err := c.ReadMessage()
		if err != nil {
			log.Error(fmt.Sprintf("validator read msg error, %s", err.Error()))
			continue
		}
		switch msgType {
		case websocket.PingMessage, websocket.PongMessage:
			continue
		default:
			var msg server.Message
			if err := json.Unmarshal(raw, &msg); err != nil {
				log.Error(fmt.Sprintf("validator unmarshal msg error, %s", err.Error()))
				continue
			}
			// TODO: validate block and save effect
		}

	}
}

func pushPump(ctx context.Context, cancel context.CancelFunc, c *websocket.Conn, log logger.Logger) {
	go func() {
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		defer cancel()
	connectionPingCloser:
		for {
			select {
			case <-ticker.C:
				err := c.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					log.Error(fmt.Sprintf("validator write msg error, %s", err.Error()))
					break connectionPingCloser
				}
			case <-ctx.Done():
				log.Info("validator closing connection")
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Error(fmt.Sprintf("validator write closing msg error, %s", err.Error()))
				}
				break connectionPingCloser
			}
		}
	}()
}

func runServer(ctx context.Context, cancel context.CancelFunc, cfg Config, log logger.Logger, srw StatusReadWriter) error {
	a := app{
		srw: srw,
		log: log,
	}

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
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", cfg.Port))
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
