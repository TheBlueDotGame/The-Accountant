package mongorepo

import (
	"context"

	"github.com/bartossh/Computantis/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WriteValidatorStatus writes validator status to the database.
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *validator.Status) error {
	_, err := db.inner.Collection(validatorStatusCollection).InsertOne(ctx, vs)
	return err
}

// ReadLastNValidatorStatuses reads last validator statuses from the database.
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]validator.Status, error) {
	var results []validator.Status
	opts := options.Find().SetSort(bson.M{"index": -1}).SetLimit(last)

	curs, err := db.inner.Collection(validatorStatusCollection).Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}

	if err := curs.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
