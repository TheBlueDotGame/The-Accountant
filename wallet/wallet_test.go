package wallet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateWallet(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)
}

func TestGobEncodingDecoding(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	b, err := w.EncodeGOB()
	assert.Nil(t, err)
	assert.NotNil(t, b)

	nw, err := DecodeGOBWallet(b)
	assert.Nil(t, err)
	assert.Equal(t, nw.Private, w.Private)
	assert.Equal(t, nw.Public, w.Public)
}

func TestSignVerifySuccess(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	message := []byte("This is test message.")

	hash, sig := w.Sign(message)
	assert.NotNil(t, hash)
	assert.NotNil(t, sig)

	ok := w.Verify(message, sig, hash)
	assert.True(t, ok)
}

func TestSignVerifyFail(t *testing.T) {
	w, err := New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	message := []byte("This is test message.")

	nw, err := New()
	assert.Nil(t, err)
	hash, sig := nw.Sign(message)
	assert.NotNil(t, hash)
	assert.NotNil(t, sig)

	ok := w.Verify(message, sig, hash)
	assert.False(t, ok)
}

func BenchmarkVerifyLargeMessage(b *testing.B) {
	w, err := New()
	assert.Nil(b, err)

	message := generateRandom(1000000)

	for n := 0; n < b.N; n++ {
		hash, sig := w.Sign(message)
		ok := w.Verify(message, sig, hash)
		assert.True(b, ok)
	}
}
