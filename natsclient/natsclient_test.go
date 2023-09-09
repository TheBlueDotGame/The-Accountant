package natsclient

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/stdoutwriter"
)

func TestPubSubCycle(t *testing.T) {
	var idx uint64 = 111
	cfg := Config{
		Address: "nats://127.0.0.1:4222",
		Name:    "integration-test-1",
		Token:   "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0=",
	}
	testBlk := block.Block{
		Index: idx,
	}

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{})

	p, err := PublisherConnect(cfg)
	assert.Nil(t, err)

	s, err := SubscriberConnect(cfg)
	assert.Nil(t, err)

	err = p.PublishNewBlock(testBlk)
	assert.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	call := func(blk *block.Block) {
		assert.Equal(t, idx, blk.Index)
		wg.Done()
	}
	err = s.SubscribeNewBlock(call, log)
	assert.Nil(t, err)

	wg.Wait()

	err = p.Disconnect()
	assert.Nil(t, err)
	err = s.Disconnect()
	assert.Nil(t, err)
}
