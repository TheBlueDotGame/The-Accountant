package repomongo

import (
	"context"
	"fmt"
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
	eventsCollection                       = "events"
	nodesCollection                        = "nodes"
)

var (
	ErrAddingToLockQueueBlockChainFailed       = fmt.Errorf("adding to lock blockchain failed")
	ErrRemovingFromLockQueueBlockChainFailed   = fmt.Errorf("removing from lock blockchain failed")
	ErrListenFailed                            = fmt.Errorf("listen failed")
	ErrCheckingIsOnTopOfBlockchainsLocksFailed = fmt.Errorf("checking is on top of blockchains locks failed")
	ErrNodeRegisterFailed                      = fmt.Errorf("node register failed")
	ErrNodeUnregisterFailed                    = fmt.Errorf("node unregister failed")
	ErrNodeLookupFailed                        = fmt.Errorf("node lookup failed")
	ErrNodeRegisteredAddressesQueryFailed      = fmt.Errorf("node registered addresses query failed")
	ErrCursorFailed                            = fmt.Errorf("cursor failed")
)

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner mongo.Database
}

// Connect creates new connection to the repository and returns pointer to the DataBase.
func Connect(ctx context.Context, conn, database string) (*DataBase, error) {
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(conn))
	if err != nil {
		return nil, err
	}

	ctxx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := cli.Ping(ctxx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &DataBase{*cli.Database(database)}, nil
}

// Disconnect disconnects user from database
func (c DataBase) Disconnect(ctx context.Context) error {
	return c.inner.Client().Disconnect(ctx)
}

func (db DataBase) Ping(ctx context.Context) error {
	return db.inner.Client().Ping(ctx, readpref.Primary())
}
