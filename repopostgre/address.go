package repopostgre

import (
	"context"
	"errors"
)

// WriteAddress writes address to the database.
func (db DataBase) WriteAddress(ctx context.Context, addr string) error {
	_, err := db.inner.ExecContext(ctx, `INSERT INTO addresses(public_key) VALUES($1)`, addr)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error) {
	var exists bool
	err := db.inner.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM addresses WHERE public_key = $1)`, addr).Scan(&exists)
	if err != nil {
		return false, errors.Join(ErrSelectFailed, err)
	}
	return exists, nil
}

// FindAddress finds address in the database.
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error) {
	if limit == 0 || limit > 1000 {
		limit = 1000
	}
	var addresses []string
	rows, err := db.inner.QueryContext(ctx, "SELECT public_key FROM addresses WHERE public_key LIKE $1 LIMIT $2", search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}
