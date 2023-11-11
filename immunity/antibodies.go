package immunity

import (
	"context"
	"errors"
	"fmt"

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
