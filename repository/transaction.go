package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/transaction"
)

const (
	awaited   = "awaited"
	temporary = "temporary"
	permanent = "permanent"
	rejected  = "rejected"
)

// MoveTransactionFromAwaitingToTemporary moves awaiting transaction marking it as temporary.
func (db DataBase) MoveTransactionFromAwaitingToTemporary(ctx context.Context, trx *transaction.Transaction) error {
	if _, err := db.inner.ExecContext(
		ctx,
		"UPDATE transactions SET status = $1, receiver_signature = $2 WHERE hash = $3 AND status = $4",
		temporary, trx.ReceiverSignature, trx.Hash[:], awaited,
	); err != nil {
		return errors.Join(ErrUpdateFailed, err)
	}
	return nil
}

// WriteIssuerSignedTransactionForReceiver writes transaction to the storage marking it as awaiting.
func (db DataBase) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	trx *transaction.Transaction,
) error {
	timestamp := trx.CreatedAt.UnixMicro()
	if _, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactions(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature, status, block_hash
			) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		timestamp, trx.Hash[:], trx.IssuerAddress,
		trx.ReceiverAddress, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature,
		awaited, []byte{}); err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadAwaitingTransactionsByReceiver reads up to the limit transactions paired with given receiver address.
// Upper limit of read all is MaxLimit constant.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	return db.readTransactionsByStatusPagginate(ctx, address, "receiver_address", awaited, 0, 0)
}

// ReadAwaitingTransactionsByIssuer reads up to the limit awaiting transactions paired with given issuer address.
// Upper limit of read all is MaxLimit constant.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	return db.readTransactionsByStatusPagginate(ctx, address, "issuer_address", awaited, 0, 0)
}

// ReadRejectedTransactionsPagginate reads rejected transactions with pagination.
func (db DataBase) ReadRejectedTransactionsPagginate(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error) {
	issuerTrx, err := db.readTransactionsByStatusPagginate(ctx, address, "issuer_address", rejected, offset, limit) // TODO: refactor this to make sense of pagination
	if err != nil {
		return nil, err
	}
	receiverTrx, err := db.readTransactionsByStatusPagginate(ctx, address, "receiver_address", rejected, offset, limit)
	if err != nil {
		return nil, err
	}
	return append(issuerTrx, receiverTrx...), nil
}

// ReadApprovedTransactions reads the approved transactions with pagination.
func (db DataBase) ReadApprovedTransactions(ctx context.Context, address string, offset, limit int) ([]transaction.Transaction, error) {
	issuerTrx, err := db.readTransactionsByStatusPagginate(ctx, address, "issuer_address", permanent, offset, limit) // TODO: refactor that to make sense of pagination
	if err != nil {
		return nil, err
	}
	receiverTrx, err := db.readTransactionsByStatusPagginate(ctx, address, "receiver_address", permanent, offset, limit)
	if err != nil {
		return nil, err
	}
	return append(issuerTrx, receiverTrx...), nil
}

// ReadTemporaryTransactions reads transactions that are marked as temporary with offset and limit.
func (db DataBase) ReadTemporaryTransactions(ctx context.Context, offset, limit int) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx,
		`SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE status = $1 ORDER by id DESC LIMIT $2 OFFSET $3`, temporary, limit, offset)
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
			ctx, "UPDATE transactions SET status = $1, block_hash = $2 WHERE hash = $3 AND status = $4",
			permanent, blockHash[:], h[:], temporary)
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

// RejectTransactions rejects transactions addressed to the receiver address.
func (db DataBase) RejectTransactions(ctx context.Context, receiver string, trxs []transaction.Transaction) error {
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

	for _, trx := range trxs {
		_, err := tx.ExecContext(
			ctx,
			"UPDATE transactions SET status = $1 WHERE hash = $2 AND receiver_address = $3",
			rejected, trx.Hash[:], receiver)
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

func (db DataBase) readTransactionsByStatusPagginate(
	ctx context.Context, address, addressColumn, status string, offset, limit int,
) ([]transaction.Transaction, error) {
	if limit > MaxLimit || limit == 0 {
		limit = MaxLimit
	}

	var query string
	switch addressColumn {
	case "issuer_address":
		query = `SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE issuer_address = $1 AND status = $2 ORDER BY id DESC, created_at DESC LIMIT $3 OFFSET $4`
	case "receiver_address":
		query = `SELECT id, created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature 
			FROM transactions WHERE receiver_address = $1 AND status = $2 ORDER BY id DESC, created_at DESC LIMIT $3 OFFSET $4`
	default:
		return nil, fmt.Errorf("unknown address colummn: %s", addressColumn)
	}

	rows, err := db.inner.QueryContext(ctx, query, address, status, limit, offset)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var trxs []transaction.Transaction
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
		trxs = append(trxs, trx)
	}
	return trxs, nil
}
