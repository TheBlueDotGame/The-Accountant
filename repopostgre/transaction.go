package repopostgre

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/transaction"
)

// WriteTemporaryTransaction writes transaction to the temporary storage.
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error {
	_, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactionsTemporary(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		trx.CreatedAt, trx.Hash[:], trx.IssuerAddress,
		trx.ReceiverAddress, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature)
	if err != nil {
		errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.
func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error {
	_, err := db.inner.ExecContext(ctx, `DELETE FROM transactionsAwaitingReceiver WHERE hash = ?`, trxHash[:])
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
	_, err := db.inner.ExecContext(
		ctx,
		`INSERT INTO 
			transactionsTemporary(
				created_at, hash, issuer_address, receiver_address, subject, data, issuer_signature, receiver_signature
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		trx.CreatedAt, trx.Hash[:], trx.IssuerAddress,
		receiverAddr, trx.Subject, trx.Data,
		trx.IssuerSignature, trx.ReceiverSignature)
	if err != nil {
		errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, `SELECT * FROM transactionsAwaitingReceiver WHERE receiver_address = ?`, address)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()
	var trxsAwaiting []transaction.Transaction
	for rows.Next() {
		var trx transaction.Transaction
		var hash [32]byte
		err := rows.Scan(
			&trx.CreatedAt, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}

// RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, `SELECT * FROM transactionsAwaitingIssuer WHERE issuer_address = ?`, address)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()
	var trxsAwaiting []transaction.Transaction
	for rows.Next() {
		var trx transaction.Transaction
		var hash [32]byte
		err := rows.Scan(
			&trx.CreatedAt, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}

// MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent storage.
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error {
	_, err := db.inner.ExecContext(ctx, `INSERT INTO transactionsPermanent SELECT * FROM transactionsTemporary WHERE hash = ?`, hash)
	if err != nil {
		return errors.Join(ErrMoveFailed, err)
	}
	_, err = db.inner.ExecContext(ctx, `DELETE FROM transactionsTemporary WHERE hash = ?`, hash)
	if err != nil {
		return errors.Join(ErrRemoveFailed, err)
	}
	return nil
}

// ReadTemporaryTransactions reads all transactions from the temporary storage.
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error) {
	rows, err := db.inner.QueryContext(ctx, `SELECT * FROM transactionsTemporary`)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()
	var trxsAwaiting []transaction.Transaction
	for rows.Next() {
		var trx transaction.Transaction
		var hash [32]byte
		err := rows.Scan(
			&trx.CreatedAt, &hash, &trx.IssuerAddress,
			&trx.ReceiverAddress, &trx.Subject, &trx.Data,
			&trx.IssuerSignature, &trx.ReceiverSignature)
		if err != nil {
			return nil, errors.Join(ErrSelectFailed, err)
		}
		copy(trx.Hash[:], hash[:])
		trxsAwaiting = append(trxsAwaiting, trx)
	}
	return trxsAwaiting, nil
}
