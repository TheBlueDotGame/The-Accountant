package transformers

import (
	"errors"
	"time"

	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/transaction"
)

var (
	ErrProcessing = errors.New("processing failed")
	ErrTrxIsEmpty = errors.New("trx is empty")
)

func TrxToProtoTrx(trx *transaction.Transaction) (*protobufcompiled.Transaction, error) {
	if trx == nil || trx.Subject == "" || trx.IssuerAddress == "" ||
		trx.ReceiverAddress == "" || len(trx.Hash) == 0 ||
		trx.CreatedAt.IsZero() || len(trx.IssuerSignature) == 0 {
		return &protobufcompiled.Transaction{}, ErrProcessing
	}
	return &protobufcompiled.Transaction{
		CreatedAt:         uint64(trx.CreatedAt.UnixNano()),
		IssuerAddress:     trx.IssuerAddress,
		ReceiverAddress:   trx.ReceiverAddress,
		Subject:           trx.Subject,
		IssuerSignature:   trx.IssuerSignature,
		ReceiverSignature: trx.ReceiverSignature,
		Data:              trx.Data,
		Hash:              trx.Hash[:],
		Spice: &protobufcompiled.Spice{
			Currency:             trx.Spice.Currency,
			SuplementaryCurrency: trx.Spice.SupplementaryCurrency,
		},
	}, nil
}

func ProtoTrxToTrx(prTrx *protobufcompiled.Transaction) (transaction.Transaction, error) {
	if prTrx == nil || prTrx.Subject == "" || prTrx.IssuerAddress == "" ||
		prTrx.ReceiverAddress == "" || len(prTrx.Hash) == 0 ||
		prTrx.CreatedAt == 0 || len(prTrx.IssuerSignature) == 0 {
		return transaction.Transaction{}, ErrTrxIsEmpty
	}
	return transaction.Transaction{
		CreatedAt:         time.Unix(0, int64(prTrx.CreatedAt)),
		IssuerAddress:     prTrx.IssuerAddress,
		ReceiverAddress:   prTrx.ReceiverAddress,
		Subject:           prTrx.Subject,
		IssuerSignature:   prTrx.IssuerSignature,
		ReceiverSignature: prTrx.ReceiverSignature,
		Data:              prTrx.Data,
		Hash:              [32]byte(prTrx.Hash),
		Spice: spice.Melange{
			Currency:              prTrx.Spice.Currency,
			SupplementaryCurrency: prTrx.Spice.SuplementaryCurrency,
		},
	}, nil
}
