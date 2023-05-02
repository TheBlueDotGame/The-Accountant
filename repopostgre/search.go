package repopostgre

import "context"

// FindAddress finds address in the database.
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error) {
	if limit == 0 || limit > 1000 {
		limit = 1000
	}
	var addresses []string
	rows, err := db.inner.QueryContext(ctx, `SELECT public_key FROM addresses WHERE public_key LIKE ? LIMIT ?`, search, limit)
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
	tx, err := db.inner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, trx := range trxHash {
		if _, err := tx.ExecContext(ctx, `INSERT INTO transactionsInBlock (block_hash, transaction_hash) VALUES (?, ?)`, blockHash, trx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error) {
	var blockHash [32]byte
	if err := db.inner.QueryRowContext(ctx, `SELECT block_hash FROM transactionsInBlock WHERE transaction_hash = ?`, trxHash).Scan(&blockHash); err != nil {
		return [32]byte{}, err
	}
	return blockHash, nil
}
