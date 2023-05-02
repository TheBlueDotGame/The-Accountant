package repomongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Token holds information about unique token.
// Token is a way of proving to the REST API of the central server
// that the request is valid and comes from the client that is allowed to use the API.
type Token struct {
	ID             any    `json:"-"               bson:"_id,omitempty"   db:"id"`
	Token          string `json:"token"           bson:"token"           db:"token"`
	Valid          bool   `json:"valid"           bson:"valid"           db:"valid"`
	ExpirationDate int64  `json:"expiration_date" bson:"expiration_date" db:"expiration_date"`
}

// CheckToken checks if token exists in the database is valid and didn't expire.
func (db DataBase) CheckToken(ctx context.Context, token string) (bool, error) {
	var t Token
	if err := db.inner.Collection(tokensCollection).FindOne(ctx, bson.M{"token": token}).Decode(&t); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
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
func (db DataBase) WriteToken(ctx context.Context, token string, expirationDate int64) error {
	t := Token{
		ID:             primitive.NewObjectID(),
		Token:          token,
		Valid:          true,
		ExpirationDate: expirationDate,
	}
	if _, err := db.inner.Collection(tokensCollection).InsertOne(ctx, t); err != nil {
		return err
	}
	return nil
}

// InvalidateToken invalidates token.
func (db DataBase) InvalidateToken(ctx context.Context, token string) error {
	return db.inner.Collection(tokensCollection).
		FindOneAndUpdate(ctx, bson.M{"token": token}, bson.M{"$set": bson.M{"valid": false}}).Err()
}
