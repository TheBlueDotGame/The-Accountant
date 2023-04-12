package bookkeeping

import (
	"github.com/bartossh/The-Accountant/block"
	"github.com/bartossh/The-Accountant/transaction"
)

type transactonReader interface {
	ReadTransactionByHash(hash string) (*transaction.Transaction, error)
}

type transactonWriter interface {
	WriteTransaction(trx *transaction.Transaction) error
}

type transactionReadWriter interface {
	transactonReader
	transactonWriter
}

type blockReader interface {
	ReadLastNBlocks(n int) ([]block.Block, error)
	ReadBlocksFromIndex(idx uint64) ([]block.Block, error)
}

type blockWriter interface {
	WriteBlock(block block.Block) error
}

type blockReadWriter interface {
	blockReader
	blockWriter
}

// Ledger is a collection of ledger functionality to perform bookkeeping.
type Ledger struct {
	tx transactionReadWriter
	bc blockReadWriter
}

// NewLedger creates new Ledger.
func NewLedger(bc blockReadWriter, tx transactionReadWriter) *Ledger {
	return &Ledger{
		tx: tx,
		bc: bc,
	}
}
