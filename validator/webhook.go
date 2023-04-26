package validator

import "github.com/bartossh/Computantis/block"

// CreateRemoveUpdateHookRequest is the request sent to create, remove or update the webhook.
type CreateRemoveUpdateHookRequest struct {
	URL       string `json:"address"`        // URL is a url  of the webhook.
	Hook      string `json:"hook"`           // Hook is a type of the webhook. It describes on what event the webhook is triggered.
	Address   string `json:"wallet_address"` // Address is the address of the wallet that is used to sign the webhook.
	Token     string `json:"token"`          // Token is the token added to the webhook to verify that the message comes from the valid source.
	Data      []byte `json:"data"`           // Data is the data is a subject of the signature. It is signed by the wallet address.
	Digest    []byte `json:"digest"`         // Digest is the digest of the data. It is used to verify that the data is not changed.
	Signature []byte `json:"signature"`      // Signature is the signature of the data. It is used to verify that the data is not changed.
}

// WebHookNewBlockMessage is the message sent to the webhook url that was created.
type WebHookNewBlockMessage struct {
	Token string      `json:"token"` // Token given to the webhook by the webhooks creator to validate the message source.
	Block block.Block `json:"block"` // Block is the block that was mined.
	Valid bool        `json:"valid"` // Valid is the flag that indicates if the block is valid.
}
