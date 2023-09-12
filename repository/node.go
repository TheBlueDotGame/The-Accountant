package repository

import (
	"context"
	"errors"
)

// RegisterNode registers node in the database.
func (db DataBase) RegisterNode(ctx context.Context, n string) error {
	_, err := db.inner.ExecContext(ctx, "INSERT INTO nodes (node) VALUES ($1)", n)
	if err != nil {
		return errors.Join(ErrNodeRegisterFailed, err)
	}
	return nil
}

// UnregisterNode unregister node from the database.
func (db DataBase) UnregisterNode(ctx context.Context, n string) error {
	_, err := db.inner.ExecContext(ctx, "DELETE FROM nodes WHERE node = $1", n)
	if err != nil {
		return errors.Join(ErrNodeUnregisterFailed, err)
	}
	return nil
}

// CountRegistered counts registered nodes in the database.
func (db DataBase) CountRegistered(ctx context.Context) (int, error) {
	var count int
	err := db.inner.QueryRowContext(ctx, "SELECT COUNT(*) FROM nodes").Scan(&count)
	if err != nil {
		return 0, errors.Join(ErrNodeLookupFailed, err)
	}
	return count, nil
}
