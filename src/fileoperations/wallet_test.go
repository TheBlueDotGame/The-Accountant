package fileoperations

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/wallet"
)

func TestSaveReadWalletEncodeDecodeSuccess(t *testing.T) {
	s := aeswrapper.New()
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			key := make([]byte, 32)

			_, err := io.ReadFull(rand.Reader, key)
			assert.Nil(t, err)

			helper := New(Config{
				WalletPath:   "../artefacts/test_wallet",
				WalletPasswd: hex.EncodeToString(key),
			}, s)

			w0, err := wallet.New()
			assert.Nil(t, err)

			err = helper.SaveWallet(&w0)
			assert.Nil(t, err)
			w1, err := helper.ReadWallet()
			assert.Nil(t, err)
			assert.Equal(t, w0.Private, w1.Private)
			assert.Equal(t, w0.Public, w1.Public)
		})
	}
}

func TestSaveReadWalletEncodeDecodeEncryptDecryptSuccess(t *testing.T) {
	s := aeswrapper.New()
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			testMessage := make([]byte, 1024)
			_, err := io.ReadFull(rand.Reader, testMessage)
			assert.Nil(t, err)

			key := make([]byte, 32)

			_, err = io.ReadFull(rand.Reader, key)
			assert.Nil(t, err)

			helper := New(Config{
				WalletPath:   "../artefacts/test_wallet",
				WalletPasswd: hex.EncodeToString(key),
			}, s)

			w0, err := wallet.New()
			assert.Nil(t, err)

			d0, s0 := w0.Sign(testMessage)

			err = helper.SaveWallet(&w0)
			assert.Nil(t, err)
			w1, err := helper.ReadWallet()
			assert.Nil(t, err)
			assert.Equal(t, w0.Private, w1.Private)
			assert.Equal(t, w0.Public, w1.Public)

			d2, s2 := w1.Sign(testMessage)

			assert.Equal(t, d0, d2)
			assert.Equal(t, s0, s2)
		})
	}
}

func TestSaveAndReadPEM(t *testing.T) {
	h := New(Config{WalletPemPath: "./wallet"}, aeswrapper.New())
	w, err := wallet.New()
	assert.Nil(t, err)
	assert.NotNil(t, w.Private)
	assert.NotNil(t, w.Public)

	err = h.SaveToPem(&w)
	assert.Nil(t, err)

	nw, err := h.ReadFromPem()
	assert.Nil(t, err)
	assert.Equal(t, w.Private, nw.Private)
	assert.Equal(t, w.Public, nw.Public)

	err = os.Remove(h.cfg.WalletPemPath)
	assert.Nil(t, err)
	err = os.Remove(h.cfg.WalletPemPath + ".pub")
	assert.Nil(t, err)
}

func BenchmarkSaveReadWalletEncodeDecodeSuccess(b *testing.B) {
	s := aeswrapper.New()
	key := make([]byte, 32)

	_, err := io.ReadFull(rand.Reader, key)
	assert.Nil(b, err)
	helper := New(Config{
		WalletPath:    "../artefacts/test_wallet",
		WalletPasswd:  hex.EncodeToString(key),
		WalletPemPath: "../artefacts/ed25519",
	}, s)
	for i := 0; i < b.N; i++ {
		w0, err := wallet.New()
		assert.Nil(b, err)
		err = helper.SaveWallet(&w0)
		assert.Nil(b, err)
		_, err = helper.ReadWallet()
		assert.Nil(b, err)
	}
}
