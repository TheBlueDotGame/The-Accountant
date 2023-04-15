package repo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Address holds information about unique PublicKey.
type Address struct {
	ID        primitive.ObjectID `json:"-"          bson:"_id,omitempty"`
	PublicKey string             `json:"public_key" bson:"public_key"`
}

// WriteAddress writes unique address to the database.
func (db DataBase) WriteAddress(ctx context.Context, address Address) error {
	address.ID = primitive.NilObjectID
	if address.PublicKey == "" {
		return errors.New("public key is empty")
	}

	if _, err := db.inner.Collection(addressesCollection).InsertOne(ctx, address); err != nil {
		return err
	}
	return nil
}

// CheckAddressExists checks if address exists in the database.
func (db DataBase) CheckAddressExists(ctx context.Context, address string) (bool, error) {
	if address == "" {
		return false, errors.New("public key is empty")
	}

	res := db.inner.Collection(addressesCollection).FindOne(ctx, bson.M{"public_key": address})
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, res.Err()
	}

	return true, nil
}
