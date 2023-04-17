package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TransactionInBlock stores relation between Transaction and Block to which Transaction was added.
// It is tored for fast lookup only.
type TransactionInBlock struct {
	ID              primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	BlockHash       [32]byte           `json:"-" bson:"block_hash"`
	TransactionHash [32]byte           `json:"-" bson:"transaction_hash"`
}

// WrirteTransactionInBlock stores relation between Transaction and Block to which Transaction was added.
func (db DataBase) WriteTransactionsInBlock(ctx context.Context, blockHash [32]byte, trxHash [][32]byte) error {
	trxsInB := make([]any, 0, len(trxHash))
	for _, trx := range trxHash {
		trxsInB = append(trxsInB, TransactionInBlock{
			BlockHash:       blockHash,
			TransactionHash: trx,
		})
	}
	_, err := db.inner.Collection(transactionsInBlockCollection).InsertMany(ctx, trxsInB)
	return err
}

// FindTransactionInBlockHash finds Block hash in to which Transaction with given hash was added.
func (db DataBase) FindTransactionInBlockHash(ctx context.Context, trxHash [32]byte) ([32]byte, error) {
	var trx TransactionInBlock
	err := db.inner.Collection(transactionsInBlockCollection).
		FindOne(ctx, bson.M{"transaction_hash": trxHash}).
		Decode(&trx)
	return trx.BlockHash, err
}
