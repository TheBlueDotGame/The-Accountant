package natsclient

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"gotest.tools/assert"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/protobufcompiled"
)

func getBlok() *block.Block {
	b := &block.Block{
		TrxHashes:  [][32]byte{},
		PrevHash:   [32]byte{1, 2, 3},
		Hash:       [32]byte{9, 8, 7, 6},
		Nonce:      11111,
		Difficulty: 12,
		Timestamp:  1212,
		Index:      123456,
	}
	for i := 0; i < 1000; i++ {
		b.TrxHashes = append(b.TrxHashes, [32]byte{byte(i % 10), byte(i % 100), byte(i % 13), byte(i % 41)})
	}
	return b
}

func TestProtoConversionBlok(t *testing.T) {
	blk := getBlok()
	protoBlk := protobufcompiled.Block{}
	protoBlk.TrxHashes = make([][]byte, 0, len(blk.TrxHashes))
	for i := range blk.TrxHashes {
		protoBlk.TrxHashes = append(protoBlk.TrxHashes, blk.TrxHashes[i][:])
	}

	protoBlk.Hash = blk.Hash[:]
	protoBlk.PrevHash = blk.PrevHash[:]
	protoBlk.Index = blk.Index
	protoBlk.Timestamp = blk.Timestamp
	protoBlk.Nonce = blk.Nonce
	protoBlk.Difficulty = blk.Difficulty

	msg, err := proto.Marshal(&protoBlk)
	if err != nil {
		t.Error(err)
	}

	var protoBlkNew protobufcompiled.Block
	if err := proto.Unmarshal(msg, &protoBlkNew); err != nil {
		t.Error(err)
	}
	blkNew := &block.Block{}
	blkNew.TrxHashes = make([][32]byte, 0, len(protoBlk.TrxHashes))
	for _, h := range protoBlk.TrxHashes {
		var hash [32]byte
		copy(hash[:], h)
		blkNew.TrxHashes = append(blkNew.TrxHashes, hash)
	}
	copy(blkNew.Hash[:], protoBlk.Hash)
	copy(blkNew.PrevHash[:], protoBlk.PrevHash)
	blkNew.Index = protoBlk.Index
	blkNew.Timestamp = protoBlk.Timestamp
	blkNew.Nonce = protoBlk.Nonce
	blkNew.Difficulty = protoBlk.Difficulty

	assert.DeepEqual(t, blk, blkNew)
}

func BenchmarkProtoMarshal(b *testing.B) {
	blk := getBlok()
	protoBlk := protobufcompiled.Block{}
	protoBlk.TrxHashes = make([][]byte, 0, len(blk.TrxHashes))
	for i := range blk.TrxHashes {
		protoBlk.TrxHashes = append(protoBlk.TrxHashes, blk.TrxHashes[i][:])
	}

	protoBlk.Hash = blk.Hash[:]
	protoBlk.PrevHash = blk.PrevHash[:]
	protoBlk.Index = blk.Index
	protoBlk.Timestamp = blk.Timestamp
	protoBlk.Nonce = blk.Nonce
	protoBlk.Difficulty = blk.Difficulty

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		proto.Marshal(&protoBlk)
	}
}

func BenchmarkProtoUnmarshalMarshal(b *testing.B) {
	blk := getBlok()
	protoBlk := protobufcompiled.Block{}
	protoBlk.TrxHashes = make([][]byte, 0, len(blk.TrxHashes))
	for i := range blk.TrxHashes {
		protoBlk.TrxHashes = append(protoBlk.TrxHashes, blk.TrxHashes[i][:])
	}

	protoBlk.Hash = blk.Hash[:]
	protoBlk.PrevHash = blk.PrevHash[:]
	protoBlk.Index = blk.Index
	protoBlk.Timestamp = blk.Timestamp
	protoBlk.Nonce = blk.Nonce
	protoBlk.Difficulty = blk.Difficulty

	msg, err := proto.Marshal(&protoBlk)
	if err != nil {
		b.Error(err)
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		var protoBlkNew protobufcompiled.Block
		proto.Unmarshal(msg, &protoBlkNew)
	}
}
