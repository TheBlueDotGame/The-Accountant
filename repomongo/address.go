package repomongo

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/address"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// WriteAddress writes unique address to the database.
func (db DataBase) WriteAddress(ctx context.Context, addr string) error {
	if addr == "" {
		return errors.New("public key is empty")
	}

	a := address.Address{
		ID:        primitive.NewObjectID(),
		PublicKey: addr,
	}

	if _, err := db.inner.Collection(addressesCollection).InsertOne(ctx, a); err != nil {
		return err
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
// Returns true if exists and error if database error different from ErrNoDocuments.
func (db DataBase) CheckAddressExists(ctx context.Context, addr string) (bool, error) {
	if addr == "" {
		return false, errors.New("public key is empty")
	}

	res := db.inner.Collection(addressesCollection).FindOne(ctx, bson.M{"public_key": addr})
	if res.Err() != nil {
		if errors.Is(res.Err(), mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, res.Err()
	}

	return true, nil
}
