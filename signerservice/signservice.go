package signerservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/client"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/transaction"
	"github.com/gofiber/fiber/v2"
)

// Config is the configuration for the server
type Config struct {
	Port               string `yaml:"port"`
	CentralNodeAddress string `yaml:"central_node_address"`
	Token              string `yaml:"token"`
}

type app struct {
	log    logger.Logger
	client client.Client
	token  string
}

const (
	Alive              = "/alive" // alive URL allows to check if server is alive and if sign service is of the same version.
	IssueTransaction   = "/issue" // issue URL allows to issue transaction signed by the issuer.
	ConfirmTransaction = "/sign"  // sign URL allows to sign transaction received by the receiver.
)

// Run runs the service application that exposes the API for creating, validating and signing transactions.
// This blocks until the context is canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier,
	wrs client.WalletReadSaver, walletCreator client.NewSignValidatorCreator) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := client.NewClient(cfg.CentralNodeAddress, timeout, fw, wrs, walletCreator)

	s := app{log: log, client: *c, token: cfg.Token}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  server.Header,
		AppName:       server.ApiVersion,
		Concurrency:   1024,
	})

	router.Get(Alive, s.alive)

	router.Post(IssueTransaction, s.issueTransaction)
	router.Post(ConfirmTransaction, s.confirmReceivedTransaction)

	var err error
	go func() {
		err = router.Listen(fmt.Sprintf("0.0.0.0:%v", cfg.Port))
		if err != nil {
			cancel()
		}
	}()

	<-ctxx.Done()

	if err = router.Shutdown(); err != nil {
		return err
	}

	return err
}

func (a *app) alive(c *fiber.Ctx) error {
	if err := a.client.ValidateApiVersion(); err != nil {
		return errors.Join(fiber.ErrConflict, err)
	}
	return c.JSON(
		server.AliveResponse{
			Alive:      true,
			APIVersion: server.ApiVersion,
			APIHeader:  server.Header,
		})
}

// IssueTransactionRequest is a request message that contains data and subject of the transaction to be issued.
type IssueTransactionRequest struct {
	ReceiverAddress string `json:"receiver_address"`
	Subject         string `json:"subject"`
	Data            []byte `json:"data"`
}

// IssueTransactionResponse is response to issued transaction.
type IssueTransactionResponse struct {
	Ok  bool   `json:"ok"`
	Err string `json:"err"`
}

func (a *app) issueTransaction(c *fiber.Ctx) error {
	var req IssueTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if err := a.client.ProposeTransaction(req.ReceiverAddress, req.Subject, req.Data); err != nil {
		err := fmt.Errorf("error proposing transaction: %v", err)
		a.log.Error(err.Error())
		return c.JSON(IssueTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(IssueTransactionResponse{Ok: true})
}

// ValidateTransactionRequest is a request to validate transaction.
type ConfirmTransactionRequest struct {
	Transaction transaction.Transaction `json:"transaction"`
}

// ConfirmTransactionResponse is response to validate transaction.
type ConfirmTransactionResponse struct {
	Ok  bool   `json:"ok"`
	Err string `json:"err"`
}

func (a *app) confirmReceivedTransaction(c *fiber.Ctx) error {
	var req ConfirmTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if err := a.client.ConfirmTransaction(&req.Transaction); err != nil {
		err := fmt.Errorf("error confirming transaction: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ConfirmTransactionResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(ConfirmTransactionResponse{Ok: true})
}
