package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	migrationsCollection                   = "migrations"
	addressesCollection                    = "addresses"
	transactionsPermanentCollection        = "transactionsPermanent"
	transactionsTemporaryCollection        = "transactionsTemporary"
	transactionsAwaitingReceiverCollection = "transactionsAwaitingReceiver"
	blocksCollection                       = "blocks"
	transactionsInBlockCollection          = "transactionsInBlock"
	tokensCollection                       = "tokens"
	logsCollection                         = "logs"
	validatorStatusCollection              = "validatorStatus"
)

// Config contains configuration for the database.
type Config struct {
	ConnStr      string `yaml:"conn_str"`         // ConnStr is the connection string to the database.
	DatabaseName string `yaml:"database_name"`    // DatabaseName is the name of the database.
	Token        string `yaml:"token"`            // Token is the token that is used to confirm api clients access.
	TokenExpire  int64  `yaml:"token_expiration"` // TokenExpire is the number of seconds after which token expires.
}

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner mongo.Database
}

// Connect creates new connection to the repository and returns pointer to the DataBase.
func Connect(ctx context.Context, cfg Config) (*DataBase, error) {
	conn, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.ConnStr))
	if err != nil {
		return nil, err
	}

	ctxx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := conn.Ping(ctxx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &DataBase{*conn.Database(cfg.DatabaseName)}, nil
}

// Disconnect disconnects user from database
func (c DataBase) Disconnect(ctx context.Context) error {
	return c.inner.Client().Disconnect(ctx)
}
