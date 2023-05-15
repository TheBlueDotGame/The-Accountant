package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	checkForRegisteredNodesInterval = 5 * time.Second
)

const (
	ApiVersion = "1.0.0"
	Header     = "Computantis-Central"
)

const (
	searchGroupURL      = "/search"
	addressURL          = "/address"
	blockURL            = "/block"
	transactionGroupURL = "/transaction"
	tokenGroupURL       = "/token"
	validatorGroupURL   = "/validator"
	proposeURL          = "/propose"
	confirmURL          = "/confirm"
	awaitedURL          = "/awaited"
	issuedURL           = "/issued"
	dataURL             = "/data"
	addressGroupURL     = "/address"
	createURL           = "/create"
	generateURL         = "/generate"
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
	GenerateTokenURL      = tokenGroupURL + generateURL      // URL to generate new token.
	WsURL                 = "/ws"                            // URL to connect to websocket.
)

const queryLimit = 100

var (
	ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
	ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
)

// Register abstracts node registration operations.
type Register interface {
	RegisterNode(ctx context.Context, n, ws string) error
	UnregisterNode(ctx context.Context, n string) error
	ReadRegisteredNodesAddresses(ctx context.Context) ([]string, error)
	CountRegistered(ctx context.Context) (int, error)
}

// AddressReaderWriterModifier abstracts address operations.
type AddressReaderWriterModifier interface {
	FindAddress(ctx context.Context, search string, limit int) ([]string, error)
	CheckAddressExists(ctx context.Context, address string) (bool, error)
	WriteAddress(ctx context.Context, address string) error
	IsAddressSuspended(ctx context.Context, addr string) (bool, error)
	IsAddressStandard(ctx context.Context, addr string) (bool, error)
	IsAddressTrusted(ctx context.Context, addr string) (bool, error)
	IsAddressAdmin(ctx context.Context, addr string) (bool, error)
}

// TokenWriteInvalidateChecker abstracts token operations.
type TokenWriteInvalidateChecker interface {
	WriteToken(ctx context.Context, tkn string, expirationDate int64) error
	CheckToken(ctx context.Context, token string) (bool, error)
	InvalidateToken(ctx context.Context, token string) error
}

// Repository is the interface that wraps the basic CRUD and Search methods.
// Repository should be properly indexed to allow for transaction and block hash.
// as well as address public keys to be and unique and the hash lookup should be fast.
// Repository holds the blocks and transaction that are part of the blockchain.
type Repository interface {
	Register
	AddressReaderWriterModifier
	TokenWriteInvalidateChecker
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
	ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
}

// Verifier provides methods to verify the signature of the message.
type Verifier interface {
	VerifySignature(message, signature []byte, hash [32]byte, address string) error
}

// Bookkeeper abstracts methods of the bookkeeping of a blockchain.
type Bookkeeper interface {
	Verifier
	Run(ctx context.Context)
	WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

// ReactiveSubscriberProvider provides reactive subscription to the blockchain.
// It allows to listen for the new blocks created by the Ladger.
type ReactiveSubscriberProvider interface {
	Cancel()
	Channel() <-chan block.Block
}

// Config contains configuration of the server.
type Config struct {
	Port             int    `yaml:"port"`              // Port to listen on.
	DataSizeBytes    int    `yaml:"data_size_bytes"`   // Size of the data to be stored in the transaction.
	WebsocketAddress string `yaml:"websocket_address"` // Address of the websocket server.
}

type server struct {
	dataSize     int
	repo         Repository
	bookkeeping  Bookkeeper
	randDataProv RandomDataProvideValidator
	hub          *hub
	log          logger.Logger
	rx           ReactiveSubscriberProvider
}

// Run initializes routing and runs the server. To stop the server cancel the context.
// It blocks until the context is canceled.
func Run(
	ctx context.Context, c Config, repo Repository,
	bookkeeping Bookkeeper, pv RandomDataProvideValidator,
	log logger.Logger, rx ReactiveSubscriberProvider,
) error {
	var err error
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := validateConfig(&c); err != nil {
		return err
	}

	id := primitive.NewObjectID().Hex()
	repo.RegisterNode(ctxx, id, c.WebsocketAddress)

	s := &server{
		dataSize:     c.DataSizeBytes,
		repo:         repo,
		bookkeeping:  bookkeeping,
		randDataProv: pv,
		hub:          newHub(log),
		log:          log,
		rx:           rx,
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
	router.Use(recover.New())

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

	token := router.Group(tokenGroupURL)
	token.Post(GenerateTokenURL, s.tokenGenerate)

	router.Group(WsURL, func(c *fiber.Ctx) error { return s.wsWrapper(ctxx, c) })

	go func() {
		bookkeeping.Run(ctxx)
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", c.Port))
		if err != nil {
			cancel()
		}
	}()
	go s.hub.run(ctxx)
	go s.runSubscriber(ctxx)
	go s.runControlCentralNodes(ctxx)

	<-ctxx.Done()

	if errx := router.Shutdown(); errx != nil {
		err = errors.Join(err, errx)
	}

	ctxxx, cancelx := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelx()
	if err := repo.UnregisterNode(ctxxx, id); err != nil {
		log.Fatal(err.Error())
	}

	return err
}

func validateConfig(c *Config) error {
	if c.Port == 0 || c.Port > 65535 {
		return ErrWrongPortSpecified
	}

	if c.DataSizeBytes < 1024 || c.DataSizeBytes > 15000000 {
		return ErrWrongMessageSize
	}

	return nil
}

func (s *server) runSubscriber(ctx context.Context) {
	defer s.rx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-s.rx.Channel():
			m := Message{
				Command:     CommandNewBlock,
				Error:       "",
				Block:       b,
				Transaction: transaction.Transaction{},
			}
			s.hub.broadcast <- &m
		}
	}
}

func (s *server) runControlCentralNodes(ctx context.Context) {
	socketCount := 1
	tc := time.NewTicker(checkForRegisteredNodesInterval)
	defer tc.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.C:
			count, err := s.repo.CountRegistered(ctx)
			if err != nil {
				s.log.Error(err.Error())
				continue
			}
			if count != socketCount {
				sockets, err := s.repo.ReadRegisteredNodesAddresses(ctx)
				if err != nil {
					s.log.Error(err.Error())
					continue
				}
				s.hub.broadcast <- &Message{
					Command:     CommandSocketList,
					Error:       "",
					Block:       block.Block{},
					Transaction: transaction.Transaction{},
					Sockets:     sockets,
				}
			}
			socketCount = count
		}
	}
}
