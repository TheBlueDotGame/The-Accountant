package repo

import (
	"context"
	"errors"

	"github.com/bartossh/The-Accountant/transaction"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransactionAwaitingReceiver represents transaction awaiting receiver signature.
type TransactionAwaitingReceiver struct {
	ID              primitive.ObjectID      `json:"-"                bson:"_id,omitempty"`
	ReceiverAddress string                  `json:"receiver_address" bson:"receiver_address"`
	IssuerAddress   string                  `json:"issuer_address"   bson:"issuer_address"`
	Transaction     transaction.Transaction `json:"transaction"      bson:"transaction"`
	TransactionHash [32]byte                `json:"transaction_hash" bson:"transaction_hash"`
}

// WriteTemporaryTransaction writes transaction to the temporary storage.
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error {
	_, err := db.inner.Collection(transactionsTemporaryCollection).InsertOne(ctx, trx)
	return err
}

// RemoveAwaitingTransaction removes transaction from the awaiting transaction storage.
func (db DataBase) RemoveAwaitingTransaction(ctx context.Context, trxHash [32]byte) error {
	_, err := db.inner.Collection(transactionsAwaitingReceiverCollection).DeleteOne(ctx, bson.M{"transaction_hash": trxHash})
	return err
}

// WriteIssuerSignedTransactionForReceiver writes transaction to the awaiting transaction storage paired with given receiver.
func (db DataBase) WriteIssuerSignedTransactionForReceiver(
	ctx context.Context,
	receiverAddr string,
	trx *transaction.Transaction,
) error {
	awaitingTrx := TransactionAwaitingReceiver{
		ReceiverAddress: receiverAddr,
		Transaction:     *trx,
		TransactionHash: trx.Hash,
	}
	_, err := db.inner.Collection(transactionsAwaitingReceiverCollection).InsertOne(ctx, awaitingTrx)
	return err
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	var trxs []transaction.Transaction
	curs, err := db.inner.Collection(transactionsAwaitingReceiverCollection).Find(ctx, bson.M{"receiver_address": address})
	if err != nil {
		return nil, err
	}

	if err := curs.All(ctx, &trxs); err != nil {
		return nil, err
	}

	return trxs, nil
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given issuer address.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	var trxs []transaction.Transaction
	curs, err := db.inner.Collection(transactionsAwaitingReceiverCollection).Find(ctx, bson.M{"issuer_address": address})
	if err != nil {
		return nil, err
	}

	if err := curs.All(ctx, &trxs); err != nil {
		return nil, err
	}

	return trxs, nil
}

// MoveTransactionsFromTemporaryToPermanent moves transactions from temporary storage to permanent.
func (db DataBase) MoveTransactionsFromTemporaryToPermanent(ctx context.Context, hash [][32]byte) error {
	var err error
	var curs *mongo.Cursor
	curs, err = db.inner.Collection(transactionsTemporaryCollection).Find(ctx, bson.M{"hash": bson.M{"$in": hash}})
	if err != nil {
		return err
	}
	deleteHashes := make([][32]byte, 0, len(hash))
	for curs.Next(ctx) {
		var trx transaction.Transaction
		if err := curs.Decode(&trx); err != nil {
			return err
		}
		if _, err := db.inner.Collection(transactionsPermanentCollection).InsertOne(ctx, trx); err != nil {
			err = errors.Join(err)
			continue
		}

		deleteHashes = append(deleteHashes, trx.Hash)
	}

	if _, err := db.inner.Collection(transactionsTemporaryCollection).DeleteMany(ctx, bson.M{"hash": bson.M{"$in": deleteHashes}}); err != nil {
		err = errors.Join(err)
	}

	return err
}
