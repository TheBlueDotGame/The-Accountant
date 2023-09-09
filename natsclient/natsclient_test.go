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

func natsPubSubTestHelper(tb testing.TB) (*Publisher, *Subscriber) {
	cfg := Config{
		Address: "nats://127.0.0.1:4222",
		Name:    "integration-test-1",
		Token:   "D9pHfuiEQPXtqPqPdyxozi8kU2FlHqC0FlSRIzpwDI0=",
	}

	p, err := PublisherConnect(cfg)
	assert.Nil(tb, err)

	s, err := SubscriberConnect(cfg)
	assert.Nil(tb, err)

	return p, s
}

func TestPubSubCycle(t *testing.T) {
	var idx uint64 = 111
	testBlk := block.Block{
		Index: idx,
	}

	p, s := natsPubSubTestHelper(t)

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{})

	err := p.PublishNewBlock(testBlk)
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

func BenchmarkNatsConnection(b *testing.B) {
	var idx uint64 = 111
	testBlk := block.Block{
		Index: idx,
	}

	p, s := natsPubSubTestHelper(b)

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{})

	call := func(blk *block.Block) {
		assert.Equal(b, idx, blk.Index)
	}
	err := s.SubscribeNewBlock(call, log)
	assert.Nil(b, err)

	for n := 0; n < b.N; n++ {
		err := p.PublishNewBlock(testBlk)
		assert.Nil(b, err)
	}
}
