package transaction

import (
	"encoding/binary"
	"math"
	"slices"
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

func TestCompareAppendAndCopy(t *testing.T) {
	data := []byte("Some short message to copy")
	tm := time.Now()
	s := spice.New(10000, 10000)
	b0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b0, uint64(tm.UnixNano()))
	b1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b1, s.Currency)
	b2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(b2, s.SupplementaryCurrency)

	message1 := make([]byte, 0, len(data)+24)
	message1 = append(message1, append(data, append(b0, append(b1, b2...)...)...)...)

	message2 := make([]byte, len(data)+24)

	n := copy(message2[:], data)
	n += copy(message2[n:], b0)
	n += copy(message2[n:], b1)
	copy(message2[n:], b2)

	assert.Equal(t, slices.Compare(message1, message2), 0)
}

func BenchmarkAppend(b *testing.B) {
	data := []byte("Some short message to copy")
	tm := time.Now()
	s := spice.New(10000, 10000)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b0 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b0, uint64(tm.UnixNano()))
		b1 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b1, s.Currency)
		b2 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b2, s.SupplementaryCurrency)

		message := make([]byte, 0, len(data)+24)
		_ = append(message, append(data, append(b0, append(b1, b2...)...)...)...)

	}
}

func BenchmarkCopy(b *testing.B) {
	data := []byte("Some short message to copy")
	tm := time.Now()
	s := spice.New(10000, 10000)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b0 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b0, uint64(tm.UnixNano()))
		b1 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b1, s.Currency)
		b2 := make([]byte, 8)
		binary.LittleEndian.PutUint64(b2, s.SupplementaryCurrency)

		message := make([]byte, len(data)+24)

		n := copy(message[:], data)
		n += copy(message[n:], b0)
		n += copy(message[n:], b1)
		copy(message[n:], b2)
	}
}

func BenchmarkNewTransaction(b *testing.B) {
	issuer, err := wallet.New()
	assert.Nil(b, err)
	receiver, err := wallet.New()
	assert.Nil(b, err)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	}
}

func BenchmarkSignTransaction(b *testing.B) {
	issuer, err := wallet.New()
	assert.Nil(b, err)
	receiver, err := wallet.New()
	assert.Nil(b, err)
	trx, err := New("subject", spice.New(math.MaxInt64, 0), []byte("message"), receiver.Address(), &issuer)
	assert.Nil(b, err)
	wh := wallet.Helper{}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		trx.Sign(&receiver, wh)
	}
}
