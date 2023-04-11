package wallet

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func generateRandom(bytesNum int) []byte {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 0, bytesNum)

	for i := 0; i < bytesNum; i++ {
		b = append(b, byte(rand.Intn(256)))
	}

	return b
}

func TestAddressVerifySuccess(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	addr := w.Address()
	assert.NotEmpty(t, addr)

	message := []byte("This is message to sign.")

	hash, sig := w.Sign(message)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, sig)

	err = Helper{}.Verify(message, sig, hash, addr)
	assert.Nil(t, err)
}

func TestAddressVerifyFail(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	message := []byte("This is message to sign.")

	hash, sig := w.Sign(message)
	assert.NotEmpty(t, hash)
	assert.NotEmpty(t, sig)

	nw, err := New()
	assert.Nil(t, err)
	addr := nw.Address()
	assert.NotEmpty(t, addr)

	err = Helper{}.Verify(message, sig, hash, addr)
	assert.NotNil(t, err)
}

func BenchmarkAddressVerifyLargeMessage(b *testing.B) {
	w, err := New()
	assert.Nil(b, err)

	message := generateRandom(1000000)
	addr := w.Address()

	for n := 0; n < b.N; n++ {
		hash, sig := w.Sign(message)
		err = Helper{}.Verify(message, sig, hash, addr)
		assert.Nil(b, err)
	}
}
