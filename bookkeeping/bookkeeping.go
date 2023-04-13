package bookkeeping

import (
	"context"
	"errors"
	"time"

	"github.com/bartossh/The-Accountant/block"
	"github.com/bartossh/The-Accountant/transaction"
)

const hashChanSize = 100 // number of transactions in a block.

const blockWriteTimestamp = time.Minute * 5 // time between writes of a block to the blockchain repository.

const difficulty uint64 = 5

var (
	ErrTrxExistsInTheLadger     = errors.New("transaction is already in the ledger")
	ErrTrxExistsInTheBlockchain = errors.New("transaction is already in the blockchain")
	ErrAddressNotExists         = errors.New("address does not exist in the addresses repository")
	ErrBlockTxsCorrupted        = errors.New("all transaction failed, block corrupted")
)

type trxReader interface {
	ReadTransactionByHash(ctx context.Context, hash [32]byte) (*transaction.Transaction, error)
}

type trxWriter interface {
	WriteTransaction(ctx context.Context, trx *transaction.Transaction) error
}

type trxReadWriter interface {
	trxReader
	trxWriter
}

type blockReader interface {
	LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error)
}

type blockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

type blockReadWriter interface {
	blockReader
	blockWriter
}

type addressChecker interface {
	CheckAddressExists(ctx context.Context, address string) (bool, error)
}

type signatureVerifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// TxHashError is a transaction error on transaction hash.
type TxHashError struct {
	Hash [32]byte
	Err  error
}

// Ledger is a collection of ledger functionality to perform bookkeeping.
type Ledger struct {
	hashC  chan [32]byte
	hashes [][32]byte
	tx     trxReadWriter
	bc     blockReadWriter
	ac     addressChecker
	vr     signatureVerifier
}

// NewLedger creates new Ledger.
func NewLedger(
	bc blockReadWriter,
	tx trxReadWriter,
	ac addressChecker,
	vr signatureVerifier,
) *Ledger {
	return &Ledger{
		hashC: make(chan [32]byte, hashChanSize),
		tx:    tx,
		bc:    bc,
		ac:    ac,
		vr:    vr,
	}
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
func (l *Ledger) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if len(l.hashes) > 0 {
				l.saveBlock(ctx)
				l.cleanHashes()
			}
			break
		case h := <-l.hashC:
			l.hashes = append(l.hashes, h)
			if len(l.hashes) == hashChanSize {
				l.saveBlock(ctx)
				l.cleanHashes()
			}
		case <-time.After(blockWriteTimestamp):
			if len(l.hashes) > 0 {
				l.saveBlock(ctx)
				l.cleanHashes()
			}
		}
	}
}

// WriteTransaction validates and writes a transaction to the repsitory.
func (l *Ledger) WriteTransaction(ctx context.Context, tx *transaction.Transaction) error {
	if err := l.validateTx(ctx, tx); err != nil {
		return err
	}
	if err := l.tx.WriteTransaction(ctx, tx); err != nil {
		return err
	}

	l.hashC <- tx.Hash

	return nil
}

func (l *Ledger) cleanHashes() {
	l.hashes = make([][32]byte, 0, hashChanSize)
}

func (l *Ledger) saveBlock(ctx context.Context) error {
	h, idx, err := l.bc.LastBlockHashIndex(ctx)
	if err != nil {
		return err
	}

	nb := block.NewBlock(difficulty, idx, h, l.hashes)

	if err := l.bc.WriteBlock(ctx, nb); err != nil {
		return err
	}

	return nil
}

func (l *Ledger) validateTx(ctx context.Context, tx *transaction.Transaction) error {
	exists, err := l.ac.CheckAddressExists(ctx, tx.IssuerAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	exists, err = l.ac.CheckAddressExists(ctx, tx.ReceiverAddress)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAddressNotExists
	}

	if err := l.vr.Verify(tx.Data, tx.IssuerSignature, tx.Hash, tx.IssuerAddress); err != nil {
		return err
	}

	if err := l.vr.Verify(tx.Data, tx.IssuerSignature, tx.Hash, tx.IssuerAddress); err != nil {
		return err
	}

	return nil
}
