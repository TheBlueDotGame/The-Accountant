package transaction

import (
	"crypto/ed25519"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Transaction contains transaction information, subject type, subject data, signatues and public keys.
type Transaction struct {
	ID                primitive.ObjectID `json:"_id"                bson:"_id"`
	Hash              []byte             `json:"hash"               bson:"hash"`
	IssuerPubKey      ed25519.PublicKey  `json:"issuer_pub_key"     bson:"issuer_pub_key"`
	ReceiverPubKey    ed25519.PublicKey  `json:"receiver_pub_key"   bson:"receiver_pub_key"`
	Subject           string             `json:"subject"            bson:"subcject"`
	Data              []byte             `json:"data"               bson:"data"`
	IssuerSignature   []byte             `json:"issuer_signature"   bson:"issuer_signature"`
	ReceiverSignature []byte             `json:"receiver_signature" bson:"receiver_signature"`
}
