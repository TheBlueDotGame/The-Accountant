package notaryserver

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/providers"
	"github.com/bartossh/Computantis/storage"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/transformers"
	"github.com/bartossh/Computantis/versioning"
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
	checkForRegisteredNodesInterval = 5 * time.Second
	transactionsUpdateTick          = time.Millisecond * 100
)

const rxNewTrxIssuerAddrBufferSize = 100

var (
	ErrWrongPortSpecified = errors.New("port must be between 1 and 65535")
	ErrWrongMessageSize   = errors.New("message size must be between 1024 and 15000000")
	ErrTrxAlreadyExists   = errors.New("transaction already exists")
	ErrRequestIsEmpty     = errors.New("request is empty")
	ErrVerification       = errors.New("verification failed, forbidden")
	ErrDataEmpty          = errors.New("empty data, invalid contract")
	ErrProcessing         = errors.New("processing request failed")
)

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type accounter interface {
	Address() string
	CreateLeaf(ctx context.Context, trx *transaction.Transaction) (accountant.Vertex, error)
	ReadTransactionsByHashes(ctx context.Context, hashes [][32]byte) ([]transaction.Transaction, error)
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
	NodePublicURL           string `yaml:"public_url"`                  // Public URL at which node can be reached.
	TrxAwaitedDBPath        string `yaml:"trx_awaited_db_path"`         // awaited transaction volume path
	AddressAwaitedTrxDBPath string `yaml:"address_awaited_trx_db_path"` // wallet address awaited transaction volume path
	Port                    int    `yaml:"port"`                        // Port to listen on.
	DataSizeBytes           int    `yaml:"data_size_bytes"`             // Size of the data to be stored in the transaction.
}

type server struct {
	protobufcompiled.UnimplementedNotaryAPIServer
	pub                  nodeNetworkingPublisher
	randDataProv         RandomDataProvideValidator
	tele                 providers.HistogramProvider
	log                  logger.Logger
	rxNewTrxIssuerAddrCh chan string
	vrxGossipCh          chan<- *accountant.Vertex
	verifier             verifier
	acc                  accounter
	trxsAwaitedDB        *badger.DB
	addressAwaitedTrxsDB *badger.DB
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

	trxsAwaitedDB, err := storage.CreateBadgerDB(ctx, c.TrxAwaitedDBPath, log)
	if err != nil {
		return err
	}
	addressAwaitedTrxsDB, err := storage.CreateBadgerDB(ctx, c.AddressAwaitedTrxDBPath, log)
	if err != nil {
		return err
	}

	s := &server{
		pub:                  pub,
		randDataProv:         pv,
		tele:                 tele,
		log:                  log,
		rxNewTrxIssuerAddrCh: make(chan string, rxNewTrxIssuerAddrBufferSize),
		vrxGossipCh:          vrxCh,
		verifier:             v,
		acc:                  acc,
		trxsAwaitedDB:        trxsAwaitedDB,
		addressAwaitedTrxsDB: addressAwaitedTrxsDB,
		nodePublicURL:        c.NodePublicURL,
		dataSize:             c.DataSizeBytes,
	}

	s.tele.CreateUpdateObservableHistogtram(proposeTrxTelemetryHistogram, "Propose trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(confirmTrxTelemetryHistogram, "Confirm trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(rejectTrxTelemetryHistogram, "Reject trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(awaitedTrxTelemetryHistogram, "Read awaited / issued trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(approvedTrxTelemetryHistogram, "Read approved trx endpoint request duration on [ ms ].")
	s.tele.CreateUpdateObservableHistogtram(dataToSignTelemetryHistogram, "Generate data to sign endpoint request duration on [ ms ].")

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", c.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterNotaryAPIServer(grpcServer, s)

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			s.log.Fatal(fmt.Sprintf("Node [ %s ] cannot start server on port [ %v ], %s", s.acc.Address(), c.Port, err))
			cancel()
		}
	}()

	time.Sleep(time.Millisecond * 50) // just wait so the server can start

	defer grpcServer.GracefulStop()

	go s.runSubscriber(ctxx)

	<-ctxx.Done()

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

func add(originalValues, newValue []byte) []byte {
	return append(originalValues, append([]byte{','}, newValue...)...)
}

func remove(values, removeValue []byte) []byte {
	sl := bytes.Split(values, []byte{','})
	newValues := make([]byte, 0, len(values))
	for _, v := range sl {
		if bytes.Equal(v, removeValue) {
			continue
		}
		newValues = append(newValues, append([]byte{','}, v...)...)
	}
	return newValues
}

func hexEncode(src []byte) []byte {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return dst
}

func hexDecode(src []byte) ([]byte, error) {
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	return dst, err
}

func (s *server) saveAwaitedTrx(ctx context.Context, trx *transaction.Transaction) error {
	if trx == nil {
		return nil
	}
	buf, err := trx.Encode()
	if err != nil {
		return err
	}
	err = s.trxsAwaitedDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(trx.Hash[:]); err == nil {
			return ErrTrxAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(trx.Hash[:], buf))
	})
	if err != nil {
		if errors.Is(err, ErrTrxAlreadyExists) {
			return nil
		}
		return err
	}

	hashHex := hexEncode(trx.Hash[:])

	for _, address := range []string{trx.IssuerAddress, trx.ReceiverAddress} {
		m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), add, time.Nanosecond)
		if err := m.Add(hashHex); err != nil {
			s.log.Error(fmt.Sprintf("saving address awaited failed for [ %s ], hex %v", address, hashHex))
		}
		m.Stop()
	}

	return nil
}

func (s *server) readAwaitedTrx(address string) ([]transaction.Transaction, error) {
	hashesHex := make([][]byte, 0)
	err := s.addressAwaitedTrxsDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(address))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}

		if err := item.Value(func(val []byte) error {
			hashesHex = bytes.Split(val, []byte{','})
			return nil
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(hashesHex) == 0 {
		return nil, nil
	}

	cleanup := make([][]byte, 0)
	trxs := make([]transaction.Transaction, 0)
	err = s.trxsAwaitedDB.View(func(txn *badger.Txn) error {
		for _, h := range hashesHex {
			hash, err := hexDecode(h)
			if err != nil {
				s.log.Error(fmt.Sprintf("read awaited trxs failed to decode hex hash %v for address [ %s ]", h, address))
				continue
			}
			item, err := txn.Get(hash)
			if err != nil {
				if errors.Is(err, badger.ErrKeyNotFound) {
					cleanup = append(cleanup, h)
					continue
				}
				return err
			}
			if err := item.Value(func(val []byte) error {
				trx, err := transaction.Decode(val)
				if err != nil {
					return err
				}
				trxs = append(trxs, trx)
				return nil
			}); err != nil {
				return err
			}
			return nil
		}
		return nil
	})

	if len(cleanup) == 0 {
		return trxs, err
	}

	m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), remove, time.Nanosecond)
	defer m.Stop()
	for _, h := range cleanup {
		err := m.Add(h)
		if err != nil {
			s.log.Error(fmt.Sprintf("cleanup for hex hash %v failed, %s", h, err))
		}
	}

	return trxs, err
}

func (s *server) removeAwaitedTrx(h []byte, receiver string) error {
	var trx transaction.Transaction
	err := s.trxsAwaitedDB.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(h)
		if err != nil {
			return err
		}
		if err := item.Value(func(val []byte) error {
			var err error
			trx, err = transaction.Decode(val)
			if err != nil {
				return err
			}
			if trx.ReceiverAddress != receiver {
				return errors.New("transaction receiver is not matching provided receiver")
			}
			return nil
		}); err != nil {
			return err
		}

		return txn.Delete(h)
	})
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		return err
	}

	hashHex := hexEncode(h)
	for _, address := range [2]string{trx.IssuerAddress, receiver} {
		m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), remove, time.Nanosecond)
		if err := m.Add(hashHex); err != nil {
			s.log.Error(fmt.Sprintf("removing address of  awaited failed for [ %s ], hex %v", address, hashHex))
		}
		m.Stop()
	}

	return err
}

// Alive returns alive information such as public address API version and API header of running server.
func (s *server) Alive(ctx context.Context, _ *emptypb.Empty) (*protobufcompiled.AliveData, error) {
	return &protobufcompiled.AliveData{
		PublicAddress: s.nodePublicURL,
		ApiVersion:    versioning.ApiVersion,
		ApiHeader:     versioning.Header,
	}, nil
}

// Propose validates the transaction and then processes the transaction according to the data in transaction.
func (s *server) Propose(ctx context.Context, in *protobufcompiled.Transaction) (*emptypb.Empty, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(proposeTrxTelemetryHistogram, time.Since(t))

	trx, err := transformers.ProtoTrxToTrx(in)
	if err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, message is empty or invalid: %s", err))
		return nil, err
	}

	if err := trx.VerifyIssuer(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, verification failed: %s", err))
		return nil, ErrVerification
	}

	switch trx.IsContract() {
	case true:
		if err := s.saveAwaitedTrx(ctx, &trx); err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, saving awaited trx for issuer [ %s ], %s", trx.IssuerAddress, err))
			return nil, ErrProcessing
		}
		addresses := []string{trx.IssuerAddress, trx.ReceiverAddress}
		if err := s.pub.PublishAddressesAwaitingTrxs(addresses, s.nodePublicURL); err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, publishing awaited trx for addresses %v, failed, %s", addresses, err))
		}
	default:
		if len(trx.Data) > s.dataSize {
			s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(trx.Data)))
			return nil, ErrProcessing
		}
		vrx, err := s.acc.CreateLeaf(ctx, &trx)
		if err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, creating leaf: %s", err))
			return nil, ErrProcessing
		}
		go func(v *accountant.Vertex) {
			s.vrxGossipCh <- v
		}(&vrx)
	}

	return &emptypb.Empty{}, nil
}

// Confirm validates the transaction and processes the confirmation according to the data in transaction.
func (s *server) Confirm(ctx context.Context, in *protobufcompiled.Transaction) (*emptypb.Empty, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(confirmTrxTelemetryHistogram, time.Since(t))

	trx, err := transformers.ProtoTrxToTrx(in)
	if err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, message is empty or invalid: %s", err))
		return nil, err
	}

	if err := trx.VerifyIssuerReceiver(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf(
			"confirm endpoint, failed to verify trx hash %v from receiver [ %s ], %s", trx.Hash, trx.ReceiverAddress, err.Error(),
		))
		return nil, ErrVerification
	}

	if err := s.removeAwaitedTrx(trx.Hash[:], trx.ReceiverAddress); err != nil {
		s.log.Error(
			fmt.Sprintf(
				"confirm endpoint, failed to remove awaited trx hash %v from receiver [ %s ] , %s", trx.Hash, trx.ReceiverAddress, err.Error(),
			))
		return nil, ErrProcessing
	}

	vrx, err := s.acc.CreateLeaf(ctx, &trx)
	if err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, creating leaf: %s", err))
		return nil, ErrProcessing
	}

	go func(v *accountant.Vertex) {
		s.vrxGossipCh <- v
	}(&vrx)

	return &emptypb.Empty{}, nil
}

// Reject validates request and then attempts to remove transaction from awaited transactions kept by this node.
func (s *server) Reject(ctx context.Context, in *protobufcompiled.SignedHash) (*emptypb.Empty, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(rejectTrxTelemetryHistogram, time.Since(t))

	if in == nil {
		return nil, ErrRequestIsEmpty
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint, failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}

	if err := s.removeAwaitedTrx(in.Data, in.Address); err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint, failed removing transaction %v for address [ %s ]", in.Hash, in.Address))
		return nil, ErrProcessing
	}
	return &emptypb.Empty{}, nil
}

// Waiting endpoint returns all the awaited transactions for given address, those received and issued.
func (s *server) Waiting(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Transactions, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(awaitedTrxTelemetryHistogram, time.Since(t))

	if ok := s.randDataProv.ValidateData(in.Address, in.Data); !ok {
		s.log.Error(fmt.Sprintf("waiting transactions endpoint, failed to validate data for address: %s", in.Address))
		return nil, ErrVerification
	}

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint, failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	trxs, err := s.readAwaitedTrx(in.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint, failed to read awaited transactions for address: %s, %s", in.Address, err))
		return nil, ErrProcessing
	}
	result := &protobufcompiled.Transactions{Array: make([]*protobufcompiled.Transaction, 0, len(trxs)), Len: uint64(len(trxs))}
	for _, trx := range trxs {
		protoTrx, err := transformers.TrxToProtoTrx(&trx)
		if err != nil {
			s.log.Warn(fmt.Sprintf("waiting endpoint, failed to map trx to protobuf trx for address: %s, %s", in.Address, err))
			continue
		}
		result.Array = append(result.Array, protoTrx)
	}

	return result, nil
}

func (s *server) Saved(ctx context.Context, in *protobufcompiled.SignedHash) (*protobufcompiled.Transaction, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(approvedTrxTelemetryHistogram, time.Since(t))

	if err := s.verifier.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		s.log.Error(fmt.Sprintf("waiting endpoint, failed to verify signature for address: %s, %s", in.Address, err))
		return nil, ErrVerification
	}

	trxs, err := s.acc.ReadTransactionsByHashes(ctx, [][32]byte{[32]byte(in.Hash)})
	if err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to read hash %v for address: %s, %s", in.Hash, in.Address, err))
		return nil, ErrProcessing
	}

	if len(trxs) == 0 {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to read hash %v for address: %s", in.Hash, in.Address))
		return nil, ErrProcessing
	}

	protoTrx, err := transformers.TrxToProtoTrx(&trxs[0])
	if err != nil {
		return nil, err
	}
	return protoTrx, nil
}

// Data generates temporary data blob for receiver to sign and proof the its identity.
func (s *server) Data(ctx context.Context, in *protobufcompiled.Address) (*protobufcompiled.DataBlob, error) {
	t := time.Now()
	defer s.tele.RecordHistogramTime(dataToSignTelemetryHistogram, time.Since(t))

	if in.Public == "" {
		s.log.Error("empty request on data endpoint")
		return nil, ErrRequestIsEmpty
	}

	d := s.randDataProv.ProvideData(in.Public)

	return &protobufcompiled.DataBlob{Blob: d}, nil
}
