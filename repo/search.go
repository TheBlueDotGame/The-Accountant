package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// FindAddress looks for matching address in the addresses repository and returns limited slice of matching addresses.
// If limit is set to 0 or above the 1000 which is maximum then serach is limited to 1000.
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error) {
	if limit == 0 || limit > 1000 {
		limit = 1000
	}
	var addresses []Address
	opts := options.Find().SetLimit(int64(limit))
	curs, err := db.inner.Collection(addressesCollection).Find(ctx, bson.M{"$text": bson.M{"$search": search}}, opts)
	if err != nil {
		return nil, err
	}
	if err := curs.All(ctx, &addresses); err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(addresses))

	for _, addr := range addresses {
		addrs = append(addrs, addr.PublicKey)
	}
	return addrs, nil
}
