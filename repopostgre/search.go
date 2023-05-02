package repopostgre

import "context"

// FindAddress finds address in the database.
func (db DataBase) FindAddress(ctx context.Context, search string, limit int) ([]string, error) {
	var addresses []string
	rows, err := db.inner.QueryContext(ctx, `SELECT public_key FROM addresses WHERE public_key LIKE ? LIMIT ?`, search, limit)
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
