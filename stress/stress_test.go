//go:build stress

// Running stress test requirements:
// - create test database to ensure your data is not polluted,
// - ensure your database contains valid tokens allowing to create wallets,
// - run server with: go run cmd/central/main.go.
// Run test with: `go test -v ./stress/... -tags stress`
// It is the best to test it when server runs on separate machine.

package stress

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/walletmiddleware"
)

// Run two central nodes for this test, one on port 8000 and second on 8088.
// Create corresponding tokens to be valid in database.
func TestFullClientApiCycle(t *testing.T) {
	serverAddress := "http://192.168.0.206"
	transactionsCount := 20
	type testCase struct {
		port   string
		tokens []string
	}
	testCases := []testCase{
		{
			port: "8000",
			tokens: []string{
				"ykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ",
				"8CdWLXrx5GGSSu3je0m6SbCqIuEj7emrsrt7lvm6AeaIQl8d6MCNZKMS00ODA6TrjVYKg4NB9Js4xlSetRdZ4edYupHgBKwX",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"G8OH7lHu5qfWVumWom0ySN29lakog8nhzSPEwROMjvhdI6VgZ6GoPcdJmoIo7sF3lxQNJMOTKxpYBr6zF992WN86uB7xTEJZ",
				"jykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"bIJZyIQLw9hTP0rnbOwmK1G4xlcAXT46IPEkqFdF03gpb2YDuASjWyYVtJIDFdbJm5cRueIbEozhxN8DeevIuapj4BPwfK3d",
				"wGrKWMTNzVT5kqtBWPAlRz58L2AOY3BSZ9PN7WGm1EonyGStnOFNX9y3Tr0p635vbe5dD1TiONgCGiP7yIVc2tVEzfCnYL15",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"ZepH88DsFcoPoZUzIE0AI3gRcCrQ8KhDpzESbxoQiyrB77CtKn7MZnjcj9cRla4aucjrgpnTMtM1AtkegwhXnE6iAKRv6hON",
				"w4NXZ8H5vebzhfgvfanFXzEIaoPwyWeZpZjRheo4LnG8vjWlMQeNVBz9lCMhTiBbj1PjVFWXHiUyZW21P7o6DkTlrx5x3tJ1",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"80fda91a43989fa81347aa011e0f1e0fdde4eaabb408bf426166a62c80456c30",
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bacc9",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bacc1",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bac22",
				"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b543",
			},
		},
		{
			port: "8000",
			tokens: []string{
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bac11",
				"11b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b543",
			},
		},
	}

	var wg sync.WaitGroup
	now := time.Now()
	for _, c := range testCases {
		wg.Add(1)
		go func(c testCase) {
			addr := fmt.Sprintf("%s:%s", serverAddress, c.port)
			issuer := walletmiddleware.NewClient(addr, 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
			err := issuer.ValidateApiVersion()
			assert.Nil(t, err)
			err = issuer.NewWallet(c.tokens[0])
			assert.Nil(t, err)

			receiver := walletmiddleware.NewClient(addr, 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
			err = receiver.ValidateApiVersion()
			assert.Nil(t, err)
			err = receiver.NewWallet(c.tokens[1])
			assert.Nil(t, err)

			for i := 0; i < transactionsCount; i++ {
				receiverAddr, err := receiver.Address()
				assert.Nil(t, err)
				err = issuer.ProposeTransaction(receiverAddr, "text", []byte(fmt.Sprintf("test_transaction_data:%v:%s", i, receiverAddr)))
				assert.Nil(t, err)

				awaitedTrx, err := receiver.ReadWaitingTransactions()
				assert.Nil(t, err)
				assert.Equal(t, 1, len(awaitedTrx))

				for i := range awaitedTrx {
					receiver.ConfirmTransaction(&awaitedTrx[i])
					assert.Nil(t, err)
				}
			}

			issuer.FlushWalletFromMemory()
			receiver.FlushWalletFromMemory()
			wg.Done()
		}(c)
	}

	wg.Wait()

	fmt.Printf("%v transactions in %v\n", transactionsCount*len(testCases), time.Since(now))
}
