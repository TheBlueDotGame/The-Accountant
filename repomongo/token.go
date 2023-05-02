package repomongo

import (
	"context"
	"time"

	"github.com/bartossh/Computantis/token"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CheckToken checks if token exists in the database is valid and didn't expire.
func (db DataBase) CheckToken(ctx context.Context, tkn string) (bool, error) {
	var t token.Token
	if err := db.inner.Collection(tokensCollection).FindOne(ctx, bson.M{"token": tkn}).Decode(&t); err != nil {
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
func (db DataBase) WriteToken(ctx context.Context, tkn string, expirationDate int64) error {
	t := token.Token{
		ID:             primitive.NewObjectID(),
		Token:          tkn,
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
