package repopostgre

import (
	"context"
	"errors"
)

const (
	suspended = "suspended"
	standard  = "standard"
	trusted   = "trusted"
	admin     = "admin"
)

// WriteAddress writes address to the database.
func (db DataBase) WriteAddress(ctx context.Context, addr string) error {
	_, err := db.inner.ExecContext(ctx, "INSERT INTO addresses(public_key, access_level) VALUES($1, $2)", addr, standard)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error) {
	var exists bool
	err := db.inner.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM addresses WHERE public_key = $1)", addr).Scan(&exists)
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
	rows, err := db.inner.QueryContext(ctx,
		"SELECT public_key FROM addresses WHERE public_key LIKE $1 AND access_level = $2 LIMIT $3", search, standard, limit)
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

func (db DataBase) readAddressAccessLevel(ctx context.Context, addr string) (string, error) {
	var accessLevel string
	err := db.inner.QueryRowContext(ctx,
		"SELECT access_level FROM addresses WHERE public_key = $1", addr).Scan(&accessLevel)
	if err != nil {
		return "", errors.Join(ErrSelectFailed, err)
	}
	return accessLevel, nil
}

// IsAddressAdmin checks if address has access level suspended.
func (db DataBase) IsAddressSuspended(ctx context.Context, addr string) (bool, error) {
	accessLevel, err := db.readAddressAccessLevel(ctx, addr)
	if err != nil {
		return false, err
	}
	return accessLevel == suspended, nil
}

// IsAddressStandard checks if address has access level standard.
func (db DataBase) IsAddressStandard(ctx context.Context, addr string) (bool, error) {
	accessLevel, err := db.readAddressAccessLevel(ctx, addr)
	if err != nil {
		return false, err
	}
	return accessLevel == standard, nil
}

// IsAddressTrusted checks if address has access level trusted.
func (db DataBase) IsAddressTrusted(ctx context.Context, addr string) (bool, error) {
	accessLevel, err := db.readAddressAccessLevel(ctx, addr)
	if err != nil {
		return false, err
	}
	return accessLevel == trusted, nil
}

// IsAddressAdmin checks if address has access level admin.
func (db DataBase) IsAddressAdmin(ctx context.Context, addr string) (bool, error) {
	accessLevel, err := db.readAddressAccessLevel(ctx, addr)
	if err != nil {
		return false, err
	}
	return accessLevel == admin, nil
}
