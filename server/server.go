package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	"github.com/gofiber/fiber/v2"
)

const (
	ApiVersion = "1.0.0"
	Header     = "Computantis"
)

const (
	searchGroupURL      = "/search"
	addressURL          = "/address"
	blockURL            = "/block"
	transactionGroupURL = "/transaction"
	proposeURL          = "/propose"
	confirmURL          = "/confirm"
	awaitedURL          = "/awaited"
	issuedURL           = "/issued"
	validatorGroupURL   = "/validator"
	dataURL             = "/data"
	addressGroupURL     = "/address"
	createURL           = "/create"
)

const (
	AliveURL              = "/alive"                         // URL to check if server is alive and version.
	SearchAddressURL      = searchGroupURL + addressURL      // URL to search for address.
	SearchBlockURL        = searchGroupURL + blockURL        // URL to search for block that contains transaction hash.
	ProposeTransactionURL = transactionGroupURL + proposeURL // URL to propose transaction signed by the issuer.
	ConfirmTransactionURL = transactionGroupURL + confirmURL // URL to confirm transaction signed by the receiver.
	AwaitedTransactionURL = transactionGroupURL + awaitedURL // URL to get awaited transactions for the receiver.
	IssuedTransactionURL  = transactionGroupURL + issuedURL  // URL to get issued transactions for the issuer.
	DataToValidateURL     = validatorGroupURL + dataURL      // URL to get data to validate address by signing rew message.
	CreateAddressURL      = addressGroupURL + createURL      // URL to create new address.
	WsURL                 = "/ws"                            // URL to connect to websocket.
)

const queryLimit = 100

var ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")

// Repository is the interface that wraps the basic CRUD and Search methods.
// Repository should be properly indexed to allow for transaction and block hash.
// as well as address public keys to be and unique and the hash lookup should be fast.
// Repository holds the blocks and transaction that are part of the blockchain.
type Repository interface {
	Disconnect(ctx context.Context) error
	RunMigration(ctx context.Context) error
	FindAddress(ctx context.Context, search string, limit int) ([]string, error)
	CheckAddressExists(ctx context.Context, address string) (bool, error)
	WriteAddress(ctx context.Context, address string) error
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
	CheckToken(ctx context.Context, token string) (bool, error)
	InvalidateToken(ctx context.Context, token string) error
	ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
}

// Bookkeeper abstracts methods of the bookkeeping of a blockchain.
type Bookkeeper interface {
	Run(ctx context.Context)
	WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
	VerifySignature(message, signature []byte, hash [32]byte, address string) error
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

// Config contains configuration of the server.
type Config struct {
	Port int `yaml:"port"`
}

type server struct {
	repo         Repository
	bookkeeping  Bookkeeper
	randDataProv RandomDataProvideValidator
	hub          *hub
	log          logger.Logger
}

// Run initializes routing and runs the server. To stop the server cancel the context.
func Run(
	ctx context.Context, c Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator, log logger.Logger,
) error {
	var err error
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		if errx := repo.Disconnect(ctxx); errx != nil {
			err = errors.Join(err, errx)
		}
	}()

	if err := repo.RunMigration(ctxx); err != nil {
		return err
	}

	if err := validateConfig(&c); err != nil {
		return err
	}

	s := &server{
		repo:         repo,
		bookkeeping:  bookkeeping,
		randDataProv: pv,
		hub:          newHub(log),
		log:          log,
	}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  Header,
		AppName:       ApiVersion,
		Concurrency:   4096,
	})

	router.Get(AliveURL, s.alive)

	search := router.Group(searchGroupURL)
	search.Post(addressURL, s.address)
	search.Post(blockURL, s.trxInBlock)

	transaction := router.Group(transactionGroupURL)
	transaction.Post(proposeURL, s.propose)
	transaction.Post(confirmURL, s.confirm)
	transaction.Post(awaitedURL, s.awaited)
	transaction.Post(issuedURL, s.issued)

	validator := router.Group(validatorGroupURL)
	validator.Post(dataURL, s.data)

	address := router.Group(addressGroupURL)
	address.Post(createURL, s.addressCreate)

	router.Group(WsURL, s.wsWrapper)

	go func() {
		bookkeeping.Run(ctxx)
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", c.Port))
		if err != nil {
			cancel()
		}
	}()
	go s.hub.run(ctx)

	<-ctxx.Done()

	if errx := router.Shutdown(); errx != nil {
		err = errors.Join(err, errx)
	}

	return err
}

func validateConfig(c *Config) error {
	if c.Port == 0 || c.Port > 65535 {
		return ErrWrongPortSpecified
	}

	return nil
}
