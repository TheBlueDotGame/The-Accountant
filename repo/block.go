package repo

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/block"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LastBlock returns last block from the database.
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error) {
	var b []block.Block

	opts := options.Find().SetSort(bson.M{"index": -1}).SetLimit(1)

	curs, err := db.inner.Collection(blocksCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		return block.Block{}, err
	}

	if err := curs.All(ctx, &b); err != nil {
		return block.Block{}, err
	}

	if len(b) == 0 {
		return block.Block{}, errors.New("unreachable code when querying last block")
	}

	return b[0], nil
}

// ReadBlockByHash returns block with given hash.
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error) {
	var b block.Block
	if err := db.inner.Collection(blocksCollection).FindOne(ctx, bson.M{"hash": hash}).Decode(&b); err != nil {
		return block.Block{}, err
	}
	return b, nil
}

// WriteBlock writes block to the database.
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error {
	block.ID = primitive.NewObjectID()
	if _, err := db.inner.Collection(blocksCollection).InsertOne(ctx, block); err != nil {
		return err
	}
	return nil
}
