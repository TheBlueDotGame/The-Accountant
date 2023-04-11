package transaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testSignerMock struct{}

func (th testSignerMock) Sign(message []byte) (digest [32]byte, signature []byte) {
	return [32]byte{}, []byte("signature")
}

func (th testSignerMock) Address() string {
	return "thisisaddress"
}

type testVerifierMock struct{}

func (tv testVerifierMock) Verify(message, signature []byte, hash [32]byte, address string) error {
	return nil
}

func TestTransaction(t *testing.T) {
	trx, err := New("subject", []byte("message"), testSignerMock{})
	assert.Nil(t, err)
	err = trx.Sign(testSignerMock{}, testVerifierMock{})
	assert.Nil(t, err)
}
