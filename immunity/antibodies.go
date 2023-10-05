package immunity

import (
	"context"
	"errors"
	"fmt"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/transaction"
)

// TransactionSizeAntibody checks if Transaction data are in approved size.
type TransactionSizeAntibody struct {
	min int
	max int
}

// NewTransactionSizeAntibody creates a new TransactionSizeAntibody if given parameters and correct
func NewTransactionSizeAntibody(min, max int) (*TransactionSizeAntibody, error) {
	if min > max {
		return nil, fmt.Errorf("min value of [ %v ] is bigger then max value of [ %v ]", min, max)
	}
	if min < 0 {
		return nil, fmt.Errorf("min value cannot be negative, got [ %v ]", min)
	}
	return &TransactionSizeAntibody{min: min, max: max}, nil
}

// AnalyzeTransaction implements TransactionAntibodyProvider.
// Validates if transaction data are in required size range.
func (tsa TransactionSizeAntibody) AnalyzeTransaction(_ context.Context, trx *transaction.Transaction) error {
	if trx == nil {
		return errors.New("cannot validate a nil transaction")
	}
	if len(trx.Data) < tsa.min {
		return fmt.Errorf("expected minimum transaction data size is [ %v ], got [ %v ]", tsa.min, len(trx.Data))
	}
	if len(trx.Data) > tsa.max {
		return fmt.Errorf("expected maximum transaction data size is [ %v ], got [ %v ]", tsa.max, len(trx.Data))
	}
	return nil
}

// BlockConsecutiveOrderAntibody checks if blocks are supplied in a consecutive order.
// It is naive a validation that does not perform hashing of the transactions and any other block calculations.
// It only accounts for proper order.
// Validates block hash order and index position order.
type BlockConsecutiveOrderAntibody struct {
	prevHash  [32]byte
	prevIndex uint64
	isFirst   bool
}

// NewBlockConsecutiveOrderAntibody creates a new BlockConsecutiveOrderAntibody.
// If blk is nil then BlockConsecutiveOrderAntibody assumes that first given block to analyze will be correct
// and will not be validated, but will be set as a reference for following blocks validation.
func NewBlockConsecutiveOrderAntibody(blk *block.Block) (*BlockConsecutiveOrderAntibody, error) {
	if blk == nil {
		return &BlockConsecutiveOrderAntibody{prevHash: [32]byte{}, prevIndex: 0, isFirst: true}, nil
	}
	if blk.Hash == [32]byte{} {
		return nil, fmt.Errorf("previous hash [ %v ] is empty", blk.Hash)
	}
	return &BlockConsecutiveOrderAntibody{prevHash: blk.Hash, prevIndex: blk.Index}, nil
}

// AnalyzeBlock implements BlockAntibodyProvider.
// Validates if block hashes are in consecutive order and if block indexes are in consecutive order.
// It is a naive validation without hash calculation from given transactions.
func (bcoa *BlockConsecutiveOrderAntibody) AnalyzeBlock(_ context.Context, blk *block.Block, _ []transaction.Transaction) error {
	if blk == nil {
		return errors.New("cannot validate a nil block")
	}
	if bcoa.isFirst {
		bcoa.isFirst = false
		bcoa.prevIndex = blk.Index
		bcoa.prevHash = blk.Hash
		return nil
	}
	if blk.Index != bcoa.prevIndex+1 {
		return fmt.Errorf("expected block index [ %v ], got [ %v ]", bcoa.prevIndex+1, blk.Index)
	}
	if blk.PrevHash != bcoa.prevHash {
		return fmt.Errorf("expected previous hash [ %v ], got [ %v ]", bcoa.prevHash, blk.PrevHash)
	}
	bcoa.prevIndex = blk.Index
	bcoa.prevHash = blk.Hash
	return nil
}
