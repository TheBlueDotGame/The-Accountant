package blockchain

import (
	"context"
	"errors"
	"sync"

	"github.com/bartossh/The-Accountant/block"
)

var (
	ErrBlockNotFound        = errors.New("block not found")
	ErrInvalidBlockPrevHash = errors.New("block prev hash is invalid")
	ErrInvalidBlockHash     = errors.New("block hash is invalid")
	ErrInvalidBlockIndex    = errors.New("block index is invalid")
)

type blockReader interface {
	LastBlock(ctx context.Context) (block.Block, error)
	ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
}

type blockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

type blockReadWriter interface {
	blockReader
	blockWriter
}

// Blockchain keeps track of the blocks.
type Blockchain struct {
	mux            sync.RWMutex
	lastBlockHash  [32]byte
	lastBlockIndex uint64
	rw             blockReadWriter
}

// NewChaion creates a new Blockchain that has access to the blockchain stired in the repository.
func NewBlockchain(ctx context.Context, rw blockReadWriter) (*Blockchain, error) {
	lastBlock, err := rw.LastBlock(ctx)
	if err != nil {
		return nil, err
	}

	return &Blockchain{
		mux:            sync.RWMutex{},
		lastBlockHash:  lastBlock.Hash,
		lastBlockIndex: lastBlock.Index,
		rw:             rw,
	}, nil
}

// LastBlock returns last block hash and index.
func (c *Blockchain) LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	bl, err := c.rw.ReadBlockByHash(ctx, c.lastBlockHash)
	if err != nil {
		return [32]byte{}, 0, err
	}
	return bl.Hash, bl.Index, nil
}

// ReadLastNBlocks reads the last n blocks.
func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	blocks := make([]block.Block, 0, n)

	lastBlockHash := c.lastBlockHash
	for n > 0 {
		block, err := c.rw.ReadBlockByHash(ctx, lastBlockHash)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		lastBlockHash = block.PrevHash
		n--
	}
	return blocks, nil
}

// ReadBlocksFromIndex reads all blocks from given index till the current block index.
func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	blocks := make([]block.Block, 0, c.lastBlockIndex-idx)

	lastBlockHash := c.lastBlockHash
	for {
		block, err := c.rw.ReadBlockByHash(ctx, lastBlockHash)
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

// WriteBlock writes block in to the blockchain repository.
func (c *Blockchain) WriteBlock(ctx context.Context, block block.Block) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if block.Index != c.lastBlockIndex+1 {
		return ErrInvalidBlockIndex
	}

	if block.PrevHash != c.lastBlockHash {
		return ErrInvalidBlockPrevHash
	}

	if err := c.rw.WriteBlock(ctx, block); err != nil {
		return err
	}

	c.lastBlockHash = block.Hash
	c.lastBlockIndex = block.Index

	return nil
}