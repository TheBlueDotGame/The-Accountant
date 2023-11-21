package natsclient

import (
	"sync"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/transaction"
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

// SubscribeNewTransactionsForAddresses subscribes to pub/sub queue for a addresses awaitng transactions.
func (s *Subscriber) SubscribeNewTransactionsForAddresses(call transaction.TrxAddressesSubscriberCallback, log logger.Logger) error {
	sub, err := s.conn.Subscribe(PubSubAwaitingTrxs, func(m *nats.Msg) {
		var protoAddresses protobufcompiled.Addresses
		if err := proto.Unmarshal(m.Data, &protoAddresses); err != nil {
			log.Error(err.Error())
			return
		}
		call(protoAddresses.GetArray(), protoAddresses.GetNotaryUrl())
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
