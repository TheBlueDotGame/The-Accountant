package repopostgre

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bartossh/Computantis/transaction"
)

// WriteTemporaryTransaction writes transaction to the temporary storage.
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error {
	timestamp := trx.CreatedAt.UnixMicro()
	_, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactionsTemporary(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature
			) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
		timestamp, trx.Hash[:], trx.IssuerAddress,
		trx.ReceiverAddress, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature)
	if err != nil {
		errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.
func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error {
	_, err := db.inner.ExecContext(ctx, "DELETE FROM transactionsAwaitingReceiver WHERE hash = $1", trxHash[:])
	if err != nil {
		errors.Join(ErrRemoveFailed, err)
	}
	return nil
}

// WriteIssuerSignedTransactionForReceiver writes transaction to the awaiting transaction storage paired with given receiver.
func (db DataBase) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	receiverAddr string,
	trx *transaction.Transaction,
) error {
	timestamp := trx.CreatedAt.UnixMicro()
	_, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactionsAwaitingReceiver(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature
			) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
		timestamp, trx.Hash[:], trx.IssuerAddress,
		receiverAddr, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM transactionsAwaitingReceiver WHERE receiver_address = $1", address)
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

// RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM transactionsAwaitingReceiver WHERE issuer_address = $1", address)
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

// MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent storage.
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error {
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

	for _, h := range hash {
		var trx transaction.Transaction
		var timestamp int64
		trxHash := make([]byte, 0, 32)
		err = tx.QueryRowContext(ctx, "SELECT * FROM transactionsTemporary WHERE hash = $1", h[:]).
			Scan(
				&trx.ID, &timestamp, &trxHash, &trx.IssuerAddress, &trx.ReceiverAddress,
				&trx.Subject, &trx.Data, &trx.IssuerSignature, &trx.ReceiverSignature,
			)
		if err != nil {
			return errors.Join(ErrSelectFailed, err)
		}
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO 
				transactionsPermanent(
					created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature
				) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`,
			timestamp, trxHash, trx.IssuerAddress,
			trx.ReceiverAddress, trx.Subject, trx.Data,
			trx.IssuerSignature, trx.ReceiverSignature)
		if err != nil {
			return errors.Join(ErrInsertFailed, err)
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM transactionsTemporary WHERE hash = $1", h[:])
		if err != nil {
			errors.Join(ErrRemoveFailed, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Join(ErrCommitFailed, err)
	}
	return nil
}

// ReadTemporaryTransactions reads all transactions from the temporary storage.
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM transactionsTemporary")
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
