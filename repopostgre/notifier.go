package repopostgre

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

const (
	ActionDelete = "DELETE"
	ActionInsert = "INSERT"
	ActionUpdate = "UPDATE"
)

// Notification represents notification from database.
type Notification struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   struct {
		ID        int  `json:"id"`
		Timestamp int  `json:"timestamp"`
		Locked    bool `json:"locked"`
	} `json:"data"`
}

// LockBlockchain locks blockchain for writing.
func (db DataBase) LockBlockchain(ctx context.Context) (bool, error) {
	var err error
	var tx *sql.Tx
	tx, err = db.inner.BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Join(ErrTrxBeginFailed, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, "LOCK TABLE blockchainLocks IN ACCESS EXCLUSIVE MODE")
	if err != nil {
		return false, errors.Join(ErrLockingBlockChainFailed, err)
	}

	var locked bool
	err = tx.QueryRowContext(ctx, "SELECT locked FROM blockchainLocks WHERE locked=true").Scan(&locked)
	if err != nil {
		switch errors.Is(sql.ErrNoRows, err) {
		case true:
		default:
			return false, errors.Join(ErrLockingBlockChainFailed, err)
		}
	}
	if locked {
		return false, errors.Join(ErrLockingBlockChainFailed, errors.New("blockchain is locked"))
	}

	timestamp := time.Now().UTC().UnixMicro()
	_, err = tx.ExecContext(ctx, "INSERT INTO blockchainLocks(timestamp, locked) VALUES($1, true)", timestamp)
	if err != nil {
		return false, errors.Join(ErrLockingBlockChainFailed, err)
	}

	err = tx.Commit()
	if err != nil {
		return false, errors.Join(ErrCommitFailed, err)
	}
	return true, nil
}

// UnlockBlockchain unlocks blockchain for writing.
func (db DataBase) UnlockBlockChain(ctx context.Context) (bool, error) {
	var err error
	var tx *sql.Tx
	tx, err = db.inner.BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Join(ErrTrxBeginFailed, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, "LOCK TABLE blockchainLocks IN ACCESS EXCLUSIVE MODE")
	if err != nil {
		return false, errors.Join(ErrLockingBlockChainFailed, err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM blockchainLocks WHERE locked=true")
	if err != nil {
		return false, errors.Join(ErrLockingBlockChainFailed, err)
	}
	err = tx.Commit()
	if err != nil {
		return false, errors.Join(ErrCommitFailed, err)
	}
	return true, nil
}

// SubscribeToLockBlockchainNotification listens for blockchain lock.
// To stop subscription, close channel.
func (db DataBase) SubscribeToLockBlockchainNotification(ctx context.Context, l *pq.Listener, c chan<- bool) {
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
				var notification Notification
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
				case ActionInsert:
					c <- notification.Data.Locked
				case ActionDelete:
					c <- false
				case ActionUpdate:
					c <- notification.Data.Locked
				}

			case <-ctx.Done():
				close(c)
				return
			}
		}
	}(ctx, l, c)
}
