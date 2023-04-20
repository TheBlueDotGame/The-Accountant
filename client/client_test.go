//go:build integration

package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlive(t *testing.T) {
	t.Parallel()
	c := NewRest("http://localhost:8080", 5*time.Second)
	err := c.ValidateApiVersion()
	assert.Nil(t, err)
}

func BenchmarkAlive(b *testing.B) {
	c := NewRest("http://localhost:8080", 5*time.Second)
	for i := 0; i < b.N; i++ {
		_ = c.ValidateApiVersion()
	}
}
