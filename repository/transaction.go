package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bartossh/Computantis/transaction"
)

const (
	awaited   = "awaited"
	temporary = "temporary"
	permanent = "permanent"
	rejected  = "rejected"
)

// MoveTransactionsFromAwaitingToTemporary moves awaiting transaction marking it as temporary.
func (db DataBase) MoveTransactionsFromAwaitingToTemporary(ctx context.Context, trxHash [32]byte) error {
	_, err := db.inner.ExecContext(ctx, "UPDATE transactions SET status = $1 WHERE hash = $2", temporary, trxHash[:])
	if err != nil {
		errors.Join(ErrRemoveFailed, err)
	}
	return nil
}

// WriteIssuerSignedTransactionForReceiver writes transaction to the storage marking it as awaiting.
func (db DataBase) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	trx *transaction.Transaction,
) error {
	timestamp := trx.CreatedAt.UnixMicro()
	_, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactions(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature, status, block_hash
			) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		timestamp, trx.Hash[:], trx.IssuerAddress,
		trx.ReceiverAddress, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature,
		awaited, []byte{})
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx,
		`SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE receiver_address = $1 AND status = $2`, address, awaited)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var trxsAwaiting []transaction.Transaction
	for rows.Next() {
		var trx transaction.Transaction
		hash := make([]byte, 0, 32)
		var timestamp int64
		err := rows.Scan(
			&trx.ID, &timestamp, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trx.CreatedAt = time.UnixMicro(timestamp)
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}

// ReadAwaitingTransactionsByIssuer  reads all transactions paired with given issuer address.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx,
		`SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE issuer_address = $1 AND status = $2`, address, awaited)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var trxsAwaiting []transaction.Transaction
	var timestamp int64
	for rows.Next() {
		var trx transaction.Transaction
		hash := make([]byte, 0, 32)
		err := rows.Scan(
			&trx.ID, &timestamp, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trx.CreatedAt = time.UnixMicro(timestamp)
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}

// MoveTransactionsFromTemporaryToPermanent moves transactions by marking transactions with matching hash to be permanent
// and sets block hash field to referenced block hash.
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, blockHash [32]byte, hashes [][32]byte) error {
	var err error
	var tx *sql.Tx
	tx, err = db.inner.BeginTx(ctx, nil)
	if err != nil {
		return errors.Join(ErrTrxBeginFailed, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, h := range hashes {
		_, err := tx.ExecContext(
			ctx, "UPDATE transactions SET status = $1, block_hash = $2 WHERE hash = $3", permanent, blockHash[:], h[:])
		if err != nil {
			return errors.Join(ErrInsertFailed, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Join(ErrCommitFailed, err)
	}
	return nil
}

// ReadTemporaryTransactions reads all transactions that are marked as temporary.
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx,
		`SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE status = $1`, temporary)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var trxsAwaiting []transaction.Transaction
	for rows.Next() {
		var trx transaction.Transaction
		var timestamp int64
		hash := make([]byte, 0, 32)
		err := rows.Scan(
			&trx.ID, &timestamp, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trx.CreatedAt = time.UnixMicro(timestamp)
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}

// FindTransactionInBlockHash returns block hash in to which transaction with given hash was added.
// If transaction is not yet added to any block, empty hash is returned.
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error) {
	blockHash := make([]byte, 0, 32)
	if err := db.inner.QueryRowContext(ctx, "SELECT block_hash FROM transactions WHERE transaction_hash = $1", trxHash[:]).
		Scan(&blockHash); err != nil {
		return [32]byte{}, err
	}
	var bh [32]byte
	copy(bh[:], blockHash)
	return bh, nil
}
