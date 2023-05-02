package repopostgre

import (
	"context"
	"errors"
)

// WriteAddress writes address to the database.
func (db DataBase) WriteAddress(ctx context.Context, addr string) error {
	_, err := db.inner.ExecContext(ctx, `INSERT INTO addresses(public_key) VALUES(?)`, addr)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error) {
	var exists bool
	err := db.inner.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM addresses WHERE public_key = ?)`, addr).Scan(&exists)
	if err != nil {
		return false, errors.Join(ErrSelectFailed, err)
	}
	return exists, nil
}
