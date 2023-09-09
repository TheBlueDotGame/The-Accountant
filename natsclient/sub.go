package natsclient

import (
	"sync"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/transaction"
)

// Subscriber provides functionality to pull messages from the pub/sub queue.
type Subscriber struct {
	*socket
	subs map[string]*nats.Subscription
	mux  sync.RWMutex
}

// SubscriberConnect connects publisher to the pub/sub queue using provided config
func SubscriberConnect(cfg Config) (*Subscriber, error) {
	var s Subscriber
	var err error
	s.socket, err = connect(cfg)
	s.subs = make(map[string]*nats.Subscription)
	return &s, err
}

// SubscribeNewBlock subscribes to pub/sub queue for a new block read.
func (s *Subscriber) SubscribeNewBlock(call block.BlockSubscriberCallback, log logger.Logger) error {
	sub, err := s.conn.Subscribe(PubSubNewBlock, func(m *nats.Msg) {
		var protoBlk protobufcompiled.Block
		if err := proto.Unmarshal(m.Data, &protoBlk); err != nil {
			log.Error(err.Error())
			return
		}
		var blk block.Block
		blk.TrxHashes = make([][32]byte, 0, len(protoBlk.TrxHashes))
		for _, h := range protoBlk.TrxHashes {
			var a [32]byte
			copy(a[:], h)
			blk.TrxHashes = append(blk.TrxHashes, a)
		}
		copy(blk.Hash[:], protoBlk.Hash)
		copy(blk.PrevHash[:], protoBlk.PrevHash)
		blk.Index = protoBlk.Index
		blk.Timestamp = protoBlk.Timestamp
		blk.Nonce = protoBlk.Nonce
		blk.Difficulty = protoBlk.Difficulty
		call(&blk)
	})
	if err != nil {
		sub.Unsubscribe()
		return err
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.subs[PubSubNewBlock] = sub

	return nil
}

// SubscribeNewTransactionsForAddresses subscribes to pub/sub queue for a addresses awaitng transactions.
func (s *Subscriber) SubscribeNewTransactionsForAddresses(call transaction.TrxAddressesSubscriberCallback, log logger.Logger) error {
	sub, err := s.conn.Subscribe(PubSubAwaitingTrxs, func(m *nats.Msg) {
		var protoAddresses protobufcompiled.Addresses
		if err := proto.Unmarshal(m.Data, &protoAddresses); err != nil {
			log.Error(err.Error())
			return
		}
		call(protoAddresses.Array)
	})
	if err != nil {
		sub.Unsubscribe()
		return err
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.subs[PubSubAwaitingTrxs] = sub

	return nil
}
