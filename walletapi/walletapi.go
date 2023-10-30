package walletapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/bartossh/Computantis/helperserver"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/walletmiddleware"
)

// Config is the configuration for the notaryserver
type Config struct {
	Port          string `yaml:"port"`
	NotaryNodeURL string `yaml:"notary_node_url"`
	HelperNodeURL string `yaml:"helper_node_url"`
}

type app struct {
	log                 logger.Logger
	centralNodeClient   walletmiddleware.Client
	validatorNodeClient walletmiddleware.Client
}

const (
	day  = time.Hour * 24
	week = day * 7
)

const (
	MetricsURL              = notaryserver.MetricsURL                 // URL serves service metrics.
	Alive                   = notaryserver.AliveURL                   // URL allows to check if server is alive and if sign service is of the same version.
	Address                 = "/address"                              // URL allows to check wallet public address
	IssueTransaction        = "/transactions/issue"                   // URL allows to issue transaction signed by the issuer.
	ConfirmTransaction      = "/transaction/sign"                     // URL allows to sign transaction received by the receiver.
	RejectTransactions      = "/transactions/reject"                  // URL allows to reject transactions received by the receiver.
	GetIssuedTransactions   = "/transactions/issued"                  // URL allows to get issued transactions for the issuer.
	GetReceivedTransactions = "/transactions/received"                // URL allows to get received transactions for the receiver.
	GetRejectedTransactions = "/transactions/rejected/:offset/:limit" // URL allows to get rejected transactions with pagination.
	GetApprovedTransactions = "/transactions/approved/:offset/:limit" // URL allows to get approved transactions with pagination.
	CreateWallet            = "/wallet/create"                        // URL allows to create new wallet.
	CreateUpdateWebhook     = "/webhook/create"                       // URL allows to creatre webhook
	ReadWalletPublicAddress = "/wallet/address"                       // URL allows to read public address of the wallet.
	GetOneDayToken          = "token/day"                             // URL allows to get one day token.
	GetOneWeekToken         = "token/week"                            // URL allows to get one week token.
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
		ServerHeader:  notaryserver.Header,
		AppName:       notaryserver.ApiVersion,
		Concurrency:   1024,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: "Wallet API Node"}))

	router.Get(Alive, s.alive)
	router.Get(Address, s.address)

	router.Get(GetIssuedTransactions, s.issuedTransactions)
	router.Post(GetReceivedTransactions, s.receivedTransactions)
	router.Get(GetRejectedTransactions, s.rejectedTransactions)
	router.Get(GetApprovedTransactions, s.approvedTransactions)
	router.Post(IssueTransaction, s.issueTransaction)
	router.Post(ConfirmTransaction, s.confirmReceivedTransaction)
	router.Post(RejectTransactions, s.rejectTransactions)
	router.Post(CreateWallet, s.createWallet)
	router.Post(CreateUpdateWebhook, s.createUpdateWebHook)
	router.Get(ReadWalletPublicAddress, s.readWalletPublicAddress)
	router.Get(GetOneDayToken, s.getOneDayToken)
	router.Get(GetOneWeekToken, s.getOneWeekToken)

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
			APIVersion: notaryserver.ApiVersion,
			APIHeader:  notaryserver.Header,
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

// ConfirmTransactionRequest is a request to confirm transaction.
type ConfirmTransactionRequest struct {
	Transaction transaction.Transaction `json:"transaction"`
}

// ConfirmTransactionResponse is response of confirming transaction.
type ConfirmTransactionResponse struct {
	Err string `json:"err"`
	Ok  bool   `json:"ok"`
}

func (a *app) confirmReceivedTransaction(c *fiber.Ctx) error {
	var req ConfirmTransactionRequest
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

	if err := a.centralNodeClient.ConfirmTransaction(&req.Transaction); err != nil {
		err := fmt.Errorf("error confirming transaction: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ConfirmTransactionResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(ConfirmTransactionResponse{Ok: true})
}

// RejectTransactionsRequest is a request to reject transactions.
type RejectTransactionsRequest struct {
	Transactions []transaction.Transaction `json:"transactions"`
}

// RejectTransactionsResponse is response of rejecting transactions.
type RejectTransactionsResponse struct {
	Err        string     `json:"err"`
	TrxsHashes [][32]byte `json:"trxs_hashes"`
	Ok         bool       `json:"ok"`
}

func (a *app) rejectTransactions(c *fiber.Ctx) error {
	var req RejectTransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading data: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}

	if req.Transactions == nil {
		a.log.Error("wrong JSON format when rejecting transactions")
		return fiber.ErrBadRequest
	}

	hashes, err := a.centralNodeClient.RejectTransactions(req.Transactions)
	ok := true
	var errMsg string
	if err != nil {
		ok = false
		errMsg = err.Error()
	}

	return c.JSON(RejectTransactionsResponse{Ok: ok, TrxsHashes: hashes, Err: errMsg})
}

// IssuedTransactionResponse is a response of issued transactions.
type IssuedTransactionResponse struct {
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
	Ok           bool                      `json:"ok"`
}

func (a *app) issuedTransactions(c *fiber.Ctx) error {
	transactions, err := a.centralNodeClient.ReadIssuedTransactions()
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(IssuedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(IssuedTransactionResponse{Ok: true, Transactions: transactions})
}

// ReceivedTransactionResponse is a response of issued transactions.
type ReceivedTransactionResponse struct {
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
	Ok           bool                      `json:"ok"`
}

func (a *app) receivedTransactions(c *fiber.Ctx) error {
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

	transactions, err := a.centralNodeClient.ReadWaitingTransactions("")
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ReceivedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(ReceivedTransactionResponse{Ok: true, Transactions: transactions})
}

// RejectedTransactionResponse is a response of rejected transactions.
type RejectedTransactionResponse struct {
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
	Ok           bool                      `json:"ok"`
}

func (a *app) rejectedTransactions(c *fiber.Ctx) error {
	offset := c.Params("offset")
	limit := c.Params("limit")

	offsetNum, err := strconv.Atoi(offset)
	if err != nil {
		a.log.Error(err.Error())
		return c.JSON(RejectedTransactionResponse{Ok: false, Err: err.Error()})
	}
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		a.log.Error(err.Error())
		return c.JSON(RejectedTransactionResponse{Ok: false, Err: err.Error()})
	}

	transactions, err := a.centralNodeClient.ReadRejectedTransactions(offsetNum, limitNum)
	if err != nil {
		err := fmt.Errorf("error getting rejected transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(RejectedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(RejectedTransactionResponse{Ok: true, Transactions: transactions})
}

// ApprovedTransactionResponse is a response of approved transactions.
type ApprovedTransactionResponse struct {
	Err          string                    `json:"err"`
	Transactions []transaction.Transaction `json:"transactions"`
	Ok           bool                      `json:"ok"`
}

func (a *app) approvedTransactions(c *fiber.Ctx) error {
	offset := c.Params("offset")
	limit := c.Params("limit")

	offsetNum, err := strconv.Atoi(offset)
	if err != nil {
		a.log.Error(err.Error())
		return c.JSON(ApprovedTransactionResponse{Ok: false, Err: err.Error()})
	}
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		a.log.Error(err.Error())
		return c.JSON(ApprovedTransactionResponse{Ok: false, Err: err.Error()})
	}

	transactions, err := a.centralNodeClient.ReadApprovedTransactions(offsetNum, limitNum)
	if err != nil {
		err := fmt.Errorf("error getting rejected transactions: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ApprovedTransactionResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(ApprovedTransactionResponse{Ok: true, Transactions: transactions})
}

// CreateWalletRequest is a request to create wallet.
type CreateWalletRequest struct {
	Token string `json:"token"`
}

// CreateWalletResponse is response to create wallet.
type CreateWalletResponse struct {
	Err string `json:"err"`
	Ok  bool   `json:"ok"`
}

func (a *app) createWallet(c *fiber.Ctx) error {
	var req CreateWalletRequest
	if err := c.BodyParser(&req); err != nil {
		err := fmt.Errorf("error reading create wallet request: %v", err)
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}
	if req.Token == "" {
		a.log.Error("wrong JSON format when creating a wallet")
		return fiber.ErrBadGateway
	}

	if err := a.centralNodeClient.NewWallet(req.Token); err != nil {
		err := fmt.Errorf("error creating wallet: %v", err)
		a.log.Error(err.Error())
		return c.JSON(CreateWalletResponse{Ok: false, Err: err.Error()})
	}

	if err := a.centralNodeClient.SaveWalletToFile(); err != nil {
		err := fmt.Errorf("error saving wallet to file: %v", err)
		a.log.Error(err.Error())
		return c.JSON(CreateWalletResponse{Ok: false, Err: err.Error()})
	}

	return c.JSON(CreateWalletResponse{Ok: true})
}

// CreateWebHookRequest is a request to create a web hook
type CreateWebHookRequest struct {
	URL string `json:"url"`
}

// CreateWebhookResponse is a response describing effect of creating a web hook
type CreateWebhookResponse struct {
	Err string `json:"error"`
	Ok  bool   `json:"ok"`
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

	var res helperserver.CreateRemoveUpdateHookResponse
	if err := a.validatorNodeClient.CreateWebhook(req.URL); err != nil {
		res.Ok = false
		res.Err = err.Error()
		return c.JSON(res)
	}

	res.Ok = true
	return c.JSON(res)
}

// ReadWalletPublicAddressResponse is a response to read wallet public address.
type ReadWalletPublicAddressResponse struct {
	Err     string `json:"err"`
	Address string `json:"address"`
	Ok      bool   `json:"ok"`
}

func (a *app) readWalletPublicAddress(c *fiber.Ctx) error {
	address, err := a.centralNodeClient.Address()
	if err != nil {
		err := fmt.Errorf("error reading wallet address: %v", err)
		a.log.Error(err.Error())
		return c.JSON(ReadWalletPublicAddressResponse{Ok: false, Err: err.Error()})
	}
	return c.JSON(ReadWalletPublicAddressResponse{Ok: true, Address: address})
}

func (a *app) getOneDayToken(c *fiber.Ctx) error {
	t := time.Now().Add(day)
	token, err := a.centralNodeClient.GenerateToken(t)
	if err != nil {
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}
	return c.JSON(token)
}

func (a *app) getOneWeekToken(c *fiber.Ctx) error {
	t := time.Now().Add(week)
	token, err := a.centralNodeClient.GenerateToken(t)
	if err != nil {
		a.log.Error(err.Error())
		return errors.Join(fiber.ErrBadRequest, err)
	}
	return c.JSON(token)
}
