package bookkeeping

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
)

const (
	minDifficulty = 1
	maxDifficulty = 124

	minBlockWriteTimestamp = time.Second   // less then a second will impose large amount of computations
	maxBlockWriteTimestamp = time.Hour * 4 // value is picked arbitrary

	minBlockTransactionsSize = 1
	maxBlockTransactionsSize = 60000
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

var (
	errSavingBlock        = errors.New("saving block failed")
	errMovingTransactions = errors.New("moving transactions failed")
)

type transactionCacher interface {
	WriteIssuerSignedTransactionForReceiver(trx *transaction.Transaction) error
	CleanSignedTransactions(trxs []transaction.Transaction)
}

type trxWriteReadMover interface {
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, trx *transaction.Transaction) error
	MoveTransactionFromAwaitingToTemporary(ctx context.Context, trx *transaction.Transaction) error
	MoveTransactionsFromTemporaryToPermanent(ctx context.Context, blockHash [32]byte, hashes [][32]byte) error
	ReadTemporaryTransactions(ctx context.Context, offset, limit int) ([]transaction.Transaction, error)
}

type blockReader interface {
	LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}

type blockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

type blockFinder interface {
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}

type blockReadWriteFinder interface {
	blockReader
	blockWriter
	blockFinder
}

type addressChecker interface {
	CheckAddressExists(ctx context.Context, address string) (bool, error)
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type nodeRegister interface {
	CountRegistered(ctx context.Context) (int, error)
	RegisterNode(ctx context.Context, n string) error
	UnregisterNode(ctx context.Context, n string) error
}

type nodeSyncRegister interface {
	synchronizer
	nodeRegister
}

type blockReactivePublisher interface {
	Publish(block.Block)
}

type trxIssuedReactivePublisher interface {
	Publish(string)
}

// Config is a configuration of the Ledger.
type Config struct {
	Difficulty            uint64 `json:"difficulty"              sql:"difficulty"              yaml:"difficulty"`
	BlockWriteTimestamp   uint64 `json:"block_write_timestamp"   sql:"block_write_timestamp"   yaml:"block_write_timestamp"`
	BlockTransactionsSize int    `json:"block_transactions_size" sql:"block_transactions_size" yaml:"block_transactions_size"`
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
	trxCache     transactionCacher
	trx          trxWriteReadMover
	nsc          nodeSyncRegister
	sub          blockchainLockSubscriber
	brwf         blockReadWriteFinder
	ac           addressChecker
	vr           signatureVerifier
	tf           blockFinder
	log          logger.Logger
	blcPub       blockReactivePublisher
	trxIssuedPub trxIssuedReactivePublisher
	hashes       map[[32]byte]struct{}
	hashC        chan [32]byte
	id           string
	config       Config
}

// New creates new Ledger if config is valid or returns error otherwise.
func New(
	config Config,
	trxCache transactionCacher,
	trx trxWriteReadMover,
	brwf blockReadWriteFinder,
	nsc nodeSyncRegister,
	sub blockchainLockSubscriber,
	ac addressChecker,
	vr signatureVerifier,
	log logger.Logger,
	blcPub blockReactivePublisher,
	trxIssuedPub trxIssuedReactivePublisher,
) (*Ledger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Ledger{
		id:           primitive.NewObjectID().Hex(),
		config:       config,
		hashC:        make(chan [32]byte, config.BlockTransactionsSize),
		hashes:       make(map[[32]byte]struct{}),
		trxCache:     trxCache,
		trx:          trx,
		nsc:          nsc,
		sub:          sub,
		brwf:         brwf,
		ac:           ac,
		vr:           vr,
		log:          log,
		blcPub:       blcPub,
		trxIssuedPub: trxIssuedPub,
	}, nil
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
// Run starts a goroutine and can be stopped by cancelling the context.
// It is non-blocking and concurrent safe.
func (l *Ledger) Run(ctx context.Context) error {
	if err := l.nsc.RegisterNode(ctx, l.id); err != nil {
		return fmt.Errorf("bookkeeper node [ %v ] failed, %w", l.id, err)
	}
	count, err := l.nsc.CountRegistered(ctx)
	if err != nil {
		return fmt.Errorf("bookkeeper node [ %v ] looking for registerd nodes failed, %w", l.id, err)
	}
	if count == 1 {
		if err := l.forgeTemporaryTrxs(ctx); err != nil {
			return fmt.Errorf("bookkeeper node [ %v ], forging temporary failed, %w", l.id, err)
		}
		l.log.Info(fmt.Sprintf("bookkeeper node [ %v ], forging temporary transactions finished", l.id))
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Duration(l.config.BlockWriteTimestamp) * time.Second)
		defer ticker.Stop()
		defer func() {
			ctxxx, cancelx := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelx()
			if err := l.nsc.UnregisterNode(ctxxx, l.id); err != nil {
				l.log.Fatal(err.Error())
			}
		}()
		for {
			select {
			case <-ctx.Done():
				if len(l.hashes) > 0 {
					if err := l.forge(ctx); err != nil {
						l.log.Error(fmt.Sprintf("cannot forge block on closing, %s", err))
					}
				}
				return
			case h := <-l.hashC:
				if _, ok := l.hashes[h]; !ok {
					l.hashes[h] = struct{}{}
					if len(l.hashes) == l.config.BlockTransactionsSize {
						if err := l.forge(ctx); err != nil {
							l.log.Error(fmt.Sprintf("cannot forge block on new transaction, %s", err))
						}
					}
				}
			case <-ticker.C:
				if len(l.hashes) > 0 {
					if err := l.forge(ctx); err != nil {
						l.log.Error(fmt.Sprintf("cannot forge block on next block tick, %s", err))
					}
				}
			}
		}
	}(ctx)

	return nil
}

// WriteIssuerSignedTransactionForReceiver validates issuer signature and writes a transaction to the repository for receiver.
func (l *Ledger) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	trx *transaction.Transaction,
) error {
	if err := l.validatePartiallyTransaction(ctx, trx); err != nil {
		return err
	}

	if err := l.trxCache.WriteIssuerSignedTransactionForReceiver(trx); err != nil {
		l.log.Error(fmt.Sprintf("bookkeeper cache failure, %s", err.Error()))
	}

	if err := l.trx.WriteIssuerSignedTransactionForReceiver(ctx, trx); err != nil {
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

	go l.trxCache.CleanSignedTransactions([]transaction.Transaction{*trx})

	if err := l.trx.MoveTransactionFromAwaitingToTemporary(ctx, trx); err != nil {
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
		trxs, err := l.trx.ReadTemporaryTransactions(ctx, 0, l.config.BlockTransactionsSize)
		if err != nil {
			return err
		}
		if len(trxs) == 0 {
			return nil
		}
		for _, trx := range trxs {
			l.hashes[trx.Hash] = struct{}{}
		}
		if err := l.forge(ctx); err != nil {
			return err
		}
		l.cleanHashes()
	}
}

func (l *Ledger) forge(ctx context.Context) error {
	defer l.cleanHashes()
	sync := newSync(l.id, l.nsc, l.sub)
	if err := sync.waitInQueueForLock(ctx); err != nil {
		return err
	}

	hashes := make([][32]byte, 0, len(l.hashes))
	for h := range l.hashes {
		hashes = append(hashes, h)
	}

	blcHash, err := l.savePublishNewBlock(ctx, hashes)
	if err != nil {
		return err
	}

	if err := l.trx.MoveTransactionsFromTemporaryToPermanent(ctx, blcHash, hashes); err != nil {
		return errors.Join(errMovingTransactions, err)
	}
	if err := sync.releaseLock(ctx); err != nil {
		return err
	}
	return nil
}

func (l *Ledger) cleanHashes() {
	l.hashes = make(map[[32]byte]struct{}, l.config.BlockTransactionsSize)
}

func (l *Ledger) savePublishNewBlock(ctx context.Context, hashes [][32]byte) ([32]byte, error) {
	h, idx, err := l.brwf.LastBlockHashIndex(ctx)
	if err != nil {
		return [32]byte{}, errors.Join(errSavingBlock, err)
	}

	idx++
	nb := block.New(l.config.Difficulty, idx, h, hashes)

	if err := l.brwf.WriteBlock(ctx, nb); err != nil {
		return [32]byte{}, errors.Join(errSavingBlock, err)
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
