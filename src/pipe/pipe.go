package pipe

import (
	"github.com/bartossh/Computantis/src/accountant"
	"github.com/bartossh/Computantis/src/protobufcompiled"
)

// Juggler juggles entities put on the pipeline between subscriber and receiver.
// There are many channels inside the piper and for the reason of performance
// types shall not be generic. This coupling with types is necessary to ensure
// high throughput.
type Juggler struct {
	trxCh  chan *protobufcompiled.Transaction
	vrxCh  chan *accountant.Vertex
	closed bool
}

func New(trxBufSize, vrxBufSize uint16) *Juggler {
	return &Juggler{
		trxCh:  make(chan *protobufcompiled.Transaction, int(trxBufSize)),
		vrxCh:  make(chan *accountant.Vertex, int(vrxBufSize)),
		closed: false,
	}
}

func (j *Juggler) Close() {
	if j.closed {
		return
	}
	close(j.trxCh)
	close(j.vrxCh)
	j.closed = true
}

// SendTrx sends concurrently transaction to the subscriber if channel is open or otherwise returns false.
func (j *Juggler) SendTrx(trx *protobufcompiled.Transaction) bool {
	if j.closed {
		return false
	}
	go func(trx *protobufcompiled.Transaction) {
		j.trxCh <- trx
	}(trx)

	return true
}

// SendVrx sends concurrently vertex to the subscriber if channel is open or otherwise returns false.
func (j *Juggler) SendVrx(vrx *accountant.Vertex) bool {
	if j.closed {
		return false
	}
	go func(vrx *accountant.Vertex) {
		j.vrxCh <- vrx
	}(vrx)

	return true
}

// SubscribeToTrx returns channel of transactions.
func (j *Juggler) SubscribeToTrx() <-chan *protobufcompiled.Transaction {
	return j.trxCh
}

// SubscribeToVrx returns channel of vertexes.
func (j *Juggler) SubscribeToVrx() <-chan *accountant.Vertex {
	return j.vrxCh
}
