package immunity

import (
	"context"
	"errors"
	"fmt"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/transaction"
)

// TransactionAntibodyProvider describes the transaction analyzer that validates transaction inner data.
type TransactionAntibodyProvider interface {
	AnalyzeTransaction(ctx context.Context, trx *transaction.Transaction) error
}

// BlockAntibodyProvider describes the block analyzer that validates block and related transactions.
type BlockAntibodyProvider interface {
	AnalyzeBlock(ctx context.Context, blk *block.Block, trxs []transaction.Transaction) error
}

// Lymphatic system uses the antibody cells to validate health of the transaction and blockchain.
// It contains all the necessary features to act on broken transaction or block.
// It has no build in analyzers and fully depends on supplied BlockAntibodyProviders and TransactionAntibodyProviders.
type LymphaticSystem struct {
	blockAntibodiesLevels        map[byte][]string
	transactionAntibodiesMapping map[string][]string
	blockAntibodies              map[string]BlockAntibodyProvider
	transactionAntibodies        map[string]TransactionAntibodyProvider
}

// New creates new LymphaticSystem.
func New() *LymphaticSystem {
	return &LymphaticSystem{
		blockAntibodiesLevels:        make(map[byte][]string),
		transactionAntibodiesMapping: make(map[string][]string),
		blockAntibodies:              make(map[string]BlockAntibodyProvider),
		transactionAntibodies:        make(map[string]TransactionAntibodyProvider),
	}
}

// AddTransactionAntibody ads transaction antibody to LymphaticSystem.
func (ls *LymphaticSystem) AddTransactionAntibody(name string, antibody TransactionAntibodyProvider) {
	ls.transactionAntibodies[name] = antibody
}

// AddBlockAntibody ads block antibody to LymphaticSystem.
func (ls *LymphaticSystem) AddBlockAntibody(name string, antibody BlockAntibodyProvider) {
	ls.blockAntibodies[name] = antibody
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

// AssignBlockAntibodiesToLevel assigns antibodies to the block level only if all antibodies exist.
func (ls *LymphaticSystem) AssignBlockAntibodiesToLevel(level byte, antibodies []string) error {
	for _, name := range antibodies {
		if _, ok := ls.blockAntibodies[name]; !ok {
			return fmt.Errorf("block antibody: [ %s ] doesn't exist", name)
		}
	}
	ls.blockAntibodiesLevels[level] = antibodies
	return nil
}

// BlockAntibodiesAnalyze maps level to antibodies to analyze the block.
func (ls *LymphaticSystem) BlockAntibodiesAnalyze(ctx context.Context, level byte, blk *block.Block, trxs []transaction.Transaction) error {
	antibodies, ok := ls.blockAntibodiesLevels[level]
	if !ok {
		return fmt.Errorf("level: [ %v ] has no antibodies assigned", level)
	}
	return ls.analyzeBlockWithListedAntibodies(ctx, antibodies, blk, trxs)
}

func (ls *LymphaticSystem) analyzeBlockWithListedAntibodies(ctx context.Context, antibodies []string, blk *block.Block, trxs []transaction.Transaction) error {
	var err error
	for _, name := range antibodies {
		if antibody, ok := ls.blockAntibodies[name]; ok {
			if errInner := antibody.AnalyzeBlock(ctx, blk, trxs); errInner != nil {
				err = errors.Join(err, errInner)
			}
			continue
		}
		err = errors.Join(err, fmt.Errorf("block antibody [ %s ] doesn't exist", name))
	}
	return err
}
