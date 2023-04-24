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
)

const (
	minDifficulty = 1
	maxDifficulty = 124

	minBlockWriteTimestamp = time.Second
	maxBlockWriteTimestamp = time.Hour * 4

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

type TrxWriteReadMover interface {
	WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error
	WriteIssuerSignedTransactionForReceiver(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error
	MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error
	RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error
	ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error)
	ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error)
}

type BlockReader interface {
	LastBlockHashIndex() ([32]byte, uint64)
}

type BlockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

type BlockReadWriter interface {
	BlockReader
	BlockWriter
}

type AddressChecker interface {
	CheckAddressExists(ctx context.Context, address string) (bool, error)
}

type SignatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type BlockFinder interface {
	WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error
	FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error)
}

type Config struct {
	Difficulty            uint64 `json:"difficulty"              bson:"difficulty"              yaml:"difficulty"`
	BlockWriteTimestamp   uint64 `json:"block_write_timestamp"   bson:"block_write_timestamp"   yaml:"block_write_timestamp"`
	BlockTransactionsSize int    `json:"block_transactions_size" bson:"block_transactions_size" yaml:"block_transactions_size"`
}

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
type Ledger struct {
	config Config
	hashC  chan [32]byte
	hashes [][32]byte
	tx     TrxWriteReadMover
	bc     BlockReadWriter
	ac     AddressChecker
	vr     SignatureVerifier
	tf     BlockFinder
	log    logger.Logger
}

// NewLedger creates new Ledger if config is valid or returns error otherwise.
func NewLedger(
	config Config,
	bc BlockReadWriter,
	tx TrxWriteReadMover,
	ac AddressChecker,
	vr SignatureVerifier,
	tf BlockFinder,
	log logger.Logger,
) (*Ledger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Ledger{
		config: config,
		hashC:  make(chan [32]byte, config.BlockTransactionsSize),
		tx:     tx,
		bc:     bc,
		ac:     ac,
		vr:     vr,
		tf:     tf,
		log:    log,
	}, nil
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
// Run starts a goroutine and can be stopped by cancelling the context.
func (l *Ledger) Run(ctx context.Context) {
	if err := l.forgeTemporary(ctx); err != nil {
		l.log.Fatal(fmt.Sprintf("forging temporary failed: %s", err.Error()))
	}
	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Duration(l.config.BlockWriteTimestamp) * time.Second)
	outer:
		for {
			select {
			case <-ctx.Done():
				if len(l.hashes) > 0 {
					l.forge(ctx)
				}
				break outer
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
		ticker.Stop()
	}(ctx)
}

// WriteIssuerSignedTransactionForReceiver validates issuer signature and writes a transaction to the repository for receiver.
func (l *Ledger) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	receiverAddr string,
	trx *transaction.Transaction,
) error {
	if err := l.validatePartiallyTransaction(ctx, receiverAddr, trx); err != nil {
		return err
	}

	if err := l.tx.WriteIssuerSignedTransactionForReceiver(ctx, receiverAddr, trx); err != nil {
		return err
	}

	return nil
}

// WriteCandidateTransaction validates and writes a transaction to the repository.
// Transaction is not yet a part of the blockchain.
func (l *Ledger) WriteCandidateTransaction(ctx context.Context, trx *transaction.Transaction) error {
	if err := l.validateFullyTransaction(ctx, trx); err != nil {
		return err
	}
	if err := l.tx.WriteTemporaryTransaction(ctx, trx); err != nil {
		return err
	}

	if err := l.tx.RemoveAwaitingTransaction(ctx, trx.Hash); err != nil {
		return err
	}

	l.hashC <- trx.Hash

	return nil
}

func (l *Ledger) VerifySignature(message, signature []byte, hash [32]byte, address string) error {
	return l.vr.Verify(message, signature, hash, address)
}

func (l *Ledger) forgeTemporary(ctx context.Context) error {
	trxs, err := l.tx.ReadTemporaryTransactions(ctx)
	if err != nil {
		return err
	}
	for _, trx := range trxs {
		l.hashes = append(l.hashes, trx.Hash)
	}
	l.forge(ctx)
	return nil
}

func (l *Ledger) forge(ctx context.Context) {
	defer l.cleanHashes()
	blcHash, err := l.saveBlock(ctx)
	if err != nil {
		log.Fatal(fmt.Sprintf("error while saving block: %s", err.Error()))
		return
	}

	if err := l.tf.WriteTransactionsInBlock(ctx, blcHash, l.hashes); err != nil {
		log.Fatal(fmt.Sprintf("error while writing transactions in block [%v]: %s", blcHash, err.Error()))
	}

	if err := l.tx.MoveTransactionsFromTemporaryToPermanent(ctx, l.hashes); err != nil {
		log.Fatal(fmt.Sprintf("error while moving transactions from temporary to permanent: %s", err.Error()))
	}
}

func (l *Ledger) cleanHashes() {
	l.hashes = make([][32]byte, 0, l.config.BlockTransactionsSize)
}

func (l *Ledger) saveBlock(ctx context.Context) ([32]byte, error) {
	h, idx := l.bc.LastBlockHashIndex()
	idx++
	nb := block.NewBlock(l.config.Difficulty, idx, h, l.hashes)

	if err := l.bc.WriteBlock(ctx, nb); err != nil {
		return [32]byte{}, err
	}

	return nb.Hash, nil
}

func (l *Ledger) validatePartiallyTransaction(ctx context.Context, receiverAddr string, trx *transaction.Transaction) error {
	exists, err := l.ac.CheckAddressExists(ctx, trx.IssuerAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	exists, err = l.ac.CheckAddressExists(ctx, receiverAddr)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	if err := l.vr.Verify(trx.Data, trx.IssuerSignature, trx.Hash, trx.IssuerAddress); err != nil {
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

	if err := l.vr.Verify(trx.Data, trx.IssuerSignature, trx.Hash, trx.IssuerAddress); err != nil {
		return err
	}

	if err := l.vr.Verify(trx.Data, trx.IssuerSignature, trx.Hash, trx.IssuerAddress); err != nil {
		return err
	}

	return nil
}
