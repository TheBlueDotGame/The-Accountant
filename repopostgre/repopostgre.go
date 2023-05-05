package repopostgre

import (
	"context"
	"fmt"

	"database/sql"

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
)

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner *sql.DB
}

// Connect creates new connection to the repository and returns pointer to the DataBase.
func Connect(ctx context.Context, conn, database string) (*DataBase, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("%s/%s?sslmode=disable", conn, database))
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
