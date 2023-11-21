package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	exp := time.Now().Add(time.Hour * 24).UnixMicro()
	token, err := New(exp)
	assert.Nil(t, err)
	assert.NotEmpty(t, token.Token)
	assert.True(t, token.Valid)
	assert.Equal(t, exp, token.ExpirationDate)
}
func BenchmarkToken(b *testing.B) {
	exp := time.Now().Add(time.Hour * 24).UnixMicro()
	for n := 0; n < b.N; n++ {
		New(exp)
	}
}
