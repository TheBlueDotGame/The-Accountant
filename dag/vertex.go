package dag

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/bartossh/Computantis/transaction"

	msgpackv2 "github.com/shamaton/msgpack/v2"
	"github.com/vmihailenco/msgpack"
)

// Vertex is a Direct Acyclic Graph vertex that creates a AccountingBook inner graph.
type Vertex struct {
	AccountantPublicAddress string                  `msgpack:"accountant_public_address"`
	CreatedAt               time.Time               `msgpack:"created_at"`
	Signature               []byte                  `msgpack:"signature"`
	Transaction             transaction.Transaction `msgpack:"transaction"`
	Hash                    [32]byte                `msgpack:"hash"`
	PrevVertexHash          [32]byte                `msgpack:"prev_vertex_hash"`
	LeftParentHash          [32]byte                `msgpack:"left_parent_hash"`
	RightParentHash         [32]byte                `msgpack:"right_parent_hash"`
}

// NewVertex creates new Vertex but first validates transaction legitimacy.
func NewVertex(
	trx transaction.Transaction,
	accountantPubAddress string,
	prevTrxHash, leftParentHash, rightParentHash [32]byte,
	separator []byte,
	verifier signatureVerifier,
	signer signer,
) (Vertex, error) {
	if err := trx.Verify(verifier); err != nil {
		return Vertex{}, err
	}

	candidate := Vertex{
		AccountantPublicAddress: accountantPubAddress,
		CreatedAt:               time.Now(),
		Signature:               nil,
		Transaction:             trx,
		Hash:                    [32]byte{},
		PrevVertexHash:          prevTrxHash,
		LeftParentHash:          leftParentHash,
		RightParentHash:         rightParentHash,
	}

	candidate.sign(separator, signer)

	return candidate, nil
}

func (v *Vertex) initData(separator []byte) []byte {
	blockData := make([]byte, 0, 8)
	blockData = binary.LittleEndian.AppendUint64(blockData, uint64(v.CreatedAt.UnixNano()))
	return bytes.Join([][]byte{
		v.Transaction.Hash[:], v.PrevVertexHash[:], v.LeftParentHash[:], v.RightParentHash[:], blockData,
	},
		separator,
	)
}

func (v *Vertex) sign(separator []byte, signer signer) {
	data := v.initData(separator)
	v.Hash, v.Signature = signer.Sign(data)
}

func (v *Vertex) verify(separator []byte, verifier signatureVerifier) error {
	if err := v.Transaction.Verify(verifier); err != nil {
		return err
	}
	data := v.initData(separator)
	return verifier.Verify(data, v.Signature[:], v.Hash, v.AccountantPublicAddress)
}

func (v *Vertex) encode() ([]byte, []byte, error) {
	buf, err := msgpack.Marshal(*v)
	if err != nil {
		return nil, nil, err
	}
	return v.Hash[:], buf, nil
}

func decode(buf []byte) (Vertex, error) {
	var v Vertex
	err := msgpackv2.Unmarshal(buf, &v)
	return v, err
}
