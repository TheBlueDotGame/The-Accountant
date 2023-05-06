package repomongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type event struct {
	ID        primitive.ObjectID `json:"id"        bson:"_id,omitempty"`
	Timestamp int64              `json:"timestamp" bson:"timestamp"`
	Node      string             `json:"node"      bson:"node"`
}

// SubscribeToLockBlockchainNotification listens for blockchain lock.
// To stop subscription, close channel.
// This is fake subscriber, it isn't using change watcher as this requires replica set.
func (db DataBase) SubscribeToLockBlockchainNotification(ctx context.Context, c chan<- bool, node string) {
	go func(ctx context.Context, c chan<- bool) {
		var memorizedCount int64
		opts := options.Count().SetHint("_id_")
		var counterNoSignal int64
		tc := time.NewTicker(time.Microsecond * 100)
		defer tc.Stop()
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tc.C:
				count, err := db.inner.Collection(eventsCollection).CountDocuments(ctx, bson.D{}, opts)
				if err != nil {
					continue
				}
				counterNoSignal++
				if memorizedCount != count || counterNoSignal > 5 {
					c <- true
					counterNoSignal = 0
				}
				memorizedCount = count
			}
		}
	}(ctx, c)
}

// AddToBlockchainLockQueue adds blockchain lock to queue.
func (db DataBase) AddToBlockchainLockQueue(ctx context.Context, nodeID string) error {
	if _, err := db.inner.Collection(eventsCollection).InsertOne(ctx, event{Timestamp: time.Now().UnixMicro(), Node: nodeID}); err != nil {
		return errors.Join(ErrAddingToLockQueueBlockChainFailed, err)
	}
	return nil
}

// RemoveFromBlockchainLocks removes blockchain lock from queue.
func (db DataBase) RemoveFromBlockchainLocks(ctx context.Context, nodeID string) error {
	if _, err := db.inner.Collection(eventsCollection).DeleteOne(ctx, bson.M{"node": nodeID}); err != nil {
		return errors.Join(ErrRemovingFromLockQueueBlockChainFailed, err)
	}
	return nil
}

// CheckIsOnTopOfBlockchainsLocks checks if node is on top of blockchain locks queue.
func (db DataBase) CheckIsOnTopOfBlockchainsLocks(ctx context.Context, nodeID string) (bool, error) {
	sort := bson.M{"timestamp": 1}
	cur, err := db.inner.Collection(eventsCollection).Find(ctx, bson.M{}, &options.FindOptions{Sort: sort})
	if err != nil {
		return false, errors.Join(ErrCheckingIsOnTopOfBlockchainsLocksFailed, err)
	}
	defer cur.Close(ctx)

	if cur.Next(ctx) {
		var e event
		if err := cur.Decode(&e); err != nil {
			return false, errors.Join(ErrCheckingIsOnTopOfBlockchainsLocksFailed, err)
		}
		return e.Node == nodeID, nil
	}
	return false, ErrCheckingIsOnTopOfBlockchainsLocksFailed
}
