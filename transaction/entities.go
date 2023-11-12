package transaction

// TransactionAwaitingReceiverSignature represents transaction awaiting receiver signature.
// It is as well the entity of all issued transactions that has not been signed by receiver yet.
type TransactionAwaitingReceiverSignature struct {
	ID              any         `json:"-"                bson:"_id,omitempty"    db:"id"`
	ReceiverAddress string      `json:"receiver_address" bson:"receiver_address" db:"receiver_address"`
	IssuerAddress   string      `json:"issuer_address"   bson:"issuer_address"   db:"issuer_address"`
	Transaction     Transaction `json:"transaction"      bson:"transaction"      db:"-"`
	TransactionHash [32]byte    `json:"transaction_hash" bson:"transaction_hash" db:"hash"`
}
