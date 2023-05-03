package repopostgre

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// FindAddress finds address in the database.
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error) {
	if limit == 0 || limit > 1000 {
		limit = 1000
	}
	var addresses []string
	rows, err := db.inner.QueryContext(ctx, "SELECT public_key FROM addresses WHERE public_key LIKE $1 LIMIT $2", search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// WriteTransactionsInBlock stores relation between Transaction and Block to which Transaction was added.
func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error {
	var err error
	var tx *sql.Tx
	tx, err = db.inner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, trx := range trxHash {
		trxToWrite := make([][]byte, 0, len(trx))
		for _, b := range trx {
			trxToWrite = append(trxToWrite, []byte{b})
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO transactionsInBlock (block_hash, transaction_hash) VALUES ($1, $2)", blockHash[:], pq.Array(trxToWrite))
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Join(ErrCommitFailed, err)
	}
	return nil
}

// FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error) {
	blockHash := make([]byte, 0, 32)
	if err := db.inner.QueryRowContext(ctx, "SELECT block_hash FROM transactionsInBlock WHERE transaction_hash = $1", trxHash[:]).
		Scan(&blockHash); err != nil {
		return [32]byte{}, err
	}
	var bh [32]byte
	copy(bh[:], blockHash)
	return bh, nil
}
