package repository

import (
	"context"
	"errors"
)

// RegisterNode registers node in the database.
func (db DataBase) RegisterNode(ctx context.Context, n, ws string) error {
	_, err := db.inner.ExecContext(ctx, "INSERT INTO nodes (node, websocket) VALUES ($1, $2)", n, ws)
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

// ReadAddresses reads registered nodes addresses from the database.
func (db DataBase) ReadRegisteredNodesAddresses(ctx context.Context) ([]string, error) {
	var addresses []string
	rows, err := db.inner.QueryContext(ctx, "SELECT websocket FROM nodes")
	if err != nil {
		errors.Join(ErrNodeRegisteredAddressesQueryFailed, err)
	}
	for rows.Next() {
		var address string
		err := rows.Scan(&address)
		if err != nil {
			return nil, errors.Join(ErrScanFailed, err)
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}
