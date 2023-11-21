package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const minHours = time.Hour * 12

const (
	size = 32
	cost = bcrypt.DefaultCost + 4
)

// Token holds information about unique token.
// Token is a way of proving to the REST API of the central server
// that the request is valid and comes from the client that is allowed to use the API.
type Token struct {
	ID             any    `json:"-"               bson:"_id,omitempty"   db:"id"`
	Token          string `json:"token"           bson:"token"           db:"token"`
	Valid          bool   `json:"valid"           bson:"valid"           db:"valid"`
	ExpirationDate int64  `json:"expiration_date" bson:"expiration_date" db:"expiration_date"`
}

// New creates new token.
func New(expiration int64) (Token, error) {
	t := time.UnixMicro(expiration)
	now := time.Now()

	if t.Before(now.Add(minHours)) {
		return Token{}, fmt.Errorf("expiration time is in the past or is to short")
	}

	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return Token{}, fmt.Errorf("failed to generate token: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword(b, cost)
	if err != nil {
		return Token{}, fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.StdEncoding.EncodeToString(hash)

	return Token{
		Token:          token,
		Valid:          true,
		ExpirationDate: expiration,
	}, nil
}
