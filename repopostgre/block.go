package repopostgre

import (
	"context"
	"errors"

	"github.com/bartossh/Computantis/block"
	"github.com/lib/pq"
)

// LastBlock returns last block from the database.
func (db DataBase) LastBlock(ctx context.Context) (block.Block, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM blocks ORDER BY index DESC LIMIT 1")
	if err != nil {
		return block.Block{}, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var b block.Block
	h := make([]byte, 0, 32)
	prevH := make([]byte, 0, 32)
	trxHashes := [][]byte{}
	for rows.Next() {
		if err := rows.Scan(&b.ID, &b.Index, &b.Timestamp, &b.Nonce, &b.Difficulty, &h, &prevH, pq.Array(&trxHashes)); err != nil {
			return block.Block{}, errors.Join(ErrScanFailed, err)
		}
	}

	copy(b.Hash[:], h)
	copy(b.PrevHash[:], prevH)
	for _, hash := range trxHashes {
		h := [32]byte{}
		copy(h[:], hash)
		b.TrxHashes = append(b.TrxHashes, h)
	}

	return b, nil
}

// ReadBlockByHash returns block with given hash.
func (db DataBase) ReadBlockByHash(ctx context.Context, hash [32]byte) (block.Block, error) {
	rows, err := db.inner.QueryContext(ctx, "SELECT * FROM blocks WHERE hash = $1", hash)
	if err != nil {
		return block.Block{}, errors.Join(ErrSelectFailed, err)
	}
	defer rows.Close()

	var b block.Block
	h := make([]byte, 0, 32)
	prevH := make([]byte, 32)
	trxHashes := [][]byte{}
	for rows.Next() {
		if err := rows.Scan(&b.ID, &b.Index, &b.Timestamp, &b.Nonce, &b.Difficulty, &h, &prevH, pq.Array(&trxHashes)); err != nil {
			return block.Block{}, errors.Join(ErrScanFailed, err)
		}
	}
	copy(b.Hash[:], h)
	copy(b.PrevHash[:], prevH)
	for _, hash := range trxHashes {
		h := [32]byte{}
		copy(h[:], hash)
		b.TrxHashes = append(b.TrxHashes, h)
	}

	return b, nil
}

// WriteBlock writes block to the database.
func (db DataBase) WriteBlock(ctx context.Context, block block.Block) error {
	trxHashes := make([][]byte, len(block.TrxHashes))
	for _, hash := range block.TrxHashes {
		trxHashes = append(trxHashes, hash[:])
	}
	_, err := db.inner.ExecContext(ctx,
		`INSERT INTO 
				blocks (index, timestamp, nonce, difficulty, hash, prev_hash, trx_hashes) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		block.Index, block.Timestamp, block.Nonce, block.Difficulty, block.Hash[:], block.PrevHash[:], pq.Array(trxHashes))
	if err != nil {
		return errors.Join(ErrInsertFailed, err)
	}
	return nil
}
