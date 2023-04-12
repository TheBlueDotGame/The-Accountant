package bookkeeping

import (
	"context"
	"errors"
	"time"

	"github.com/bartossh/The-Accountant/block"
	"github.com/bartossh/The-Accountant/transaction"
)

const blockTxsBufferSize = 100 // number of transactions in a block.

const blockWriteTimestamp = time.Minute * 5 // time between writes of a block to the blockchain repository.

var (
	ErrTrxExistsInTheLadger     = errors.New("transaction is already in the ledger")
	ErrTrxExistsInTheBlockchain = errors.New("transaction is already in the blockchain")
	ErrAddressNotExists         = errors.New("address does not exist in the addresses repository")
	ErrBlockTxsCorrupted        = errors.New("all transaction failed, block corrupted")
)

type trxReadChecker interface {
	ReadTransactionByHash(ctx context.Context, hash [32]byte) (*transaction.Transaction, error)
	CheckExistsByHash(ctx context.Context, hash [32]byte) (bool, error)
}

type trxWriter interface {
	WriteTransaction(ctx context.Context, trx *transaction.Transaction) error
}

type trxDeleter interface {
	DeleteTransacionByHash(ctx context.Context, hash [32]byte) error
}

type txrReadCheckWriteDeleter interface {
	trxReadChecker
	trxWriter
	trxDeleter
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
	waitingTxs       map[[32]byte]*transaction.Transaction
	writtenTxHashSub chan [32]byte
	failedTxHashSub  chan TxHashError
	txC              chan *transaction.Transaction
	tx               txrReadCheckWriteDeleter
	bc               blockReadWriter
	ac               addressChecker
	vr               signatureVerifier
}

// NewLedger creates new Ledger.
func NewLedger(
	bc blockReadWriter,
	tx txrReadCheckWriteDeleter,
	ac addressChecker,
	vr signatureVerifier,
) *Ledger {
	writtenTxHashSubSuccess := make(chan [32]byte, 1000)
	failedTxHashSubFailure := make(chan TxHashError, 1000)
	return &Ledger{
		waitingTxs:       make(map[[32]byte]*transaction.Transaction),
		writtenTxHashSub: writtenTxHashSubSuccess,
		failedTxHashSub:  failedTxHashSubFailure,
		txC:              make(chan *transaction.Transaction, blockTxsBufferSize),
		tx:               tx,
		bc:               bc,
		ac:               ac,
		vr:               vr,
	}
}

// SuccessSubscription provides channel of successfully written transactions hash.
func (l *Ledger) SuccessSubscription() <-chan [32]byte {
	return l.writtenTxHashSub
}

// FailureSubscription provides channel of fail to write transactions hash  with errors.
func (l *Ledger) FailureSubscription() <-chan TxHashError {
	return l.failedTxHashSub
}

// Run runs the Ladger engine that writes blocks to the blockchain repository.
func (l *Ledger) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if len(l.waitingTxs) > 0 {
				l.saveBlockAndTxs(ctx)
				l.waitingTxs = make(map[[32]byte]*transaction.Transaction)
			}
			break
		case tx := <-l.txC:
			if err := l.validateTx(ctx, tx); err != nil {
				l.failedTxHashSub <- TxHashError{
					tx.Hash,
					err,
				}
				return
			}
			l.waitingTxs[tx.Hash] = tx
			if len(l.waitingTxs) == blockTxsBufferSize {
				l.saveBlockAndTxs(ctx)
				l.waitingTxs = make(map[[32]byte]*transaction.Transaction)
			}
		case <-time.After(blockWriteTimestamp):
			if len(l.waitingTxs) > blockTxsBufferSize {
				l.saveBlockAndTxs(ctx)
				l.waitingTxs = make(map[[32]byte]*transaction.Transaction)
			}
		}
	}
}

// WriteTransaction writes transaction to the Ladger queue when it waits to be written to the blockchain.
func (l *Ledger) WriteTransaction(tx *transaction.Transaction) {
	l.txC <- tx
}

func (l *Ledger) saveBlockAndTxs(ctx context.Context) {
	txsHashes := make([][32]byte, 0, len(l.waitingTxs))
	txsSuccess := make([]*transaction.Transaction, 0, len(l.waitingTxs))
	txsFailure := make([]TxHashError, 0, len(l.waitingTxs))

	for _, tx := range l.waitingTxs {
		if err := l.tx.WriteTransaction(ctx, tx); err != nil {
			txsFailure = append(txsFailure, TxHashError{
				tx.Hash,
				err,
			})
		}
		txsSuccess = append(txsSuccess, tx)
		txsHashes = append(txsHashes, tx.Hash)
	}

	if len(txsFailure) > 0 {
		for _, fail := range txsFailure {
			l.failedTxHashSub <- fail
		}
	}

	if len(txsHashes) == 0 {
		return
	}

	h, idx, err := l.bc.LastBlockHashIndex(ctx)
	if err != nil {
		for _, txH := range txsHashes {
			l.tx.DeleteTransacionByHash(ctx, txH) // TODO: log if err and hash
		}
		return
	}

	nb := block.NewBlock(idx, h, txsHashes)

	if err := l.bc.WriteBlock(ctx, nb); err != nil {
		for _, txH := range txsHashes {
			l.tx.DeleteTransacionByHash(ctx, txH) // TODO: log err and hash
			l.failedTxHashSub <- TxHashError{
				Hash: txH,
				Err:  err,
			}
		}
		return
	}

	for _, txH := range txsHashes {
		l.writtenTxHashSub <- txH
	}
}

func (l *Ledger) validateTx(ctx context.Context, tx *transaction.Transaction) error {
	if _, ok := l.waitingTxs[tx.Hash]; ok {
		return ErrTrxExistsInTheLadger
	}

	exists, err := l.tx.CheckExistsByHash(ctx, tx.Hash)
	if err != nil {
		return err
	}

	if exists {
		return ErrTrxExistsInTheBlockchain
	}

	exists, err = l.ac.CheckAddressExists(ctx, tx.IssuerAddress)

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
