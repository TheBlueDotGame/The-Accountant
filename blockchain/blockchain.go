package blockchain

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/bartossh/Computantis/block"
)

var (
	ErrBlockNotFound        = errors.New("block not found")
	ErrInvalidBlockPrevHash = errors.New("block prev hash is invalid")
	ErrInvalidBlockHash     = errors.New("block hash is invalid")
	ErrInvalidBlockIndex    = errors.New("block index is invalid")
)

// BlockReader provides read access to the blockchain repository.
type BlockReader interface {
	LastBlock(ctx context.Context) (block.Block, error)
	ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error)
}

// BlockWriter provides write access to the blockchain repository.
type BlockWriter interface {
	WriteBlock(ctx context.Context, block block.Block) error
}

// BlockReadWriter provides read and write access to the blockchain repository.
type BlockReadWriter interface {
	BlockReader
	BlockWriter
}

// Blockchain keeps track of the blocks creating immutable chain of data.
// Blockchain is stored in repository as separate blocks that relates to each other
// based on the hash of the previous block.
type Blockchain struct {
	rw BlockReadWriter
}

// GenesisBlock creates a genesis block. It is a first block in the blockchain.
// The genesis block is created only if there is no other block in the repository.
// Otherwise returning an error.
func GenesisBlock(ctx context.Context, rw BlockReadWriter) error {
	if b, err := rw.LastBlock(ctx); err == nil && b.Index != 0 {
		return errors.New("genesis block already exists")
	}
	h := sha256.Sum256([]byte("genesis block"))
	b := block.New(0, 1, h, [][32]byte{})
	return rw.WriteBlock(ctx, b)
}

// New creates a new Blockchain that has access to the blockchain stored in the repository.
// The access to the repository is injected via BlockReadWriter interface.
// You can use any implementation of repository that implements BlockReadWriter interface
// and ensures unique indexing for Block Hash, PrevHash and Index.
func New(ctx context.Context, rw BlockReadWriter) (*Blockchain, error) {
	return &Blockchain{
		rw: rw,
	}, nil
}

// LastBlockHashIndex returns last block hash and index.
func (c *Blockchain) LastBlockHashIndex(ctx context.Context) ([32]byte, uint64, error) {
	lastBlock, err := c.rw.LastBlock(ctx)
	if err != nil {
		return [32]byte{}, 0, err
	}
	return lastBlock.Hash, lastBlock.Index, nil
}

// ReadLastNBlocks reads the last n blocks in reverse consecutive order.
func (c *Blockchain) ReadLastNBlocks(ctx context.Context, n int) ([]block.Block, error) {
	lastBlock, err := c.rw.LastBlock(ctx)
	if err != nil {
		return nil, err
	}

	blocks := make([]block.Block, 0, n)
	lastBlockHash := lastBlock.Hash
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

// ReadBlocksFromIndex reads all blocks from given index till the current block in consecutive order.
func (c *Blockchain) ReadBlocksFromIndex(ctx context.Context, idx uint64) ([]block.Block, error) {
	lastBlock, err := c.rw.LastBlock(ctx)
	if err != nil {
		return nil, err
	}

	blocks := make([]block.Block, 0, lastBlock.Index-idx)
	lastBlockHash := lastBlock.Hash
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
	lastBlock, err := c.rw.LastBlock(ctx)
	if err != nil {
		return err
	}

	if block.Index != lastBlock.Index+1 {
		return ErrInvalidBlockIndex
	}

	if block.PrevHash != lastBlock.Hash {
		return ErrInvalidBlockPrevHash
	}

	if err := c.rw.WriteBlock(ctx, block); err != nil {
		return err
	}

	return nil
}
