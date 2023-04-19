package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testSignerMock struct{}

func (th testSignerMock) Sign(message []byte) (digest [32]byte, signature []byte) {
	return [32]byte{
			1,
			2,
			3,
			4,
			5,
			6,
			7,
			8,
			9,
			1,
			2,
			3,
			4,
			6,
			7,
			8,
			9,
			1,
			2,
			3,
			4,
			5,
			6,
			7,
			8,
			9,
			1,
			2,
			3,
			4,
			5,
			6,
		}, []byte(
			"signature",
		)
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
	h, err := trx.Sign(testSignerMock{}, testVerifierMock{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedSuccess(t *testing.T) {
	trx, err := New("subject", []byte("message"), testSignerMock{})
	assert.Nil(t, err)

	trx.CreatedAt = time.Now().Add(-4 * time.Minute)
	h, err := trx.Sign(testSignerMock{}, testVerifierMock{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedFutureFail(t *testing.T) {
	trx, err := New("subject", []byte("message"), testSignerMock{})
	assert.Nil(t, err)

	trx.CreatedAt = time.Now().Add(4 * time.Minute)
	h, err := trx.Sign(testSignerMock{}, testVerifierMock{})
	assert.NotNil(t, err)
	assert.Empty(t, h)
}
