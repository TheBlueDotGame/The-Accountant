//go:build stress

// Running stress test requirements:
// - create test database to ensure your data is not polluted,
// - ensure your database contains valid tokens allowing to create wallets,
// - run server with: go run cmd/central/main.go.
// Run test with: `go test -v -run ./stress_test.go -tags stress`
// It is the best to test it when server runs on separate machine.

package stress

import (
	"fmt"
	"testing"
	"time"

	"github.com/bartossh/Computantis/client"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/wallet"
	"github.com/stretchr/testify/assert"
)

func TestFullClientApiCycle(t *testing.T) {
	issuer := client.NewClient("http://localhost:8080", 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
	err := issuer.ValidateApiVersion()
	assert.Nil(t, err)
	err = issuer.NewWallet("wpg6d0grqJjyRicC8oI0/w6IGivm5ypFNTO/wwPGW9A=")
	assert.Nil(t, err)

	receiver := client.NewClient("http://localhost:8080", 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
	err = receiver.ValidateApiVersion()
	assert.Nil(t, err)
	err = receiver.NewWallet("GWFuhvyFnmMg1/vhPCfoa9ct1pAMC1pWwlRg4kt0D/w=")
	assert.Nil(t, err)

	now := time.Now()
	for i := 0; i < 1000; i++ {
		receiverAddr, err := receiver.Address()
		assert.Nil(t, err)
		err = issuer.ProposeTransaction(receiverAddr, "text", []byte("test_transaction_data"))
		assert.Nil(t, err)

		awaitedTrx, err := receiver.ReadWaitingTransactions()
		assert.Nil(t, err)
		assert.Equal(t, 1, len(awaitedTrx))

		err = receiver.ConfirmTransaction(awaitedTrx[0])
		assert.Nil(t, err)
	}
	fmt.Printf("1000 transactions in %v\n", time.Since(now))

	issuer.FlushWalletFromMemory()
	receiver.FlushWalletFromMemory()
}
