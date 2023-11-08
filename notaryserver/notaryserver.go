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

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/providers"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/versioning"
)

const (
	checkForRegisteredNodesInterval = 5 * time.Second
	transactionsUpdateTick          = time.Millisecond * 100
)

const (
	proposeTrxTelemetryHistogram  = "propose_trx_request_duration"
	confirmTrxTelemetryHistogram  = "confirm_trx_request_duration"
	rejectTrxTelemetryHistogram   = "reject_trx_request_duration"
	awaitedTrxTelemetryHistogram  = "read_awaited_trx_request_duration"
	approvedTrxTelemetryHistogram = "read_approved_trx_request_duration"
	dataToSignTelemetryHistogram  = "data_to_sign_request_duration"
)

const (
	transactionGroupURL = "/transaction"
	validatorGroupURL   = "/validator"
	proposeURL          = "/propose"
	confirmURL          = "/confirm"
	awaitedURL          = "/awaited"
	issuedURL           = "/issued"
	approvedURL         = "/approved"
	dataURL             = "/data"
	rejectURL           = "/reject"
)

const (
	MetricsURL             = "/metrics"                        // URL to check service metrics
	AliveURL               = "/alive"                          // URL to check if server is alive and version.
	ProposeTransactionURL  = transactionGroupURL + proposeURL  // URL to propose transaction signed by the issuer.
	ConfirmTransactionURL  = transactionGroupURL + confirmURL  // URL to confirm transaction signed by the receiver.
	RejectTransactionURL   = transactionGroupURL + rejectURL   // URL to reject transaction signed only by issuer.
	AwaitedTransactionURL  = transactionGroupURL + awaitedURL  // URL to get awaited transactions for the receiver.
	ApprovedTransactionURL = transactionGroupURL + approvedURL // URL to get approved transactions for given address.
	DataToValidateURL      = validatorGroupURL + dataURL       // URL to get data to validate address by signing raw message.
)

const rxNewTrxIssuerAddrBufferSize = 100

var (
	ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
	ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
)

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type accounter interface {
	CreateLeaf(ctx context.Context, trx *transaction.Transaction) (accountant.Vertex, error)
}

// RandomDataProvideValidator provides random binary data for signing to prove identity and
// the validator of data being valid and not expired.
type RandomDataProvideValidator interface {
	ProvideData(address string) []byte
	ValidateData(address string, data []byte) bool
}

type nodeNetworkingPublisher interface {
	PublishAddressesAwaitingTrxs(addresses []string, notaryNodeURL string) error
}

// Config contains configuration of the server.
type Config struct {
	NodePublicURL string `yaml:"public_url"`      // Public URL at which node can be reached.
	Port          int    `yaml:"port"`            // Port to listen on.
	DataSizeBytes int    `yaml:"data_size_bytes"` // Size of the data to be stored in the transaction.
}

type server struct {
	pub                  nodeNetworkingPublisher
	randDataProv         RandomDataProvideValidator
	tele                 providers.HistogramProvider
	log                  logger.Logger
	rxNewTrxIssuerAddrCh chan string
	vrxGossipCh          chan<- *accountant.Vertex
	verifier             verifier
	acc                  accounter
	nodePublicURL        string
	dataSize             int
}

// Run initializes routing and runs the server. To stop the server cancel the context.
// It blocks until the context is canceled.
func Run(
	ctx context.Context, c Config, pub nodeNetworkingPublisher, pv RandomDataProvideValidator, tele providers.HistogramProvider,
	log logger.Logger, v verifier, acc accounter, vrxCh chan<- *accountant.Vertex,
) error {
	var err error
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer close(vrxCh)

	if err = validateConfig(&c); err != nil {
		return err
	}

	if _, err = url.Parse(c.NodePublicURL); err != nil {
		return err
	}

	id := primitive.NewObjectID().Hex()

	s := &server{
		pub:                  pub,
		randDataProv:         pv,
		tele:                 tele,
		log:                  log,
		rxNewTrxIssuerAddrCh: make(chan string, rxNewTrxIssuerAddrBufferSize),
		vrxGossipCh:          vrxCh,
		verifier:             v,
		acc:                  acc,
		nodePublicURL:        c.NodePublicURL,
		dataSize:             c.DataSizeBytes,
	}

	router := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
		ReadTimeout:   time.Second * 5,
		WriteTimeout:  time.Second * 5,
		ServerHeader:  versioning.Header,
		AppName:       versioning.ApiVersion,
		Concurrency:   4096,
	})
	router.Use(recover.New())
	router.Get(MetricsURL, monitor.New(monitor.Config{Title: fmt.Sprintf("The Computantis Node %s", id)}))
	router.Get(AliveURL, s.alive)

	transaction := router.Group(transactionGroupURL)
	transaction.Post(proposeURL, s.propose)
	transaction.Post(confirmURL, s.confirm)
	transaction.Post(rejectURL, s.reject)
	transaction.Post(awaitedURL, s.awaited)
	transaction.Post(approvedURL, s.approved)

	validator := router.Group(validatorGroupURL)
	validator.Post(dataURL, s.data)

	s.tele.CreateUpdateObservableHistogtram(proposeTrxTelemetryHistogram, "Propose trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(confirmTrxTelemetryHistogram, "Confirm trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(rejectTrxTelemetryHistogram, "Reject trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(awaitedTrxTelemetryHistogram, "Read awaited / issued trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(approvedTrxTelemetryHistogram, "Read approved trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(dataToSignTelemetryHistogram, "Generate data to sign endpoint request duration on [ ms ].")

	go func() {
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
	ticker := time.NewTicker(transactionsUpdateTick)
	defer ticker.Stop()

	receiverAddrSet := make(map[string]struct{}, 100)

	for {
		select {
		case <-ctx.Done():
			return
		case recAddr := <-s.rxNewTrxIssuerAddrCh:
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
