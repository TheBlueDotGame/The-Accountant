package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrBlockNotFound        = errors.New("block not found")
	ErrInvalidBlockPrevHash = errors.New("block prev hash is invalid")
	ErrInvalidBlockHash     = errors.New("block hash is invalid")
	ErrInvalidBlockIndex    = errors.New("block index is invalid")
)

type blockReader interface {
	LastBlock() (Block, error)
	ReadBlockByHash(hash [32]byte) (Block, error)
}

type blockWriter interface {
	WriteBlock(block Block) error
}

type blockReadWriter interface {
	blockReader
	blockWriter
}

// Block holds block information.
type Block struct {
	ID        primitive.ObjectID `json:"-"          bson:"_id"`
	Index     uint64             `json:"index"      bson:"index"`
	Timestamp uint64             `json:"timestamp"  bson:"timestamp"`
	Hash      [32]byte           `json:"hash"       bson:"hash"`
	PrevHash  [32]byte           `json:"prevHash"   bson:"prevHash"`
	TrxHashes [][32]byte         `json:"trx_hashes" bson:"trx_hashes"`
}

// NewBlock creates a new block.
func NewBlock(next uint64, prevHash [32]byte, trxHashes [][32]byte) Block {
	ts := uint64(time.Now().UnixNano())

	block := Block{
		ID:        primitive.NilObjectID,
		Index:     next,
		Timestamp: ts,
		Hash:      [32]byte{},
		PrevHash:  prevHash,
		TrxHashes: trxHashes,
	}
	block.calculateHash()

	return block
}

func (b *Block) calculateHash() {
	var data []byte
	binary.LittleEndian.AppendUint64(data, b.Index)
	binary.LittleEndian.AppendUint64(data, b.Timestamp)
	data = append(data, b.PrevHash[:]...)
	for _, trx := range b.TrxHashes {
		data = append(data, trx[:]...)
	}

	b.Hash = sha256.Sum256(data)
}

// Chain keeps track of the blocks.
type Chain struct {
	mux            sync.RWMutex
	lastBlockHash  [32]byte
	lastBlockIndex uint64
	rw             blockReadWriter
}

// NewChaion creates a new Chain that has access to the blockchain stired in the repository.
func NewChain(rw blockReadWriter) (*Chain, error) {
	lastBlock, err := rw.LastBlock()
	if err != nil {
		return nil, err
	}

	return &Chain{
		mux:            sync.RWMutex{},
		lastBlockHash:  lastBlock.Hash,
		lastBlockIndex: lastBlock.Index,
		rw:             rw,
	}, nil
}

// LastNBlocks return the last n blocks.
func (c *Chain) LastNBlocks(n int) ([]Block, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	blocks := make([]Block, 0, n)

	lastBlockHash := c.lastBlockHash
	for n > 0 {
		block, err := c.rw.ReadBlockByHash(lastBlockHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		lastBlockHash = block.PrevHash
		n--
	}
	return blocks, nil
}

// BlocksFromIndex returns all blocks from given index till the current block index.
func (c *Chain) BlocksFromIndex(idx uint64) ([]Block, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	blocks := make([]Block, 0, c.lastBlockIndex-idx)

	lastBlockHash := c.lastBlockHash
	for {
		block, err := c.rw.ReadBlockByHash(lastBlockHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		lastBlockHash = block.PrevHash

		if block.Index == idx {
			break
		}

	}
	return blocks, nil
}

// AddBlock adds block in to the blockchain repository.
func (c *Chain) AddBlock(block Block) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if block.Index != c.lastBlockIndex+1 {
		return ErrInvalidBlockIndex
	}

	if block.PrevHash != c.lastBlockHash {
		return ErrInvalidBlockPrevHash
	}

	if err := c.rw.WriteBlock(block); err != nil {
		return err
	}

	c.lastBlockHash = block.Hash
	c.lastBlockIndex = block.Index

	return nil
}
