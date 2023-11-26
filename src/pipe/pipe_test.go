package pipe

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/bartossh/Computantis/src/accountant"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"gotest.tools/v3/assert"
)

func random32(t *testing.T) [32]byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	assert.NilError(t, err)
	return [32]byte(b)
}

func TestJugglerPipe(t *testing.T) {
	iters := 50
	juggler := New(100, 100)

	go func(juggler *Juggler) {
		for i := 0; i < iters; i++ {
			vrx := accountant.Vertex{Hash: random32(t)}
			h := random32(t)
			trx := protobufcompiled.Transaction{Hash: h[:]}
			juggler.SendVrx(&vrx)
			juggler.SendTrx(&trx)
		}
	}(juggler)
	time.Sleep(time.Second)
	juggler.Close()

	trxs := make(map[[32]byte]struct{})
	vrxs := make(map[[32]byte]struct{})
Outer:
	for {
		select {
		case trx := <-juggler.SubscribeToTrx():
			if trx == nil {
				break Outer
			}
			trxs[[32]byte(trx.Hash)] = struct{}{}
		case vrx := <-juggler.SubscribeToVrx():
			if vrx == nil {
				break Outer
			}
			vrxs[vrx.Hash] = struct{}{}
		}
	}

	assert.Equal(t, len(trxs), iters)
	assert.Equal(t, len(vrxs), iters)
}
