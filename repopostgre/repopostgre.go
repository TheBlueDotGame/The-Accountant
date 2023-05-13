package repopostgre

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var (
	ErrInsertFailed                            = fmt.Errorf("insert failed")
	ErrRemoveFailed                            = fmt.Errorf("remove failed")
	ErrSelectFailed                            = fmt.Errorf("select failed")
	ErrMoveFailed                              = fmt.Errorf("move failed")
	ErrScanFailed                              = fmt.Errorf("scan failed")
	ErrUnmarshalFailed                         = fmt.Errorf("unmarshal failed")
	ErrCommitFailed                            = fmt.Errorf("transaction commit failed")
	ErrTrxBeginFailed                          = fmt.Errorf("transaction begin failed")
	ErrAddingToLockQueueBlockChainFailed       = fmt.Errorf("adding to lock blockchain failed")
	ErrRemovingFromLockQueueBlockChainFailed   = fmt.Errorf("removing from lock blockchain failed")
	ErrListenFailed                            = fmt.Errorf("listen failed")
	ErrCheckingIsOnTopOfBlockchainsLocksFailed = fmt.Errorf("checking is on top of blockchains locks failed")
	ErrNodeRegisterFailed                      = fmt.Errorf("node register failed")
	ErrNodeUnregisterFailed                    = fmt.Errorf("node unregister failed")
	ErrNodeLookupFailed                        = fmt.Errorf("node lookup failed")
	ErrNodeRegisteredAddressesQueryFailed      = fmt.Errorf("node registered addresses query failed")
)

// Config contains configuration for the database.
type DBConfig struct {
	ConnStr      string `yaml:"conn_str"`      // ConnStr is the connection string to the database.
	DatabaseName string `yaml:"database_name"` // DatabaseName is the name of the database.
}

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner *sql.DB
}

// Subscribe subscribes to the database events.
func Subscribe(ctx context.Context, cfg DBConfig) (Listener, error) {
	f := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			panic(err)
		}
	}
	lister, err := Listen(cfg.ConnStr, f)
	if err != nil {
		return Listener{}, err
	}
	return lister, nil
}

// Connect creates new connection to the repository and returns pointer to the DataBase.
func Connect(ctx context.Context, cfg DBConfig) (*DataBase, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("%s/%s?sslmode=disable", cfg.ConnStr, cfg.DatabaseName)) // TODO: enable sslmode=verify-full
	if err != nil {
		return nil, err
	}

	return &DataBase{inner: db}, nil
}

// Disconnect disconnects user from database
func (db DataBase) Disconnect(ctx context.Context) error {
	return db.inner.Close()
}

// Ping checks if the connection to the database is still alive.
func (db DataBase) Ping(ctx context.Context) error {
	return db.inner.PingContext(ctx)
}
