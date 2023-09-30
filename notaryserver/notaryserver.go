package notaryserver

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/providers"
	"github.com/bartossh/Computantis/transaction"
)

const (
	checkForRegisteredNodesInterval = 5 * time.Second
	transactionsUpdateTick          = time.Millisecond * 100
)

const (
	ApiVersion = "1.0.0"
	Header     = "Computantis-Notary"
)

const (
	discoverCentralNodeTelemetryHistogram = "discover_central_nodes_request_duration"
	addressURLTelemetryHistogram          = "address_url_request_duration"
	trxInBlockTelemetryHistogram          = "trx_in_block_request_duration"
	proposeTrxTelemetryHistogram          = "propose_trx_request_duration"
	confirmTrxTelemetryHistogram          = "confirm_trx_request_duration"
	rejectTrxTelemetryHistogram           = "reject_trx_request_duration"
	awaitedTrxTelemetryHistogram          = "read_awaited_trx_request_duration"
	issuedTrxTelemetryHistogram           = "read_issued_trx_request_duration"
	rejectedTrxTelemetryHistogram         = "read_rejected_trx_request_duration"
	approvedTrxTelemetryHistogram         = "read_approved_trx_request_duration"
	dataToSignTelemetryHistogram          = "data_to_sign_request_duration"
	addressCreateTelemetryHistogram       = "address_create_request_duration"
	tokenGenerateTelemetryHistogram       = "token_generate_request_duration"
	wsSocketListTelemetryHistogram        = "get_socket_list_ws_request_duration"
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
	rejectedURL         = "/rejected"
	approvedURL         = "/approved"
	dataURL             = "/data"
	addressGroupURL     = "/address"
	createURL           = "/create"
	rejectURL           = "/reject"
	generateURL         = "/generate"
)

const (
	MetricsURL             = "/metrics"                        // URL to check service metrics
	AliveURL               = "/alive"                          // URL to check if server is alive and version.
	SearchAddressURL       = searchGroupURL + addressURL       // URL to search for address.
	SearchBlockURL         = searchGroupURL + blockURL         // URL to search for block that contains transaction hash.
	ProposeTransactionURL  = transactionGroupURL + proposeURL  // URL to propose transaction signed by the issuer.
	ConfirmTransactionURL  = transactionGroupURL + confirmURL  // URL to confirm transaction signed by the receiver.
	RejectTransactionURL   = transactionGroupURL + rejectURL   // URL to reject transaction signed only by issuer.
	AwaitedTransactionURL  = transactionGroupURL + awaitedURL  // URL to get awaited transactions for the receiver.
	IssuedTransactionURL   = transactionGroupURL + issuedURL   // URL to get issued transactions for the issuer.
	RejectedTransactionURL = transactionGroupURL + rejectedURL // URL to get rejected transactions for given address.
	ApprovedTransactionURL = transactionGroupURL + approvedURL // URL to get approved transactions for given address.
	DataToValidateURL      = validatorGroupURL + dataURL       // URL to get data to validate address by signing rew message.
	CreateAddressURL       = addressGroupURL + createURL       // URL to create new address.
	GenerateTokenURL       = tokenGroupURL + generateURL       // URL to generate new token.
	WsURL                  = "/ws"                             // URL to connect to websocket.
)

const queryLimit = 100

var (
	ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
	ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
)

type addressReadWriteModdifier interface {
	FindAddress(ctx context.Context, search string, limit int) ([]string, error)
	CheckAddressExists(ctx context.Context, address string) (bool, error)
	WriteAddress(ctx context.Context, address string) error
	IsAddressSuspended(ctx context.Context, addr string) (bool, error)
	IsAddressStandard(ctx context.Context, addr string) (bool, error)
	IsAddressTrusted(ctx context.Context, addr string) (bool, error)
	IsAddressAdmin(ctx context.Context, addr string) (bool, error)
}

type nodeWriteInvalidateChecker interface {
	WriteToken(ctx context.Context, tkn string, expirationDate int64) error
	CheckToken(ctx context.Context, token string) (bool, error)
	InvalidateToken(ctx context.Context, token string) error
}

type transactionCacher interface {
	ReadAwaitingTransactionsByIssuer(address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByReceiver(address string) ([]transaction.Transaction, error)
	CleanSignedTransactions(hashes []transaction.Transaction)
}

type trxReadWriteRejectApprover interface {
	FindTransactionInBlockHash(ctx context.Context, trxBlockHash [32]byte) ([32]byte, error)
	ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadRejectedTransactionsPagginate(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
	ReadApprovedTransactions(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error)
	RejectTransactions(ctx context.Context, receiver string, trxs []transaction.Transaction) error
}

type verifier interface {
	VerifySignature(message, signature []byte, hash [32]byte, address string) error
}

type bookkeeper interface {
	verifier
	Run(ctx context.Context) error
	WriteCandidateTransaction(ctx context.Context, tx *transaction.Transaction) error
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, trxBlock *transaction.Transaction) error
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

type reactiveBlock interface {
	Cancel()
	Channel() <-chan block.Block
}

type reactiveTrxIssued interface {
	Cancel()
	Channel() <-chan string
}

type nodeNetworkingPublisher interface {
	PublishNewBlock(blk *block.Block, notaryNodeURL string) error
	PublishAddressesAwaitingTrxs(addresses []string, notaryNodeURL string) error
}

// Config contains configuration of the server.
type Config struct {
	NodePublicURL string `yaml:"node_public_url"` // Public URL at which node can be reached.
	Port          int    `yaml:"port"`            // Port to listen on.
	DataSizeBytes int    `yaml:"data_size_bytes"` // Size of the data to be stored in the transaction.
}

type server struct {
	cache         transactionCacher
	trxProv       trxReadWriteRejectApprover
	pub           nodeNetworkingPublisher
	addressProv   addressReadWriteModdifier
	tokenProv     nodeWriteInvalidateChecker
	bookkeeping   bookkeeper
	randDataProv  RandomDataProvideValidator
	tele          providers.HistogramProvider
	log           logger.Logger
	rxBlock       reactiveBlock
	rxTrxIssued   reactiveTrxIssued
	nodePublicURL string
	dataSize      int
}

// Run initializes routing and runs the server. To stop the server cancel the context.
// It blocks until the context is canceled.
func Run(
	ctx context.Context, c Config, cache transactionCacher, trxProv trxReadWriteRejectApprover, pub nodeNetworkingPublisher,
	addressProv addressReadWriteModdifier, tokenProv nodeWriteInvalidateChecker, bookkeeping bookkeeper,
	pv RandomDataProvideValidator, tele providers.HistogramProvider, log logger.Logger,
	rxBlock reactiveBlock, rxTrxIssued reactiveTrxIssued,
) error {
	var err error
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err = validateConfig(&c); err != nil {
		return err
	}

	if _, err = url.Parse(c.NodePublicURL); err != nil {
		return err
	}

	id := primitive.NewObjectID().Hex()

	s := &server{
		cache:         cache,
		pub:           pub,
		trxProv:       trxProv,
		addressProv:   addressProv,
		tokenProv:     tokenProv,
		bookkeeping:   bookkeeping,
		randDataProv:  pv,
		tele:          tele,
		log:           log,
		rxBlock:       rxBlock,
		rxTrxIssued:   rxTrxIssued,
		nodePublicURL: c.NodePublicURL,
		dataSize:      c.DataSizeBytes,
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
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: fmt.Sprintf("Central Node %s", id)}))
	router.Get(AliveURL, s.alive)

	search := router.Group(searchGroupURL)
	search.Post(addressURL, s.address)
	search.Post(blockURL, s.trxInBlock)

	transaction := router.Group(transactionGroupURL)
	transaction.Post(proposeURL, s.propose)
	transaction.Post(confirmURL, s.confirm)
	transaction.Post(rejectURL, s.reject)
	transaction.Post(awaitedURL, s.awaited)
	transaction.Post(issuedURL, s.issued)
	transaction.Post(rejectedURL, s.rejected)
	transaction.Post(approvedURL, s.approved)

	validator := router.Group(validatorGroupURL)
	validator.Post(dataURL, s.data)

	address := router.Group(addressGroupURL)
	address.Post(createURL, s.addressCreate)

	token := router.Group(tokenGroupURL)
	token.Post(GenerateTokenURL, s.tokenGenerate)

	s.tele.CreateUpdateObservableHistogtram(discoverCentralNodeTelemetryHistogram, "Discover central nodes endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(addressURLTelemetryHistogram, "Address URL endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(trxInBlockTelemetryHistogram, "Transaction in block lookup endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(proposeTrxTelemetryHistogram, "Propose trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(confirmTrxTelemetryHistogram, "Confirm trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(rejectTrxTelemetryHistogram, "Reject trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(awaitedTrxTelemetryHistogram, "Read awaited trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(issuedTrxTelemetryHistogram, "Read issued trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(rejectedTrxTelemetryHistogram, "Read rejected trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(approvedTrxTelemetryHistogram, "Read approved trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(dataToSignTelemetryHistogram, "Generate data to sign endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(addressCreateTelemetryHistogram, "Create address endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(tokenGenerateTelemetryHistogram, "Generate token endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(wsSocketListTelemetryHistogram, "Websocket read socket list endpoint request duration on [ ms ].")

	go func() {
		if err := bookkeeping.Run(ctxx); err != nil {
			log.Error(err.Error())
			cancel()
		}
		err := router.Listen(fmt.Sprintf("0.0.0.0:%v", c.Port))
		if err != nil {
			log.Error(err.Error())
			cancel()
		}
	}()
	go s.runSubscriber(ctxx)

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

	if c.DataSizeBytes < 1024 || c.DataSizeBytes > 15000000 {
		return ErrWrongMessageSize
	}

	return nil
}

func (s *server) runSubscriber(ctx context.Context) {
	defer s.rxBlock.Cancel()
	defer s.rxTrxIssued.Cancel()
	ticker := time.NewTicker(transactionsUpdateTick)
	defer ticker.Stop()

	receiverAddrSet := make(map[string]struct{}, 100)

	for {
		select {
		case <-ctx.Done():
			return
		case b := <-s.rxBlock.Channel():
			s.pub.PublishNewBlock(&b, s.nodePublicURL)
		case recAddr := <-s.rxTrxIssued.Channel():
			receiverAddrSet[recAddr] = struct{}{}
		case <-ticker.C:
			if len(receiverAddrSet) == 0 {
				continue
			}

			addresses := make([]string, 0, len(receiverAddrSet))
			for addr := range receiverAddrSet {
				addresses = append(addresses, addr)
			}

			s.pub.PublishAddressesAwaitingTrxs(addresses, s.nodePublicURL)

			receiverAddrSet = make(map[string]struct{}, 1000)
		}
	}
}
