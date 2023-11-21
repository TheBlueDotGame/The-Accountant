package immunity

import (
	"context"
	"errors"
	"fmt"

	"github.com/bartossh/Computantis/src/transaction"
)

// TransactionAntibodyProvider describes the transaction analyzer that validates transaction inner data.
type TransactionAntibodyProvider interface {
	AnalyzeTransaction(ctx context.Context, trx *transaction.Transaction) error
}

// Lymphatic system uses the antibody cells to validate health of the transaction and blockchain.
// It contains all the necessary features to act on broken transaction or block.
// It has no build in analyzers and fully depends on supplied BlockAntibodyProviders and TransactionAntibodyProviders.
type LymphaticSystem struct {
	transactionAntibodiesMapping map[string][]string
	transactionAntibodies        map[string]TransactionAntibodyProvider
}

// New creates new LymphaticSystem.
func New() *LymphaticSystem {
	return &LymphaticSystem{
		transactionAntibodiesMapping: make(map[string][]string),
		transactionAntibodies:        make(map[string]TransactionAntibodyProvider),
	}
}

// AddTransactionAntibody ads transaction antibody to LymphaticSystem.
func (ls *LymphaticSystem) AddTransactionAntibody(name string, antibody TransactionAntibodyProvider) {
	ls.transactionAntibodies[name] = antibody
}

// AssignTransactionAntibodiesToSubject assigns antibodies to the transaction subject only if all antibodies exist.
func (ls *LymphaticSystem) AssignTransactionAntibodiesToSubject(subject string, antibodies []string) error {
	for _, name := range antibodies {
		if _, ok := ls.transactionAntibodies[name]; !ok {
			return fmt.Errorf("transaction antibody: [ %s ] doesn't exist", name)
		}
	}
	ls.transactionAntibodiesMapping[subject] = antibodies
	return nil
}

// TransactionsAntibodyAnalize maps transaction by the subject to corresponding antibodies to analyze.
func (ls *LymphaticSystem) TransactionsAntibodiesAnalize(ctx context.Context, trx *transaction.Transaction) error {
	antibodies, ok := ls.transactionAntibodiesMapping[trx.Subject]
	if !ok {
		return fmt.Errorf("subject: [ %s ] has no antibodies assigned", trx.Subject)
	}
	return ls.analyzeTransactionWithListedAntibodies(ctx, antibodies, trx)
}

func (ls *LymphaticSystem) analyzeTransactionWithListedAntibodies(ctx context.Context, antibodies []string, trx *transaction.Transaction) error {
	var err error
	for _, name := range antibodies {
		if antibody, ok := ls.transactionAntibodies[name]; ok {
			if errInner := antibody.AnalyzeTransaction(ctx, trx); errInner != nil {
				err = errors.Join(err, errInner)
			}
			continue
		}
		err = errors.Join(err, fmt.Errorf("transaction antibody: [ %s ] doesn't exist", name))
	}
	return err
}
