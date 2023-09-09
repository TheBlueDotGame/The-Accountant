package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bartossh/Computantis/helperserver"
)

// WriteValidatorStatus writes validator status to the database.
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *helperserver.Status) error {
	timestamp := vs.CreatedAt.UnixMicro()
	_, err := db.inner.ExecContext(ctx,
		`INSERT INTO 
			validator_status (index, valid, created_at) VALUES ($1, $2, $3)`,
		vs.Index, vs.Valid, timestamp)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadLastNValidatorStatuses reads last validator statuses from the database.
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]helperserver.Status, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM validator_status ORDER BY index DESC LIMIT $1", last)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var results []helperserver.Status
	for rows.Next() {
		var vs helperserver.Status
		var timestamp int64
		if err := rows.Scan(&vs.ID, &vs.Index, &vs.Valid, &timestamp); err != nil {
			return nil, errors.Join(ErrScanFailed, err)
		}
		vs.CreatedAt = time.UnixMicro(timestamp)
		results = append(results, vs)
	}

	return results, nil
}
