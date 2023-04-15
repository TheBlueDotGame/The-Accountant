package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	migrationsCollection            = "migrations"
	addressesCollection             = "addresses"
	transactionsPermanentCollection = "transactionsPermanent"
	transactionsTemporaryCollection = "transactionsTemporary"
	blocksCollection                = "blocks"
)

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner mongo.Database
}

// Connect creates new connection to the playableassets repository and returns pointer to that user instance
func Connect(ctx context.Context, connStr, databaseName string) (*DataBase, error) {
	conn, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))
	if err != nil {
		return nil, err
	}

	ctxx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := conn.Ping(ctxx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &DataBase{*conn.Database(databaseName)}, nil
}

// Disconnect disconnects user from database
func (c DataBase) Disconnect(ctx context.Context) error {
	return c.inner.Client().Disconnect(ctx)
}
