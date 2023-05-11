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
}

type app struct {
	log    logger.Logger
	client client.Client
}

const (
	Alive                   = "/alive"                 // alive URL allows to check if server is alive and if sign service is of the same version.
	IssueTransaction        = "/transactions/issue"    // issue URL allows to issue transaction signed by the issuer.
	ConfirmTransaction      = "/transaction/sign"      // sign URL allows to sign transaction received by the receiver.
	GetIssuedTransactions   = "/transactions/issued"   // issued URL allows to get issued transactions for the issuer.
	GetReceivedTransactions = "/transactions/received" // received URL allows to get received transactions for the receiver.
	CreateWallet            = "/wallet/create"         // create URL allows to create new wallet.
	ReadWalletPublicAddress = "/wallet/address"        // address URL allows to read public address of the wallet.
)

// Run runs the service application that exposes the API for creating, validating and signing transactions.
// This blocks until the context is canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier,
	wrs client.WalletReadSaver, walletCreator client.NewSignValidatorCreator) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := client.NewClient(cfg.CentralNodeAddress, timeout, fw, wrs, walletCreator)
	defer c.FlushWalletFromMemory()

	if err := c.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	s := app{log: log, client: *c}

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

	router.Get(GetIssuedTransactions, s.issuedTransactions)
	router.Get(GetReceivedTransactions, s.receivedTransactions)
	router.Post(IssueTransaction, s.issueTransaction)
	router.Post(ConfirmTransaction, s.confirmReceivedTransaction)
	router.Post(CreateWallet, s.createWallet)
	router.Get(ReadWalletPublicAddress, s.readWalletPublicAddress)

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

// IssuedTransactionResponse is a response of issued transactions.
type IssuedTransactionResponse struct {
	Ok           bool                      `json:"ok"`
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
}

func (a *app) issuedTransactions(c *fiber.Ctx) error {
	transactions, err := a.client.ReadIssuedTransactions()
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(IssuedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(IssuedTransactionResponse{Ok: true, Transactions: transactions})
}

// ReceivedTransactionResponse is a response of issued transactions.
type ReceivedTransactionResponse struct {
	Ok           bool                      `json:"ok"`
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
}

func (a *app) receivedTransactions(c *fiber.Ctx) error {
	transactions, err := a.client.ReadWaitingTransactions()
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ReceivedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(ReceivedTransactionResponse{Ok: true, Transactions: transactions})
}

// CreateWalletRequest is a request to create wallet.
type CreateWalletRequest struct {
	Token string `json:"token"`
}

// CreateWalletResponse is response to create wallet.
type CreateWalletResponse struct {
	Ok  bool   `json:"ok"`
	Err string `json:"err"`
}

func (a *app) createWallet(c *fiber.Ctx) error {
	var req CreateWalletRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading create wallet request: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if err := a.client.NewWallet(req.Token); err != nil {
		err := fmt.Errorf("error creating wallet: %v", err)
		a.log.Error(err.Error())
		return c.JSON(CreateWalletResponse{Ok: false, Err: err.Error()})
	}

	if err := a.client.SaveWalletToFile(); err != nil {
		err := fmt.Errorf("error saving wallet to file: %v", err)
		a.log.Error(err.Error())
		return c.JSON(CreateWalletResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(CreateWalletResponse{Ok: true})
}

// ReadWalletPublicAddressResponse is a response to read wallet public address.
type ReadWalletPublicAddressResponse struct {
	Ok      bool   `json:"ok"`
	Err     string `json:"err"`
	Address string `json:"address"`
}

func (a *app) readWalletPublicAddress(c *fiber.Ctx) error {
	address, err := a.client.Address()
	if err != nil {
		err := fmt.Errorf("error reading wallet address: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ReadWalletPublicAddressResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(ReadWalletPublicAddressResponse{Ok: true, Address: address})
}
