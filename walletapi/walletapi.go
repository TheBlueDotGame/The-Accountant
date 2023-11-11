package walletapi

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/versioning"
	"github.com/bartossh/Computantis/walletmiddleware"
	"github.com/bartossh/Computantis/webhooksserver"
)

// Config is the configuration for the notaryserver
type Config struct {
	Port          string `yaml:"port"`
	NotaryNodeURL string `yaml:"notary_node_url"`
	HelperNodeURL string `yaml:"helper_node_url"`
}

type app struct {
	protobufcompiled.UnimplementedWalletClientAPIServer
	log                 logger.Logger
	centralNodeClient   walletmiddleware.Client
	validatorNodeClient walletmiddleware.Client
}

const (
	MetricsURL              = notaryserver.MetricsURL  // URL serves service metrics.
	Alive                   = notaryserver.AliveURL    // URL allows to check if server is alive and if sign service is of the same version.
	Address                 = "/address"               // URL allows to validate address and API version
	IssueTransaction        = "/transactions/issue"    // URL allows to issue transaction signed by the issuer.
	ConfirmTransaction      = "/transaction/sign"      // URL allows to sign transaction received by the receiver.
	RejectTransactions      = "/transactions/reject"   // URL allows to reject transactions received by the receiver.
	GetWaitingTransactions  = "/transactions/waiting"  // URL allows to get issued transactions for the issuer.
	GetApprovedTransactions = "/transactions/approved" // URL allows to get approved transactions with pagination.
	CreateUpdateWebhook     = "/webhook/create"        // URL allows to create webhook
)

// Run runs the service application that exposes the API for creating, validating and signing transactions.
// This blocks until the context is canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier,
	wrs walletmiddleware.WalletReadSaver, walletCreator walletmiddleware.NewSignValidatorCreator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := walletmiddleware.NewClient(cfg.NotaryNodeURL, timeout, fw, wrs, walletCreator)
	defer c.FlushWalletFromMemory()

	if err := c.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	v := walletmiddleware.NewClient(cfg.HelperNodeURL, timeout, fw, wrs, walletCreator)
	defer v.FlushWalletFromMemory()

	if err := v.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	s := app{log: log, centralNodeClient: *c, validatorNodeClient: *v}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  versioning.Header,
		AppName:       versioning.ApiVersion,
		Concurrency:   1024,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: "Wallet API Node"}))

	router.Get(Alive, s.alive)
	router.Get(Address, s.address)

	router.Get(GetWaitingTransactions, s.waitingTransactions)
	router.Get(GetApprovedTransactions, s.approvedTransactions)
	router.Post(IssueTransaction, s.issueTransaction)
	router.Post(ConfirmTransaction, s.confirmReceivedTransaction)
	router.Post(RejectTransactions, s.rejectTransactions)
	router.Post(CreateUpdateWebhook, s.createUpdateWebHook)

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

// AliveResponse is containing server alive data such as ApiVersion and APIHeader.
type AliveResponse notaryserver.AliveResponse

func (a *app) alive(c *fiber.Ctx) error {
	if err := a.centralNodeClient.ValidateApiVersion(); err != nil {
		return errors.Join(fiber.ErrConflict, err)
	}
	return c.JSON(
		AliveResponse{
			Alive:      true,
			APIVersion: versioning.ApiVersion,
			APIHeader:  versioning.Header,
		})
}

// AddressResponse is wallet public address response.
type AddressResponse struct {
	Address string `json:"address"`
}

func (a *app) address(c *fiber.Ctx) error {
	if err := a.centralNodeClient.ValidateApiVersion(); err != nil {
		return errors.Join(fiber.ErrConflict, err)
	}

	addr, err := a.centralNodeClient.Address()
	if err != nil {
		return errors.Join(fiber.ErrNotFound, err)
	}
	return c.JSON(
		AddressResponse{
			Address: addr,
		})
}

// IssueTransactionRequest is a request message that contains data and subject of the transaction to be issued.
type IssueTransactionRequest struct {
	ReceiverAddress       string `json:"receiver_address"`
	Subject               string `json:"subject"`
	Data                  []byte `json:"data"`
	Curency               uint64 `json:"currency"`
	SupplementaryCurrency uint64 `json:"supplementary_currency"`
}

// IssueTransactionResponse is response to issued transaction.
type IssueTransactionResponse struct {
	Err string `json:"err"`
	Ok  bool   `json:"ok"`
}

func (a *app) issueTransaction(c *fiber.Ctx) error {
	var req IssueTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if req.Data == nil || req.Subject == "" || req.ReceiverAddress == "" {
		a.log.Error("wrong JSON format when issuing transaction")
		return fiber.ErrBadRequest
	}

	spc := spice.New(req.Curency, req.SupplementaryCurrency)

	if err := a.centralNodeClient.ProposeTransaction(req.ReceiverAddress, req.Subject, spc, req.Data); err != nil {
		err := fmt.Errorf("error proposing transaction: %v", err)
		a.log.Error(err.Error())
		return c.JSON(IssueTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(IssueTransactionResponse{Ok: true})
}

// TransactionsRequest is a request for group of transactions.
type TransactionsRequest struct {
	NotaryNodeURL string                    `json:"notary_node_url"`
	Transactions  []transaction.Transaction `json:"transactions"`
}

// TransactionsHashesResponse is response of group of transactions hashes.
type TransactionsHashesResponse struct {
	Err        string     `json:"err"`
	TrxsHashes [][32]byte `json:"trxs_hashes,omitempty"`
	Ok         bool       `json:"ok"`
}

func (a *app) rejectTransactions(c *fiber.Ctx) error {
	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if req.Transactions == nil {
		a.log.Error("wrong JSON format when rejecting transactions")
		return fiber.ErrBadRequest
	}

	hashes, err := a.centralNodeClient.RejectTransactions(req.NotaryNodeURL, req.Transactions)
	ok := true
	var errMsg string
	if err != nil {
		ok = false
		errMsg = err.Error()
	}

	return c.JSON(TransactionsHashesResponse{Ok: ok, TrxsHashes: hashes, Err: errMsg})
}

// TransactionRequest is a request to confirm transaction.
type TransactionRequest struct {
	NotaryNodeURL string                  `json:"notary_node_url"`
	Transaction   transaction.Transaction `json:"transaction"`
}

// TransactionResponse is response of confirming transaction.
type TransactionResponse struct {
	Err string `json:"err"`
	Ok  bool   `json:"ok"`
}

func (a *app) confirmReceivedTransaction(c *fiber.Ctx) error {
	var req TransactionRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if req.Transaction.ReceiverAddress == "" || req.Transaction.Subject == "" ||
		req.Transaction.Data == nil || req.Transaction.CreatedAt.IsZero() || req.Transaction.IssuerAddress == "" ||
		req.Transaction.IssuerSignature == nil || req.Transaction.Hash == [32]byte{} {
		a.log.Error("wrong JSON format to confirm transaction")
		return fiber.ErrBadRequest
	}

	if err := a.centralNodeClient.ConfirmTransaction(req.NotaryNodeURL, &req.Transaction); err != nil {
		err := fmt.Errorf("error confirming transaction: %v", err)
		a.log.Error(err.Error())
		return c.JSON(TransactionResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(TransactionResponse{Ok: true})
}

// TransactionsResponse is a response containing transactions, success indicator and error.
type TransactionsResponse struct {
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
	Ok           bool                      `json:"ok"`
}

func (a *app) waitingTransactions(c *fiber.Ctx) error {
	var notaryNodeURL string
	if err := c.BodyParser(&notaryNodeURL); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}
	if notaryNodeURL == "" {
		a.log.Error("wrong message format, notary node URL is empty in the message")
		return fiber.ErrBadRequest
	}
	if _, err := url.Parse(notaryNodeURL); err != nil {
		a.log.Error(fmt.Sprintf("wrong URL format, notary node URL cannot be parsed, %s", err))
		return fiber.ErrBadRequest
	}

	transactions, err := a.centralNodeClient.ReadWaitingTransactions(notaryNodeURL)
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(TransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(TransactionsResponse{Ok: true, Transactions: transactions})
}

func (a *app) approvedTransactions(c *fiber.Ctx) error {
	transactions, err := a.centralNodeClient.ReadApprovedTransactions()
	if err != nil {
		err := fmt.Errorf("error getting rejected transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(TransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(TransactionsResponse{Ok: true, Transactions: transactions})
}

// CreateWebHookRequest is a request to create a web hook
type CreateWebHookRequest struct {
	URL string `json:"url"`
}

func (a *app) createUpdateWebHook(c *fiber.Ctx) error {
	var req CreateWebHookRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading create, update webhook request: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if req.URL == "" {
		a.log.Error("wrong JSON format when creating a web hook")
		return fiber.ErrBadGateway
	}

	var res webhooksserver.CreateRemoveUpdateHookResponse
	if err := a.validatorNodeClient.CreateWebhook(req.URL); err != nil {
		res.Ok = false
		res.Err = err.Error()
		return c.JSON(res)
	}

	res.Ok = true
	return c.JSON(res)
}
