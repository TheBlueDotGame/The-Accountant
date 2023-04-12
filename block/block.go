package block

import (
	"crypto/sha256"
	"encoding/binary"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
