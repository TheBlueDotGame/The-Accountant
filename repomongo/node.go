package repomongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

// RegisterNode registers node in the database.
func (db DataBase) RegisterNode(ctx context.Context, n, ws string) error {
	_, err := db.inner.Collection(nodesCollection).InsertOne(ctx, bson.M{"node": n, "websocket": ws})
	if err != nil {
		return errors.Join(ErrNodeRegisterFailed, err)
	}
	return nil
}

// UnregisterNode unregister node from the database.
func (db DataBase) UnregisterNode(ctx context.Context, n string) error {
	_, err := db.inner.Collection(nodesCollection).DeleteOne(ctx, bson.M{"node": n})
	if err != nil {
		return errors.Join(ErrNodeUnregisterFailed, err)
	}
	return nil
}

// CountRegistered counts registered nodes in the database.
func (db DataBase) CountRegistered(ctx context.Context) (int, error) {
	count, err := db.inner.Collection(nodesCollection).CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, errors.Join(ErrNodeLookupFailed, err)
	}
	return int(count), nil
}

// ReadAddresses reads registered nodes addresses from the database.
func (db DataBase) ReadRegisteredNodesAddresses(ctx context.Context) ([]string, error) {
	var addresses []string
	cur, err := db.inner.Collection(nodesCollection).Find(ctx, bson.M{})
	if err != nil {
		errors.Join(ErrNodeRegisteredAddressesQueryFailed, err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var register struct {
			Node      string `bson:"node"`
			WebSocket string `bson:"websocket"`
		}
		err := cur.Decode(&register)
		if err != nil {
			return nil, errors.Join(ErrCursorFailed, err)
		}
		addresses = append(addresses, register.WebSocket)
	}
	return addresses, nil
}
