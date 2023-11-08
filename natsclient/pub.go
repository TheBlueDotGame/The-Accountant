package natsclient

import (
	"google.golang.org/protobuf/proto"

	"github.com/bartossh/Computantis/protobufcompiled"
)

// Publisher provides functionality to push messages to the pub/sub queue
type Publisher struct {
	*socket
}

// PublisherConnect connects publisher to the pub/sub queue using provided config
func PublisherConnect(cfg Config) (*Publisher, error) {
	var p Publisher
	var err error
	p.socket, err = connect(cfg)
	return &p, err
}

// PublishAddressesAwaitingTrxs publishes addresses of the clients that have awaiting transactions.
func (p *Publisher) PublishAddressesAwaitingTrxs(addresses []string, notaryNodeURL string) error {
	protoAddresses := protobufcompiled.Addresses{
		Array:     addresses,
		NotaryUrl: notaryNodeURL,
	}
	msg, err := proto.Marshal(&protoAddresses)
	if err != nil {
		return err
	}
	if err := p.conn.Publish(PubSubAwaitingTrxs, msg); err != nil {
		return err
	}
	return nil
}
