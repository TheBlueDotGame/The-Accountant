package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/The-Accountant/transaction"
	"github.com/gofiber/fiber/v2"
)

const (
	apiVersion = "1.0.0"
	header     = "The Accountant"
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
}

// Bookkeeper abstracts methods of the bookkeeping of a blockchain.
type Bookkeeper interface {
	Run(ctx context.Context)
	WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
	ReadAwaitedTransactionsForAddress(
		ctx context.Context,
		message, signature []byte,
		hash [32]byte,
		address string,
	) ([]transaction.Transaction, error)
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

// Config contains configuration of the server.
type Config struct {
	Port int
}

type server struct {
	repo         Repository
	bookkeeping  Bookkeeper
	randDataProv RandomDataProvideValidator
}

// Run initializes routing and runs the server. To stop the server cancel the context.
func Run(ctx context.Context, c *Config, repo Repository, bookkeeping Bookkeeper, pv RandomDataProvideValidator) error {
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

	if err := validateConfig(c); err != nil {
		return err
	}

	s := &server{
		repo:         repo,
		bookkeeping:  bookkeeping,
		randDataProv: pv,
	}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  header,
		AppName:       apiVersion,
		Concurrency:   256 * 2048,
	})

	router.Get("/alive", s.alive)

	search := router.Group("/search")
	search.Post("/address", s.address)
	search.Post("/block", s.trxInBlock)

	transaction := router.Group("/transaction")
	transaction.Post("/propose", s.propose)
	transaction.Post("/confirm", s.confirm)
	transaction.Post("/awaited", s.awaited)
	transaction.Post("/data", s.data)

	address := router.Group("/address")
	address.Post("/create", s.addressCreate)

	go func() {
		bookkeeping.Run(ctxx)
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", c.Port))
		if err != nil {
			cancel()
		}
	}()

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
