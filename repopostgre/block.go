package repopostgre

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/block"
)

// LastBlock returns last block from the database.
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error) {
	res, err := db.inner.QueryContext(ctx, "SELECT * FROM blocks ORDER BY index DESC LIMIT 1")
	if err != nil {
		return block.Block{}, errors.Join(ErrSelectFailed, err)
	}

	var b block.Block
	if err := res.Scan(&b.ID, &b.Index, &b.Timestamp, &b.Nonce, &b.Difficulty, &b.Hash, &b.PrevHash, &b.TrxHashes); err != nil {
		return block.Block{}, errors.Join(ErrScanFailed, err)
	}

	return b, nil
}

// ReadBlockByHash returns block with given hash.
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error) {
	res, err := db.inner.QueryContext(ctx, "SELECT * FROM blocks WHERE hash = $1", hash)
	if err != nil {
		return block.Block{}, errors.Join(ErrSelectFailed, err)
	}

	var b block.Block
	if err := res.Scan(&b.ID, &b.Index, &b.Timestamp, &b.Nonce, &b.Difficulty, &b.Hash, &b.PrevHash, &b.TrxHashes); err != nil {
		return block.Block{}, errors.Join(ErrScanFailed, err)
	}

	return b, nil
}

// WriteBlock writes block to the database.
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error {
	_, err := db.inner.ExecContext(ctx,
		`INSERT INTO 
				blocks (index, timestamp, nonce, difficulty, hash, prev_hash, trx_hashes) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		block.Index, block.Timestamp, block.Nonce, block.Difficulty, block.Hash, block.PrevHash, block.TrxHashes)
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}
