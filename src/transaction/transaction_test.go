package transaction

import (
	"math"
	"testing"
	"time"

	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/stretchr/testify/assert"
)

func TestTransaction(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), signer.Address(), &signer)
	assert.Nil(t, err)
	h, err := trx.Sign(&signer, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedSuccess(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), signer.Address(), &signer)
	assert.Nil(t, err)
	h, err := trx.Sign(&signer, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedFutureFail(t *testing.T) {
	signer, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), signer.Address(), &signer)
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
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), signer0.Address(), &signer1)
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
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	assert.Nil(t, err)

	h, err := trx.Sign(&receiver, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestTransactionCreatedAndSignedFutureSuccessIssuerReceiverVerify(t *testing.T) {
	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	assert.Nil(t, err)

	h, err := trx.Sign(&receiver, wallet.Helper{})
	assert.Nil(t, err)
	assert.NotEmpty(t, h)
}

func TestSmallTimeSeparation(t *testing.T) {
	issuer, err := wallet.New()
	assert.Nil(t, err)
	receiver, err := wallet.New()
	assert.Nil(t, err)
	trx0, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	assert.Nil(t, err)
	time.Sleep(time.Second)
	trx1, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	assert.Nil(t, err)

	msg0 := trx0.GetMessage()
	msg1 := trx1.GetMessage()
	if len(msg0) != len(msg1) {
		return
	}

	var diff bool
	for i := range msg0 {
		if msg0[i] != msg1[i] {
			diff = true
		}
	}

	assert.True(t, diff)
}
