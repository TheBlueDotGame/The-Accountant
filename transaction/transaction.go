package transaction

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	minAddressLength     = 49
	ExpirationTimeInDays = 7 // transaction validity expiration time in days. TODO: move to config
)

var (
	ErrTransactionHasAFutureTime        = errors.New("transaction has a future time")
	ErrExpiredTransaction               = errors.New("transaction has expired")
	ErrTransactionHashIsInvalid         = errors.New("transaction hash is invalid")
	ErrSignatureNotValidOrDataCorrupted = errors.New("signature not valid or data are corrupted")
	ErrSubjectIsEmpty                   = errors.New("subject cannot be empty")
	ErrAddressIsInvalid                 = errors.New("address is invalid")
)

// TrxAddressesSubscriberCallback is a method or function performing compoutantion on the transactions addresses.
type TrxAddressesSubscriberCallback func(addresses []string)

// Signer provides signing and address methods.
type Signer interface {
	Sign(message []byte) (digest [32]byte, signature []byte)
	Address() string
}

// Verifier provides signature verification method.
type Verifier interface {
	Verify(message, signature []byte, hash [32]byte, issuer string) error
}

// Transaction contains transaction information, subject type, subject data, signatures and public keys.
// Transaction is valid for a week from being issued.
// Subject represents an information how to read the Data and / or how to decode them.
// Data is not validated by the computantis server, Ladger ior block.
// What is stored in Data is not important for the whole Computantis system.
// It is only important that the data are signed by the issuer and the receiver and
// both parties agreed on them.
type Transaction struct {
	ID                any       `json:"-"                  bson:"_id"                db:"id"`
	CreatedAt         time.Time `json:"created_at"         bson:"created_at"         db:"created_at"`
	IssuerAddress     string    `json:"issuer_address"     bson:"issuer_address"     db:"issuer_address"`
	ReceiverAddress   string    `json:"receiver_address"   bson:"receiver_address"   db:"receiver_address"`
	Subject           string    `json:"subject"            bson:"subject"            db:"subject"`
	Data              []byte    `json:"data"               bson:"data"               db:"data"`
	IssuerSignature   []byte    `json:"issuer_signature"   bson:"issuer_signature"   db:"issuer_signature"`
	ReceiverSignature []byte    `json:"receiver_signature" bson:"receiver_signature" db:"receiver_signature"`
	Hash              [32]byte  `json:"hash"               bson:"hash"               db:"hash"`
}

// New creates new transaction signed by the issuer.
func New(subject string, data []byte, receiverAddress string, issuer Signer) (Transaction, error) {
	if len(subject) == 0 {
		return Transaction{}, ErrSubjectIsEmpty
	}

	if len(receiverAddress) < minAddressLength {
		return Transaction{}, ErrAddressIsInvalid
	}

	createdAt := time.Now()

	msgLen := len(subject) + len(data) + len(issuer.Address()) + len(receiverAddress) + 8
	message := make([]byte, 0, msgLen)
	message = append(message, []byte(subject)...)
	message = append(message, data...)
	message = append(message, []byte(issuer.Address())...)
	message = append(message, []byte(receiverAddress)...)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(createdAt.UnixMicro()))
	message = append(message, b...)
	hash, signature := issuer.Sign(message)

	return Transaction{
		ID:                primitive.NilObjectID,
		CreatedAt:         createdAt,
		Hash:              hash,
		IssuerAddress:     issuer.Address(),
		ReceiverAddress:   receiverAddress,
		Subject:           subject,
		Data:              data,
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

	msgLen := len(t.Subject) + len(t.Data) + len(t.IssuerAddress) + len(receiver.Address()) + 8
	message := make([]byte, 0, msgLen)
	message = append(message, []byte(t.Subject)...)
	message = append(message, t.Data...)
	message = append(message, []byte(t.IssuerAddress)...)
	message = append(message, []byte(receiver.Address())...)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(t.CreatedAt.UnixMicro()))
	message = append(message, b...)

	if err := v.Verify(message, t.IssuerSignature, [32]byte(t.Hash), t.IssuerAddress); err != nil {
		return [32]byte{}, errors.Join(ErrSignatureNotValidOrDataCorrupted, err)
	}

	hash, signature := receiver.Sign(message)

	if !bytes.Equal(hash[:], t.Hash[:]) {
		return [32]byte{}, ErrTransactionHashIsInvalid
	}

	t.ReceiverSignature = signature
	return hash, nil
}

// GeMessage returns message used for signature validation.
func (t *Transaction) GetMessage() []byte {
	msgLen := len(t.Subject) + len(t.Data) + len(t.IssuerAddress) + len(t.ReceiverAddress) + 8
	message := make([]byte, 0, msgLen)
	message = append(message, []byte(t.Subject)...)
	message = append(message, t.Data...)
	message = append(message, []byte(t.IssuerAddress)...)
	message = append(message, []byte(t.ReceiverAddress)...)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(t.CreatedAt.UnixMicro()))
	return append(message, b...)
}

func addTime(t time.Time) time.Time {
	return t.AddDate(0, 0, ExpirationTimeInDays)
}
