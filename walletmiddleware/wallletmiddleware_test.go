//go:build integration

package walletmiddleware

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/wallet"
)

func TestAlive(t *testing.T) {
	t.Parallel()
	s := aeswrapper.New()
	c := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../test_wallet",
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
			WalletPath:   "../test_wallet",
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
			WalletPath:   "../test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err := issuer.ValidateApiVersion()
	assert.Nil(t, err)
	err = issuer.NewWallet("G8OH7lHu5qfWVumWom0ySN29lakog8nhzSPEwROMjvhdI6VgZ6GoPcdJmoIo7sF3lxQNJMOTKxpYBr6zF992WN86uB7xTEJZ")
	assert.Nil(t, err)

	receiver := NewClient(
		"http://localhost:8080",
		5*time.Second,
		wallet.Helper{},
		fileoperations.New(fileoperations.Config{
			WalletPath:   "../test_wallet",
			WalletPasswd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d",
		}, s),
		wallet.New)
	err = receiver.ValidateApiVersion()
	assert.Nil(t, err)
	err = receiver.NewWallet("jykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ")
	assert.Nil(t, err)

	receiverAddr, err := receiver.Address()
	assert.Nil(t, err)
	err = issuer.ProposeTransaction(receiverAddr, "text", spice.New(0, 0), []byte("test_transaction_data"))
	assert.Nil(t, err)

	awaitedTrx, err := receiver.ReadWaitingTransactions("http://localhost:8080")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(awaitedTrx))

	err = receiver.ConfirmTransaction("http://localhost:8080", &awaitedTrx[0])
	assert.Nil(t, err)
	if err != nil {
		fmt.Printf("err: %v\n", err.Error())
	}

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
			WalletPath:   "../test_wallet",
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
