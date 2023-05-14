package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

const (
	minReconnectInterval = 5 * time.Second
	maxReconnectInterval = 30 * time.Second
)

const (
	actionDelete = "DELETE"
	actionInsert = "INSERT"
	actionUpdate = "UPDATE"
)

type notification struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   struct {
		ID        int    `json:"id"`
		Timestamp int    `json:"timestamp"`
		Node      string `json:"node"`
	} `json:"data"`
}

// Listener wraps listener for notifications from database.
// Provides methods for listening and closing.
type Listener struct {
	inner *pq.Listener
}

// Listen creates Listener for notifications from database.
func Listen(conn string, report func(ev pq.ListenerEventType, err error)) (Listener, error) {
	listener := pq.NewListener(fmt.Sprintf("%s?sslmode=disable", conn), minReconnectInterval, maxReconnectInterval, report)
	err := listener.Listen("events")
	if err != nil {
		return Listener{}, errors.Join(ErrListenFailed, err)
	}
	return Listener{inner: listener}, err
}

// SubscribeToLockBlockchainNotification listens for blockchain lock.
// To stop subscription, close channel.
func (l Listener) SubscribeToLockBlockchainNotification(ctx context.Context, c chan<- bool, node string) {
	go func(ctx context.Context, l *pq.Listener, c chan<- bool) {
		for {
			select {
			case n := <-l.Notify:
				var prettyJSON bytes.Buffer
				err := json.Indent(&prettyJSON, []byte(n.Extra), "", "\t")
				if err != nil {
					fmt.Printf("failed to indent json: %v\n", err)
					continue
				}
				var notification notification
				err = json.Unmarshal(prettyJSON.Bytes(), &notification)
				if err != nil {
					fmt.Printf("failed to unmarshal json: %v\n", err)
					continue
				}
				if notification.Table != "blockchainlocks" {
					fmt.Println("table is not a blockchainlocks")
					continue
				}
				switch notification.Action {
				case actionInsert:
					if notification.Data.Node == node {
						c <- true
						continue
					}
					c <- false
				case actionDelete:
					c <- true // lock is removed, checking if moved in the queue is needed
				case actionUpdate:
					if notification.Data.Node == node {
						c <- true
						continue
					}
					c <- false
				}

			case <-ctx.Done():
				close(c)
				return
			}
		}
	}(ctx, l.inner, c)
}

// Close closes listener.
func (l Listener) Close() {
	l.inner.Close()
}

// AddToBlockchainLockQueue adds blockchain lock to queue.
func (db DataBase) AddToBlockchainLockQueue(ctx context.Context, nodeID string) error {
	ts := time.Now().UnixMicro()
	_, err := db.inner.ExecContext(ctx, "INSERT INTO blockchainLocks (timestamp, node) VALUES ($1, $2)", ts, nodeID)
	if err != nil {
		return errors.Join(ErrAddingToLockQueueBlockChainFailed, err)
	}
	return nil
}

// RemoveFromBlockchainLocks removes blockchain lock from queue.
func (db DataBase) RemoveFromBlockchainLocks(ctx context.Context, nodeID string) error {
	_, err := db.inner.ExecContext(ctx, "DELETE FROM blockchainLocks where node = $1", nodeID)
	if err != nil {
		return errors.Join(ErrRemovingFromLockQueueBlockChainFailed, err)
	}
	return nil
}

// CheckIsOnTopOfBlockchainsLocks checks if node is on top of blockchain locks queue.
func (db DataBase) CheckIsOnTopOfBlockchainsLocks(ctx context.Context, nodeID string) (bool, error) {
	var firstNodeID string
	err := db.inner.QueryRowContext(ctx, "SELECT node FROM blockchainLocks ORDER BY timestamp ASC LIMIT 1").Scan(&firstNodeID)
	if err != nil {
		return false, errors.Join(ErrCheckingIsOnTopOfBlockchainsLocksFailed, err)
	}
	if firstNodeID == nodeID {
		return true, nil
	}
	return false, nil
}
