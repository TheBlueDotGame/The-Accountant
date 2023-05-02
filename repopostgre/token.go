package repopostgre

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bartossh/Computantis/token"

	"time"
)

// CheckToken checks if token exists in the database is valid and didn't expire.
func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error) {
	var t token.Token
	if err := db.inner.QueryRowContext(ctx,
		`SELECT token, valid, expiration_date 
			FROM tokens WHERE token = ?`, tkn).
		Scan(&t.Token, &t.Valid, &t.ExpirationDate); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Join(ErrSelectFailed, err)
	}
	if !t.Valid {
		return false, nil
	}
	if t.ExpirationDate < time.Now().UnixNano() {
		return false, nil
	}
	return true, nil
}

// WriteToken writes unique token to the database.
func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error {
	if _, err := db.inner.ExecContext(ctx,
		`INSERT INTO tokens (token, valid, expiration_date) 
			VALUES (?, ?, ?)`, tkn, true, expirationDate); err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// InvalidateToken invalidates token.
func (db DataBase) InvalidateToken(ctx context.Context, token string) error {
	if _, err := db.inner.ExecContext(ctx,
		`UPDATE tokens SET valid = ? WHERE token = ?`, false, token); err != nil {
		return errors.Join(ErrRemoveFailed, err)
	}
	return nil
}
