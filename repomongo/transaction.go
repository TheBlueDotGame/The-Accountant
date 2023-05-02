package repomongo

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/transaction"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransactionAwaitingReceiverSignature represents transaction awaiting receiver signature.
// It is as well the entity of all issued transactions that has not been signed by receiver yet.
type TransactionAwaitingReceiverSignature struct {
	ID              any                     `json:"-"                bson:"_id,omitempty"    db:"id"`
	ReceiverAddress string                  `json:"receiver_address" bson:"receiver_address" db:"receiver_address"`
	IssuerAddress   string                  `json:"issuer_address"   bson:"issuer_address"   db:"issuer_address"`
	Transaction     transaction.Transaction `json:"transaction"      bson:"transaction"      db:"-"`
	TransactionHash [32]byte                `json:"transaction_hash" bson:"transaction_hash" db:"hash"`
}

// WriteTemporaryTransaction writes transaction to the temporary storage.
func (db DataBase) WriteTemporaryTransaction(ctx context.Context, trx *transaction.Transaction) error {
	trx.ID = primitive.NewObjectID()
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
	awaitingTrx := TransactionAwaitingReceiverSignature{
		ID:              primitive.NilObjectID,
		ReceiverAddress: receiverAddr,
		IssuerAddress:   trx.IssuerAddress,
		Transaction:     *trx,
		TransactionHash: trx.Hash,
	}
	_, err := db.inner.Collection(transactionsAwaitingReceiverCollection).InsertOne(ctx, awaitingTrx)
	return err
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given receiver address.
func (db DataBase) ReadAwaitingTransactionsByReceiver(ctx context.Context, address string) ([]transaction.Transaction, error) {
	var trxsAwaiting []TransactionAwaitingReceiverSignature
	curs, err := db.inner.Collection(transactionsAwaitingReceiverCollection).Find(ctx, bson.M{"receiver_address": address})
	if err != nil {
		return nil, err
	}

	if err := curs.All(ctx, &trxsAwaiting); err != nil {
		return nil, err
	}
	result := make([]transaction.Transaction, 0, len(trxsAwaiting))
	for _, awaitTrx := range trxsAwaiting {
		result = append(result, awaitTrx.Transaction)
	}

	return result, nil
}

// ReadAwaitingTransactionsByReceiver reads all transactions paired with given issuer address.
func (db DataBase) ReadAwaitingTransactionsByIssuer(ctx context.Context, address string) ([]transaction.Transaction, error) {
	var awaitTrxs []TransactionAwaitingReceiverSignature
	curs, err := db.inner.Collection(transactionsAwaitingReceiverCollection).Find(ctx, bson.M{"issuer_address": address})
	if err != nil {
		return nil, err
	}

	if err := curs.All(ctx, &awaitTrxs); err != nil {
		return nil, err
	}
	result := make([]transaction.Transaction, 0, len(awaitTrxs))
	for _, awaitTrx := range awaitTrxs {
		result = append(result, awaitTrx.Transaction)
	}

	return result, nil
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
		trx.ID = primitive.NewObjectID()
		if _, er := db.inner.Collection(transactionsPermanentCollection).InsertOne(ctx, trx); err != nil {
			err = errors.Join(er)
			continue
		}

		deleteHashes = append(deleteHashes, trx.Hash)
	}

	if _, er := db.inner.Collection(transactionsTemporaryCollection).
		DeleteMany(ctx, bson.M{"hash": bson.M{"$in": deleteHashes}}); err != nil {
		err = errors.Join(er)
	}

	return err
}

// ReadTemporaryTransactions reads transactions from the temporary storage.
func (db DataBase) ReadTemporaryTransactions(ctx context.Context) ([]transaction.Transaction, error) {
	var trxs []transaction.Transaction
	curs, err := db.inner.Collection(transactionsTemporaryCollection).Find(ctx, bson.M{})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	if err := curs.All(ctx, &trxs); err != nil {
		return nil, err
	}
	return trxs, nil
}