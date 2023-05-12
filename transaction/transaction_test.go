package transaction

import (
	"testing"
	"time"

	"github.com/bartossh/Computantis/wallet"
	"github.com/stretchr/testify/assert"
)

func TestTransaction(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", []byte("message"), signer.Address(), &signer)
	assert.Nil(t, err)
	h, err := trx.Sign(&signer, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedSuccess(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", []byte("message"), signer.Address(), &signer)
	assert.Nil(t, err)
	h, err := trx.Sign(&signer, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedFutureFail(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", []byte("message"), signer.Address(), &signer)
	assert.Nil(t, err)

	trx.CreatedAt = time.Now().Add(4 * time.Minute)
	h, err := trx.Sign(&signer, wallet.Helper{})
	assert.NotNil(t, err)
	assert.Empty(t, h)
}

func TestTransactionCreatedAndSignedFutureFailWrongSignature(t *testing.T) {
	signer0, err := wallet.New()
	assert.Nil(t, err)
	signer1, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", []byte("message"), signer0.Address(), &signer1)
	assert.Nil(t, err)

	h, err := trx.Sign(&signer1, wallet.Helper{})
	assert.NotNil(t, err)
	assert.Empty(t, h)
}

func TestTransactionCreatedAndSignedFutureSuccessIssuerReceiver(t *testing.T) {
	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", []byte("message"), receiver.Address(), &issuer)
	assert.Nil(t, err)

	h, err := trx.Sign(&receiver, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}
