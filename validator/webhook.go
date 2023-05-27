package validator

import (
	"github.com/bartossh/Computantis/block"
	"time"
)

// CreateRemoveUpdateHookRequest is the request send to create, remove or update the webhook.
type CreateRemoveUpdateHookRequest struct {
	URL       string   `json:"address"`        // URL is a url  of the webhook.
	Address   string   `json:"wallet_address"` // Address is the address of the wallet that is used to sign the webhook.
	Data      []byte   `json:"data"`           // Data is the data is a subject of the signature. It is signed by the wallet address.
	Digest    [32]byte `json:"digest"`         // Digest is the digest of the data. It is used to verify that the data is not changed.
	Signature []byte   `json:"signature"`      // Signature is the signature of the data. It is used to verify that the data is not changed.
}

// CreateRemoveUpdateHookResponse is the response send back to the webhook creator.
type CreateRemoveUpdateHookResponse struct {
	Ok  bool   `json:"ok"`
	Err string `json:"error"`
}

// WebHookNewBlockMessage is the message send to the webhook url about new forged block.
type WebHookNewBlockMessage struct {
	Token string      `json:"token"` // Token given to the webhook by the webhooks creator to validate the message source.
	Block block.Block `json:"block"` // Block is the block that was mined.
	Valid bool        `json:"valid"` // Valid is the flag that indicates if the block is valid.
}

const (
	StateIssued      byte = 0 // StateIssued is state of the transaction meaning it is only signed by the issuer.
	StateAcknowleged          // StateAcknowledged is a state ot the transaction meaning it is acknowledged and signed by the receiver.
)

// NewTransactionMessage is the message send to the webhook url about new transaction for given wallet address.
type NewTransactionMessage struct {
	State byte      `json:"state"`
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}
