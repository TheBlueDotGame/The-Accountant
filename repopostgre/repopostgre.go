package repopostgre

import (
	"context"
	"fmt"

	"database/sql"

	"github.com/bartossh/Computantis/configuration"

	_ "github.com/lib/pq"
)

var (
	ErrInsertFailed = fmt.Errorf("insert failed")
	ErrRemoveFailed = fmt.Errorf("remove failed")
	ErrSelectFailed = fmt.Errorf("select failed")
	ErrMoveFailed   = fmt.Errorf("move failed")
	ErrScanFailed   = fmt.Errorf("scan failed")
)

// Database provides database access for read, write and delete of repository entities.
type DataBase struct {
	inner *sql.DB
}

// Connect creates new connection to the repository and returns pointer to the DataBase.
func Connect(ctx context.Context, cfg configuration.DBConfig) (*DataBase, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("%s/%s?sslmode=disable", cfg.ConnStr, cfg.DatabaseName))
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
