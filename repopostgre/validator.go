package repopostgre

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/validator"
)

// WriteValidatorStatus writes validator status to the database.
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error {
	_, err := db.inner.ExecContext(ctx,
		`INSERT INTO 
			validator_status (index, valid, created_at) VALUES ($1, $2, $3)`,
		vs.Index, vs.Valid, vs.CreatedAt)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}

// ReadLastNValidatorStatuses reads last validator statuses from the database.
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error) {
	res, err := db.inner.QueryContext(ctx, "SELECT * FROM validator_status ORDER BY index DESC LIMIT $1", last)
	if err != nil {
		return nil, errors.Join(ErrSelectFailed, err)
	}

	var results []validator.Status
	for res.Next() {
		var vs validator.Status
		if err := res.Scan(&vs.Index, &vs.Valid, &vs.CreatedAt); err != nil {
			return nil, errors.Join(ErrScanFailed, err)
		}
		results = append(results, vs)
	}

	return results, nil
}
