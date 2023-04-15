package repo

import (
	"context"
	"errors"

	"github.com/bartossh/The-Accountant/transaction"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// WriteTemporaryTransaction writes transaction to the temporary storage.
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error {
	if _, err := db.inner.Collection(transactionsTemporaryCollection).InsertOne(ctx, trx); err != nil {
		return err
	}
	return nil
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
