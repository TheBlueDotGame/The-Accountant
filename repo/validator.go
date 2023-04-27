package repo

import (
	"context"
	"time"

	"github.com/bartossh/Computantis/block"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ValidatorStatus is a status of each received block by the validator.
// It keeps track of invalid blocks in case of blockchain corruption.
type ValidatorStatus struct {
	ID        primitive.ObjectID `json:"-"          bson:"_id,omitempty"`
	Index     int64              `json:"index"      bson:"index"`
	Block     block.Block        `json:"block"      bson:"block"`
	Valid     bool               `json:"valid"      bson:"valid"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// WriteValidatorStatus writes validator status to the database.
func (db DataBase) WriteValidatorStatus(ctx context.Context, vs *ValidatorStatus) error {
	_, err := db.inner.Collection(validatorStatusCollection).InsertOne(ctx, vs)
	return err
}

// ReadLastNValidatorStatuses reads last validator statuses from the database.
func (db DataBase) ReadLastNValidatorStatuses(ctx context.Context, last int64) ([]ValidatorStatus, error) {
	var results []ValidatorStatus
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
