package client

import (
	"testing"
	"time"

	"github.com/bartossh/The-Accountant/fileoperations"
	"github.com/bartossh/The-Accountant/wallet"
	"github.com/stretchr/testify/assert"
)

func TestAlive(t *testing.T) {
	t.Parallel()
	c := NewRest("http://localhost:8080", 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
	err := c.ValidateApiVersion()
	assert.Nil(t, err)
}

func BenchmarkAlive(b *testing.B) {
	c := NewRest("http://localhost:8080", 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
	for i := 0; i < b.N; i++ {
		_ = c.ValidateApiVersion()
	}
}
