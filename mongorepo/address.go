package mongorepo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Address holds information about unique PublicKey.
type Address struct {
	ID        any    `json:"-"          bson:"_id,omitempty" db:"id"`
	PublicKey string `json:"public_key" bson:"public_key"    db:"public_key"`
}

// WriteAddress writes unique address to the database.
func (db DataBase) WriteAddress(ctx context.Context, address string) error {
	if address == "" {
		return errors.New("public key is empty")
	}

	addr := Address{
		ID:        primitive.NewObjectID(),
		PublicKey: address,
	}

	if _, err := db.inner.Collection(addressesCollection).InsertOne(ctx, addr); err != nil {
		return err
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
// Returns true if exists and error if database error different from ErrNoDocuments.
func (db DataBase) CheckAddressExists(ctx context.Context, address string) (bool, error) {
	if address == "" {
		return false, errors.New("public key is empty")
	}

	res := db.inner.Collection(addressesCollection).FindOne(ctx, bson.M{"public_key": address})
	if res.Err() != nil {
		if errors.Is(res.Err(), mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, res.Err()
	}

	return true, nil
}
