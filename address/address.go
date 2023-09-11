package address

// Address holds information about unique PublicKey.
type Address struct {
	ID        any    `json:"-"          sql:"id"         db:"id"`
	PublicKey string `json:"public_key" sql:"public_key" db:"public_key"`
}
