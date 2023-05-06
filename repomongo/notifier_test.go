package repomongo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestNotifierCycle(t *testing.T) {
	nodeID := "1"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	godotenv.Load("../.env")
	user := os.Getenv("MONGO_DB_USER")
	passwd := os.Getenv("MONGO_DB_PASSWORD")
	dbName := os.Getenv("MONGO_DB_NAME")

	db, err := Connect(ctx, fmt.Sprintf("mongodb://%s:%s@localhost:27017/?authSource=admin&authMechanism=SCRAM-SHA-256&readPreference=primary&&ssl=false&directConnection=true", user, passwd), dbName)
	assert.Nil(t, err)

	err = db.Ping(ctx)
	assert.Nil(t, err)

	c := make(chan bool)
	db.SubscribeToLockBlockchainNotification(ctx, c, nodeID)

	err = db.AddToBlockchainLockQueue(ctx, nodeID)
	assert.Nil(t, err)

	v := <-c
	assert.True(t, v)

	ok, err := db.CheckIsOnTopOfBlockchainsLocks(ctx, nodeID)
	assert.Nil(t, err)
	assert.True(t, ok)

	err = db.RemoveFromBlockchainLocks(ctx, nodeID)
	assert.Nil(t, err)

	v = <-c
	assert.True(t, v)

	db.Disconnect(ctx)
}

func TestNotifierCycleManySubscribers(t *testing.T) {
	nodesNum := 50

	run := func(t *testing.T, nodeID int, infoC chan<- int) {
		nodeIDStr := fmt.Sprintf("node_%d", nodeID)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

		godotenv.Load("../.env")
		user := os.Getenv("MONGO_DB_USER")
		passwd := os.Getenv("MONGO_DB_PASSWORD")
		dbName := os.Getenv("MONGO_DB_NAME")

		db, err := Connect(ctx, fmt.Sprintf("mongodb://%s:%s@localhost:27017/?authSource=admin&authMechanism=SCRAM-SHA-256&readPreference=primary&&ssl=false&directConnection=true", user, passwd), dbName)
		assert.Nil(t, err)

		err = db.Ping(ctx)
		assert.Nil(t, err)

		c := make(chan bool)
		db.SubscribeToLockBlockchainNotification(ctx, c, nodeIDStr)

		err = db.AddToBlockchainLockQueue(ctx, nodeIDStr)
		assert.Nil(t, err)

		time.Sleep(time.Millisecond * 200)
		go func() {
			tc := time.NewTicker(time.Second * 10)
			defer tc.Stop()

			fin := func() {
				ok, err := db.CheckIsOnTopOfBlockchainsLocks(ctx, nodeIDStr)
				assert.Nil(t, err)
				if ok {
					err = db.RemoveFromBlockchainLocks(ctx, nodeIDStr)
					assert.Nil(t, err)
				}
				infoC <- nodeID
			}
		outer:
			for {
				select {
				case <-tc.C:
					t.Error("timeout")
					panic("timeout")
				case v := <-c:
					if v {
						fin()
						break outer
					}
				}
			}
			cancel()
			db.Disconnect(ctx)
		}()
	}

	infoC := make(chan int, nodesNum)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		pv := -1
		for v := range infoC {
			if v != pv+1 {
				t.Errorf("wrong order: %d, %d", pv, v)
			}
			pv = v
			if pv == nodesNum-1 {
				wg.Done()
				return
			}
		}
	}()

	for i := 0; i < nodesNum; i++ {
		t.Run(fmt.Sprintf("node_%d", i), func(t *testing.T) {
			run(t, i, infoC)
		})
	}

	wg.Wait()
}
