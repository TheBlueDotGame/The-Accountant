package transaction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const createSignTimeDiff = time.Hour * 24 * 7 // week

type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Transaction contains transaction information, subject type, subject data, signatues and public keys.
type Transaction struct {
	ID                primitive.ObjectID `json:"-"                  bson:"_id"`
	CreatedAt         time.Time          `json:"created_at"         bson:"created_at"`
	Hash              [32]byte           `json:"hash"               bson:"hash"`
	IssuerAddress     string             `json:"issuer_address"     bson:"issuer_address"`
	ReceiverAddress   string             `json:"receiver_address"   bson:"receiver_address"`
	Subject           string             `json:"subject"            bson:"subcject"`
	Data              []byte             `json:"data"               bson:"data"`
	IssuerSignature   []byte             `json:"issuer_signature"   bson:"issuer_signature"`
	ReceiverSignature []byte             `json:"receiver_signature" bson:"receiver_signature"`
}

// New creates new transaction signed by issuer.
func New(subject string, message []byte, issuer Signer) (Transaction, error) {
	createdAt := time.Now()
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(createdAt.UnixMicro()))
	message = append(message, b...)
	hash, signature := issuer.Sign(message)

	return Transaction{
		ID:                primitive.NilObjectID,
		CreatedAt:         createdAt,
		Hash:              hash,
		IssuerAddress:     issuer.Address(),
		ReceiverAddress:   "",
		Subject:           subject,
		Data:              message,
		IssuerSignature:   signature,
		ReceiverSignature: []byte{},
	}, nil
}

// Sign signs Transaction by receiver.
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error) {
	now := time.Now()

	if t.CreatedAt.UnixMicro() > now.UnixMicro() {
		return [32]byte{}, errors.New("transaction is created in future")
	}

	if t.CreatedAt.Add(createSignTimeDiff).UnixMicro() < now.UnixMicro() {
		return [32]byte{}, errors.New("transaction is created too long ago")
	}

	if err := v.Verify(t.Data, t.IssuerSignature, [32]byte(t.Hash), t.IssuerAddress); err != nil {
		return [32]byte{}, err
	}

	hash, signature := receiver.Sign(t.Data)

	if !bytes.Equal(hash[:], t.Hash[:]) {
		return [32]byte{}, errors.New("hash is not equal")
	}

	t.ReceiverAddress = receiver.Address()
	t.ReceiverSignature = signature
	return hash, nil
}
