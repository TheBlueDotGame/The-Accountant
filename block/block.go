package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var separator = []byte{}

// Block holds block information.
type Block struct {
	ID         primitive.ObjectID `json:"-"          bson:"_id"`
	Index      uint64             `json:"index"      bson:"index"`
	Timestamp  uint64             `json:"timestamp"  bson:"timestamp"`
	Nonce      uint64             `json:"nonce"      bson:"nonce"`
	Difficulty uint64             `json:"difficulty" bson:"difficulty"`
	Hash       [32]byte           `json:"hash"       bson:"hash"`
	PrevHash   [32]byte           `json:"prevHash"   bson:"prevHash"`
	TrxHashes  [][32]byte         `json:"trx_hashes" bson:"trx_hashes"`
}

// NewBlock creates a new Block hashing it with given difficulty.
// Higher difficulty requires more computations to happen to find possible target hash.
// Difficulty is stored inside the Block and is a part of a hashed data.
// Transactions hashes are prehashed before calculating the Block hash with merkle tree.
func NewBlock(difficulty, next uint64, prevHash [32]byte, trxHashes [][32]byte) Block {
	ts := uint64(time.Now().UnixNano())

	block := Block{
		ID:         primitive.NilObjectID,
		Index:      next,
		Timestamp:  ts,
		Nonce:      0,
		Difficulty: difficulty,
		Hash:       [32]byte{},
		PrevHash:   prevHash,
		TrxHashes:  trxHashes,
	}

	trxHash := block.hashTrxs()

	proof := newProof(&block)
	block.Nonce, block.Hash = proof.run(trxHash)

	return block
}

// Validate validates the Block.
// Validations goes in the same order like Block hashing allgorithm,
// just the proof of work part is not required as Nonce is arleady known.
func (b *Block) Validate(trxHashes [][32]byte) bool {
	if !b.validateTransactionsHashesMatch(trxHashes) {
		return false
	}
	trxHash := b.hashTrxs()
	proof := newProof(b)
	return proof.validate(trxHash)
}

func (b *Block) validateTransactionsHashesMatch(trxHashes [][32]byte) bool {
	if len(trxHashes) != len(b.TrxHashes) {
		return false
	}

	set := make(map[[32]byte]struct{})
	for _, h := range b.TrxHashes {
		set[h] = struct{}{}
	}

	if len(set) != len(b.TrxHashes) {
		return false
	}

	for _, h := range trxHashes {
		if _, ok := set[h]; !ok {
			return false
		}
	}

	return true
}

func (b *Block) hashTrxs() [32]byte {
	merkle := newMerkleTree(b.TrxHashes)
	if merkle == nil {
		return [32]byte{}
	}
	return merkle.rootNode.hash
}

type merkleTree struct {
	rootNode *merkleNode
}

type merkleNode struct {
	left  *merkleNode
	right *merkleNode
	hash  [32]byte
}

func newMerkleNode(left, right *merkleNode, hash [32]byte) *merkleNode {
	node := merkleNode{}

	switch {
	case left == nil && right == nil:
		node.hash = hash
	default:
		prevHashes := append(left.hash[:], right.hash[:]...)
		hash := sha256.Sum256(prevHashes)
		node.hash = hash
	}

	node.left = left
	node.right = right

	return &node
}

func newMerkleTree(hashes [][32]byte) *merkleTree {
	if len(hashes) == 0 {
		return nil
	}

	nodes := make([]merkleNode, 0, len(hashes))
	for _, d := range hashes {
		node := newMerkleNode(nil, nil, d)
		nodes = append(nodes, *node)
	}

	for len(nodes) > 1 {
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}

		level := make([]merkleNode, 0, len(nodes)/2)
		for i := 0; i < len(nodes); i += 2 {
			node := newMerkleNode(&nodes[i], &nodes[i+1], [32]byte{})
			level = append(level, *node)
		}
		nodes = level
	}

	tree := merkleTree{&nodes[0]}
	return &tree
}

type proofOfWork struct {
	block  *Block
	target *big.Int
}

func newProof(b *Block) *proofOfWork {
	if b == nil {
		return nil
	}
	target := big.NewInt(1)
	target.Lsh(target, uint(256-b.Difficulty))

	pow := &proofOfWork{b, target}

	return pow
}

func (pow *proofOfWork) initData(nonce uint64, h [32]byte) []byte {
	blockData := make([]byte, 0, 32) // to fit four uint64
	blockData = binary.LittleEndian.AppendUint64(blockData, nonce)
	blockData = binary.LittleEndian.AppendUint64(blockData, pow.block.Difficulty)
	blockData = binary.LittleEndian.AppendUint64(blockData, pow.block.Index)
	blockData = binary.LittleEndian.AppendUint64(blockData, pow.block.Timestamp)
	data := bytes.Join(
		[][]byte{
			pow.block.PrevHash[:],
			h[:],
			blockData,
		},
		separator,
	)

	return data
}

func (pow *proofOfWork) run(trxHash [32]byte) (uint64, [32]byte) {
	var intHash big.Int
	var hash [32]byte
	var nonce uint64

	for nonce <= math.MaxUint64 {
		data := pow.initData(nonce, trxHash)
		hash = sha256.Sum256(data)
		intHash.SetBytes(hash[:])

		if intHash.Cmp(pow.target) == -1 {
			break
		}
		nonce++
	}
	return nonce, hash
}

// Validate validates proof of work
func (pow *proofOfWork) validate(trxHash [32]byte) bool {
	var intHash big.Int
	data := pow.initData(pow.block.Nonce, trxHash)
	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.target) == -1
}
