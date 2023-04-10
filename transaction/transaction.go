package transaction

import (
	"bytes"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Transaction contains transaction information, subject type, subject data, signatues and public keys.
type Transaction struct {
	ID                primitive.ObjectID `json:"_id"                bson:"_id"`
	Hash              []byte             `json:"hash"               bson:"hash"`
	IssuerAddress     string             `json:"issuer_address"     bson:"issuer_address"`
	ReceiverAddress   string             `json:"receiver_address"   bson:"receiver_address"`
	Subject           string             `json:"subject"            bson:"subcject"`
	Data              []byte             `json:"data"               bson:"data"`
	IssuerSignature   []byte             `json:"issuer_signature"   bson:"issuer_signature"`
	ReceiverSignature []byte             `json:"receiver_signature" bson:"receiver_signature"`
}

// New creates new transaction signed by issuer.
func New(subject string, message []byte, issuer signer) (Transaction, error) {
	hash, signature := issuer.Sign(message)

	return Transaction{
		ID:                primitive.NilObjectID,
		Hash:              hash[:],
		IssuerAddress:     issuer.Address(),
		ReceiverAddress:   "",
		Subject:           subject,
		Data:              message,
		IssuerSignature:   signature,
		ReceiverSignature: []byte{},
	}, nil
}

// Sign signs trasaction by receiver.
func (t *Transaction) Sign(receiver signer, v verifier) error {
	if err := v.Verify(t.Data, t.IssuerSignature, [32]byte(t.Hash), t.IssuerAddress); err != nil {
		return err
	}

	hash, signature := receiver.Sign(t.Data)

	if !bytes.Equal(hash[:], t.Hash) {
		return errors.New("hash is not equal")
	}

	t.ReceiverAddress = receiver.Address()
	t.ReceiverSignature = signature
	return nil
}
