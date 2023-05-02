package address

// Address holds information about unique PublicKey.
type Address struct {
	ID        any    `json:"-"          bson:"_id,omitempty" db:"id"`
	PublicKey string `json:"public_key" bson:"public_key"    db:"public_key"`
}
