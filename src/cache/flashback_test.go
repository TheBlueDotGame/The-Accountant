package cache

import (
	"math/rand"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func generate32(r *rand.Rand) []byte {
	data := make([]byte, 32)
	r.Read(data)
	return data
}

func TestHasHashUnique(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	f, err := NewFlash()
	assert.NilError(t, err)

	for i := 0; i < 1000; i++ {
		pseudoHash := generate32(r)
		ok, err := f.HasHash(pseudoHash)
		assert.NilError(t, err)
		assert.Equal(t, ok, false)
	}
}

func TestHasHashRepeating(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	f, err := NewFlash()
	assert.NilError(t, err)

	iter := 1000
	repeated := make([][]byte, 0, iter)
	for i := 0; i < iter; i++ {
		pseudoHash := generate32(r)
		ok, err := f.HasHash(pseudoHash)
		assert.NilError(t, err)
		assert.Equal(t, ok, false)
		repeated = append(repeated, pseudoHash)
	}

	for _, hash := range repeated {
		ok, err := f.HasHash(hash)
		assert.NilError(t, err)
		assert.Equal(t, ok, true)

	}
}
