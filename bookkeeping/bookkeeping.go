package bookkeeping

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	minDifficulty = 1
	maxDifficulty = 124

	minBlockWriteTimestamp = time.Second
	maxBlockWriteTimestamp = time.Hour * 4 // value is picked arbitrary

	minBlockTransactionsSize = 1
	maxBlockTransactionsSize = 60000 // calculated to be below 16MB of a block size, it is a limit of single document in mongodb
)

var (
	ErrTrxExistsInTheLadger            = errors.New("transaction is already in the ledger")
	ErrTrxExistsInTheBlockchain        = errors.New("transaction is already in the blockchain")
	ErrAddressNotExists                = errors.New("address does not exist in the addresses repository")
	ErrBlockTxsCorrupted               = errors.New("all transaction failed, block corrupted")
	ErrDifficultyNotInRange            = errors.New("invalid difficulty, difficulty can by in range [1 : 255]")
	ErrBlockWriteTimestampNoInRange    = errors.New("block write timestamp is not in range of [one second : four hours]")
	ErrBlockTransactionsSizeNotInRange = errors.New("block transactions size is not in range of [1 : 60000]")
)

// TrxWriteReadMover provides transactions write, read and move methods.
// It allows to access temporary, permanent and awaiting transactions.
type TrxWriteReadMover interface {
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
	MoveTransactionsFromTemporaryToPermanent(ctx context.Context, blockHash [32]byte, hashes [][32]byte) error
	MoveTransactionFromAwaitingToTemporary(ctx context.Context, trx *transaction.Transaction) error
	ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadTemporaryTransactions(ctx context.Context, offset, limit int) ([]transaction.Transaction, error)
}

// BlockReader provides block read methods.
type BlockReader interface {
	LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}

// BlockWriter provides block write methods.
type BlockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

// BlockReadWriter provides block read and write methods.
type BlockReadWriter interface {
	BlockReader
	BlockWriter
}

// AddressChecker provides address existence check method.
// If you use other repository than addresses repository, you can implement this interface
// but address should be uniquely indexed in your repository implementation.
type AddressChecker interface {
	CheckAddressExists(ctx context.Context, address string) (bool, error)
}

// SignatureVerifier provides signature verification method.
type SignatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// BlockFindWriter provides block find and write method.
type BlockFindWriter interface {
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}

// NodeRegister abstracts node registration operations.
type NodeRegister interface {
	CountRegistered(ctx context.Context) (int, error)
}

// DataBaseProvider abstracts all the methods that are expected from repository.
type DataBaseProvider interface {
	Synchronizer
	TrxWriteReadMover
	NodeRegister
}

// BlockReactivePublisher provides block publishing method.
// It uses reactive package. It you are using your own implementation of reactive package
// take care of Publish method to be non-blocking.
type BlockReactivePublisher interface {
	Publish(block.Block)
}

// IssuerTrxSubscription provides trx issuer address publishing method.
// It uses reactive package. It you are using your own implementation of reactive package
// take care of Publish method to be non-blocking.
type TrxIssuedReactivePunlisher interface {
	Publish(string)
}

// Config is a configuration of the Ledger.
type Config struct {
	Difficulty            uint64 `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
	BlockWriteTimestamp   uint64 `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
	BlockTransactionsSize int    `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}

// Validate validates the Ledger configuration.
func (c Config) Validate() error {
	if c.Difficulty < minDifficulty || c.Difficulty > maxDifficulty {
		return ErrDifficultyNotInRange
	}

	if time.Duration(c.BlockWriteTimestamp)*time.Second < minBlockWriteTimestamp ||
		time.Duration(c.BlockWriteTimestamp)*time.Second > maxBlockWriteTimestamp {
		return ErrBlockWriteTimestampNoInRange
	}

	if c.BlockTransactionsSize < minBlockTransactionsSize || c.BlockTransactionsSize > maxBlockTransactionsSize {
		return ErrBlockTransactionsSizeNotInRange
	}

	return nil
}

// Ledger is a collection of ledger functionality to perform bookkeeping.
// It performs all the actions on the transactions and blockchain.
// Ladger seals all the transaction actions in the blockchain.
type Ledger struct {
	id           string
	config       Config
	hashC        chan [32]byte
	hashes       [][32]byte
	db           DataBaseProvider
	bc           BlockReadWriter
	ac           AddressChecker
	vr           SignatureVerifier
	tf           BlockFindWriter
	log          logger.Logger
	blcPub       BlockReactivePublisher
	trxIssuedPub TrxIssuedReactivePunlisher
	sub          BlockchainLockSubscriber
}

// New creates new Ledger if config is valid or returns error otherwise.
func New(
	config Config,
	bc BlockReadWriter,
	db DataBaseProvider,
	ac AddressChecker,
	vr SignatureVerifier,
	tf BlockFindWriter,
	log logger.Logger,
	blcPub BlockReactivePublisher,
	trxIssuedPub TrxIssuedReactivePunlisher,
	sub BlockchainLockSubscriber,
) (*Ledger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Ledger{
		id:           primitive.NewObjectID().Hex(),
		config:       config,
		hashC:        make(chan [32]byte, config.BlockTransactionsSize),
		db:           db,
		bc:           bc,
		ac:           ac,
		vr:           vr,
		tf:           tf,
		log:          log,
		blcPub:       blcPub,
		trxIssuedPub: trxIssuedPub,
		sub:          sub,
	}, nil
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
// Run starts a goroutine and can be stopped by cancelling the context.
// It is non-blocking and concurrent safe.
func (l *Ledger) Run(ctx context.Context) {
	count, err := l.db.CountRegistered(ctx)
	if err != nil {
		l.log.Fatal(err.Error())
	}
	if count == 1 {
		if err := l.forgeTemporaryTrxs(ctx); err != nil {
			l.log.Fatal(fmt.Sprintf("forging temporary failed: %s", err.Error()))
		}
		l.log.Info("forging temporary transactions finished")
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Duration(l.config.BlockWriteTimestamp) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				if len(l.hashes) > 0 {
					l.forge(ctx)
				}
				return
			case h := <-l.hashC:
				l.hashes = append(l.hashes, h)
				if len(l.hashes) == l.config.BlockTransactionsSize {
					l.forge(ctx)
				}
			case <-ticker.C:
				if len(l.hashes) > 0 {
					l.forge(ctx)
				}
			}
		}
	}(ctx)
}

// WriteIssuerSignedTransactionForReceiver validates issuer signature and writes a transaction to the repository for receiver.
func (l *Ledger) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	trx *transaction.Transaction,
) error {
	if err := l.validatePartiallyTransaction(ctx, trx); err != nil {
		return err
	}

	if err := l.db.WriteIssuerSignedTransactionForReceiver(ctx, trx); err != nil {
		return err
	}

	l.trxIssuedPub.Publish(trx.ReceiverAddress)

	return nil
}

// WriteCandidateTransaction validates and writes a transaction to the repository.
// Transaction is not yet a part of the blockchain at this point.
// Ladger will perform all the necessary checks and validations before writing it to the repository.
// The candidate needs to be signed by the receiver later in the process  to be placed as a candidate in the blockchain.
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, trx *transaction.Transaction) error {
	if err := l.validateFullyTransaction(ctx, trx); err != nil {
		return err
	}

	if err := l.db.MoveTransactionFromAwaitingToTemporary(ctx, trx); err != nil {
		return err
	}

	l.hashC <- trx.Hash

	return nil
}

// VerifySignature verifies signature of the message.
func (l *Ledger) VerifySignature(message, signature []byte, hash [32]byte, address string) error {
	return l.vr.Verify(message, signature, hash, address)
}

func (l *Ledger) forgeTemporaryTrxs(ctx context.Context) error {
	for {
		trxs, err := l.db.ReadTemporaryTransactions(ctx, 0, l.config.BlockTransactionsSize)
		if err != nil {
			return err
		}
		if len(trxs) == 0 {
			return nil
		}
		for _, trx := range trxs {
			l.hashes = append(l.hashes, trx.Hash)
		}
		if len(l.hashes) > 0 {
			l.forge(ctx)
		}
	}
}

func (l *Ledger) forge(ctx context.Context) {
	defer l.cleanHashes()
	sync := newSync(l.id, l.db, l.sub)
	if err := sync.waitInQueueForLock(ctx); err != nil {
		log.Fatal(err.Error())
		return
	}

	blcHash, err := l.savePublishNewBlock(ctx)
	if err != nil {
		msg := fmt.Sprintf("error while saving block: %s", err.Error())
		log.Fatal(msg)
		return
	}

	if err := l.db.MoveTransactionsFromTemporaryToPermanent(ctx, blcHash, l.hashes); err != nil {
		msg := fmt.Sprintf("error while moving transactions from temporary to permanent: %s", err.Error())
		log.Fatal(msg)
	}
	if err := sync.releaseLock(ctx); err != nil {
		log.Fatal(err.Error())
	}
}

func (l *Ledger) cleanHashes() {
	l.hashes = make([][32]byte, 0, l.config.BlockTransactionsSize)
}

func (l *Ledger) savePublishNewBlock(ctx context.Context) ([32]byte, error) {
	h, idx, err := l.bc.LastBlockHashIndex(ctx)
	if err != nil {
		return [32]byte{}, err
	}

	idx++
	nb := block.New(l.config.Difficulty, idx, h, l.hashes)

	if err := l.bc.WriteBlock(ctx, nb); err != nil {
		return [32]byte{}, err
	}

	l.blcPub.Publish(nb)

	return nb.Hash, nil
}

func (l *Ledger) validatePartiallyTransaction(ctx context.Context, trx *transaction.Transaction) error {
	exists, err := l.ac.CheckAddressExists(ctx, trx.IssuerAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	exists, err = l.ac.CheckAddressExists(ctx, trx.ReceiverAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	if err := l.vr.Verify(trx.GetMessage(), trx.IssuerSignature, trx.Hash, trx.IssuerAddress); err != nil {
		return err
	}
	return nil
}

func (l *Ledger) validateFullyTransaction(ctx context.Context, trx *transaction.Transaction) error {
	exists, err := l.ac.CheckAddressExists(ctx, trx.IssuerAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	exists, err = l.ac.CheckAddressExists(ctx, trx.ReceiverAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	if err := l.vr.Verify(trx.GetMessage(), trx.IssuerSignature, trx.Hash, trx.IssuerAddress); err != nil {
		return err
	}

	if err := l.vr.Verify(trx.GetMessage(), trx.ReceiverSignature, trx.Hash, trx.ReceiverAddress); err != nil {
		return err
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
