package webhooksserver

// CreateRemoveUpdateHookRequest is the request send to create, remove or update the webhook.
type CreateRemoveUpdateHookRequest struct {
	URL       string   `json:"address"`        // URL is a url  of the webhook.
	Address   string   `json:"wallet_address"` // Address is the address of the wallet that is used to sign the webhook.
	Data      []byte   `json:"data"`           // Data is the data is a subject of the signature. It is signed by the wallet address.
	Signature []byte   `json:"signature"`      // Signature is the signature of the data. It is used to verify that the data is not changed.
	Digest    [32]byte `json:"digest"`         // Digest is the digest of the data. It is used to verify that the data is not changed.
}

// CreateRemoveUpdateHookResponse is the response send back to the webhook creator.
type CreateRemoveUpdateHookResponse struct {
	Err string `json:"error"`
	Ok  bool   `json:"ok"`
}
