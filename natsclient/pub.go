package natsclient

import (
	"google.golang.org/protobuf/proto"

	"github.com/bartossh/Computantis/block"
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

// PublishNewBlock publishes new block.
func (p *Publisher) PublishNewBlock(blk block.Block) error {
	protoBlk := protobufcompiled.Block{}
	protoBlk.TrxHashes = make([][]byte, 0, len(blk.TrxHashes))
	for _, h := range blk.TrxHashes {
		protoBlk.TrxHashes = append(protoBlk.TrxHashes, h[:])
	}
	protoBlk.Hash = blk.Hash[:]
	protoBlk.PrevHash = blk.PrevHash[:]
	protoBlk.Index = blk.Index
	protoBlk.Timestamp = blk.Timestamp
	protoBlk.Nonce = blk.Nonce
	protoBlk.Difficulty = blk.Difficulty

	msg, err := proto.Marshal(&protoBlk)
	if err != nil {
		return err
	}
	if err := p.conn.Publish(PubSubNewBlock, msg); err != nil {
		return err
	}

	return nil
}
