package token

// Token holds information about unique token.
// Token is a way of proving to the REST API of the central server
// that the request is valid and comes from the client that is allowed to use the API.
type Token struct {
	ID             any    `json:"-"               bson:"_id,omitempty"   db:"id"`
	Token          string `json:"token"           bson:"token"           db:"token"`
	Valid          bool   `json:"valid"           bson:"valid"           db:"valid"`
	ExpirationDate int64  `json:"expiration_date" bson:"expiration_date" db:"expiration_date"`
}
