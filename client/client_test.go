//go:build integration

package client

import (
	"fmt"
	"testing"
	"time"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/wallet"
	"github.com/stretchr/testify/assert"
)

func TestAlive(t *testing.T) {
	t.Parallel()
	s := aeswrapper.New()
	c := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../artefacts/test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err := c.ValidateApiVersion()
	assert.Nil(t, err)
}

func BenchmarkAlive(b *testing.B) {
	s := aeswrapper.New()
	c := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../artefacts/test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	for i := 0; i < b.N; i++ {
		_ = c.ValidateApiVersion()
	}
}

func TestFullClientApiCycle(t *testing.T) {
	s := aeswrapper.New()
	issuer := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../artefacts/test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err := issuer.ValidateApiVersion()
	assert.Nil(t, err)
	err = issuer.NewWallet("wpg6d0grqJjyRicC8oI0/w6IGivm5ypFNTO/wwPGW9A=")
	assert.Nil(t, err)

	receiver := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../artefacts/test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err = receiver.ValidateApiVersion()
	assert.Nil(t, err)
	err = receiver.NewWallet("GWFuhvyFnmMg1/vhPCfoa9ct1pAMC1pWwlRg4kt0D/w=")
	assert.Nil(t, err)

	receiverAddr, err := receiver.Address()
	assert.Nil(t, err)
	err = issuer.ProposeTransaction(receiverAddr, "text", []byte("test_transaction_data"))
	assert.Nil(t, err)
	issuedTrx, err := issuer.ReadIssuedTransactions()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(issuedTrx))

	awaitedTrx, err := receiver.ReadWaitingTransactions()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(awaitedTrx))

	err = receiver.ConfirmTransaction(&awaitedTrx[0])
	assert.Nil(t, err)

	issuer.FlushWalletFromMemory()
	receiver.FlushWalletFromMemory()
}

func TestSaveWallet(t *testing.T) {
	t.Parallel()
	s := aeswrapper.New()
	c := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../artefacts/test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err := c.ValidateApiVersion()
	assert.Nil(t, err)
	err = c.NewWallet("80fda91a43989fa81347aa011e0f1e0fdde4eaabb408bf426166a62c80456c30")
	assert.Nil(t, err)
	err = c.SaveWalletToFile()
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
	}
	assert.Nil(t, err)
}
