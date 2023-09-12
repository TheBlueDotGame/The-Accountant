//go:build integrations

package natsclient

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/stdoutwriter"
)

var (
	idx     uint64      = 111
	testBlk block.Block = block.Block{
		Index: idx,
		TrxHashes: [][32]byte{
			{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			{2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			{3, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			{4, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
		Hash:       [32]byte{100, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		PrevHash:   [32]byte{255, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		Timestamp:  uint64(time.Now().UnixMilli()),
		Nonce:      111,
		Difficulty: 12,
	}
)

func natsPubSubTestHelper(tb testing.TB) (*Publisher, *Subscriber) {
	cfg := Config{
		Address: "nats://127.0.0.1:4222",
		Name:    "integration-test-1",
		Token:   "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0=",
	}

	p, err := PublisherConnect(cfg)
	assert.NilError(tb, err)

	s, err := SubscriberConnect(cfg)
	assert.NilError(tb, err)

	return p, s
}

func TestPubSubCycle(t *testing.T) {
	p, s := natsPubSubTestHelper(t)

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{})

	err := p.PublishNewBlock(&testBlk)
	assert.NilError(t, err)

	call := func(blk *block.Block) {
		assert.Equal(t, testBlk.Index, blk.Index)
		assert.DeepEqual(t, testBlk.TrxHashes, blk.TrxHashes)
		assert.Equal(t, testBlk.Hash, blk.Hash)
		assert.Equal(t, testBlk.PrevHash, blk.PrevHash)
		assert.Equal(t, testBlk.Timestamp, blk.Timestamp)
		assert.Equal(t, testBlk.Nonce, blk.Nonce)
		assert.Equal(t, testBlk.Difficulty, blk.Difficulty)
	}
	err = s.SubscribeNewBlock(call, log)
	assert.NilError(t, err)

	err = p.Disconnect()
	assert.NilError(t, err)
	err = s.Disconnect()
	assert.NilError(t, err)
}

func TestSingleProducerMultipleConsumerPattern(t *testing.T) {
	p, s := natsPubSubTestHelper(t)

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{})

	err := p.PublishNewBlock(&testBlk)
	assert.NilError(t, err)

	for i := 0; i < 1000; i++ {
		call := func(blk *block.Block) {
			assert.Equal(t, testBlk.Index, blk.Index)
			assert.DeepEqual(t, testBlk.TrxHashes, blk.TrxHashes)
			assert.Equal(t, testBlk.Hash, blk.Hash)
			assert.Equal(t, testBlk.PrevHash, blk.PrevHash)
			assert.Equal(t, testBlk.Timestamp, blk.Timestamp)
			assert.Equal(t, testBlk.Nonce, blk.Nonce)
			assert.Equal(t, testBlk.Difficulty, blk.Difficulty)
		}
		err = s.SubscribeNewBlock(call, log)
		assert.NilError(t, err)
	}

	err = p.Disconnect()
	assert.NilError(t, err)
	err = s.Disconnect()
	assert.NilError(t, err)
}
