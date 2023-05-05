package repopostgre

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestNotifierCycle(t *testing.T) {
	nodeID := "1"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	godotenv.Load("../.env")
	user := os.Getenv("POSTGRES_DB_USER")
	passwd := os.Getenv("POSTGRES_DB_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB_NAME")

	db, err := Connect(ctx, fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd), dbName)
	assert.Nil(t, err)

	err = db.Ping(ctx)
	assert.Nil(t, err)

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			t.Error(err)
		}
	}

	listener, err := Listen(fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd), reportProblem)
	assert.Nil(t, err)

	c := make(chan bool)
	listener.SubscribeToLockBlockchainNotification(ctx, c, nodeID)

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

	listener.Close()
}

func TestNotifierCycleManySubscribers(t *testing.T) {
	run := func(t *testing.T, nodeID string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		godotenv.Load("../.env")
		user := os.Getenv("POSTGRES_DB_USER")
		passwd := os.Getenv("POSTGRES_DB_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB_NAME")

		db, err := Connect(ctx, fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd), dbName)
		assert.Nil(t, err)

		err = db.Ping(ctx)
		assert.Nil(t, err)

		reportProblem := func(ev pq.ListenerEventType, err error) {
			if err != nil {
				t.Error(err)
			}
		}

		listener, err := Listen(fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd), reportProblem)
		assert.Nil(t, err)

		c := make(chan bool)
		listener.SubscribeToLockBlockchainNotification(ctx, c, nodeID)

		err = db.AddToBlockchainLockQueue(ctx, nodeID)
		assert.Nil(t, err)

		tc := time.NewTicker(time.Microsecond * 500)
		defer tc.Stop()

		fin := func() {
			ok, err := db.CheckIsOnTopOfBlockchainsLocks(ctx, nodeID)
			assert.Nil(t, err)
			if ok {
				err = db.RemoveFromBlockchainLocks(ctx, nodeID)
				assert.Nil(t, err)
			}
		}
	outer:
		for {
			select {
			case <-tc.C:
				fin()
				break outer
			case v := <-c:
				if v {
					fin()
					break outer
				}
			}
		}

		listener.Close()
	}

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("node_%d", i), func(t *testing.T) {
			go run(t, fmt.Sprintf("node_%d", i))
		})
	}

}
