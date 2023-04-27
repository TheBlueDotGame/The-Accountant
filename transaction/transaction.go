package transaction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ExpirationTimeInDays = 7 // transaction validity expiration time in days. TODO: move to config

var (
	ErrTransactionHasAFutureTime        = errors.New("transaction has a future time")
	ErrExpiredTransaction               = errors.New("transaction has expired")
	ErrTransactionHashIsInvalid         = errors.New("transaction hash is invalid")
	ErrSignatureNotValidOrDataCorrupted = errors.New("signature not valid or data are corrupted")
)

// Signer provides signing and address methods.
type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// Verifier provides signature verification method.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

// Transaction contains transaction information, subject type, subject data, signatures and public keys.
// Transaction is valid for a week from being issued.
// Subject represents an information how to read the Data and / or how to decode them.
// Data is not validated by the computantis server, Ladger ior block.
// What is stored in Data is not important for the whole Computantis system.
// It is only important that the data are signed by the issuer and the receiver and
// both parties agreed on them.
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

// New creates new transaction signed by the issuer.
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

// Sign verifies issuer signature and signs Transaction by the receiver.
func (t *Transaction) Sign(receiver Signer, v Verifier) ([32]byte, error) {
	now := time.Now()

	if t.CreatedAt.Unix() > now.Unix() {
		return [32]byte{}, ErrTransactionHasAFutureTime
	}

	if addTime(t.CreatedAt).Unix() < now.Unix() {
		return [32]byte{}, ErrExpiredTransaction
	}

	if err := v.Verify(t.Data, t.IssuerSignature, [32]byte(t.Hash), t.IssuerAddress); err != nil {
		return [32]byte{}, errors.Join(ErrSignatureNotValidOrDataCorrupted, err)
	}

	hash, signature := receiver.Sign(t.Data)

	if !bytes.Equal(hash[:], t.Hash[:]) {
		return [32]byte{}, ErrTransactionHashIsInvalid
	}

	t.ReceiverAddress = receiver.Address()
	t.ReceiverSignature = signature
	return hash, nil
}

func addTime(t time.Time) time.Time {
	return t.AddDate(0, 0, ExpirationTimeInDays)
}
