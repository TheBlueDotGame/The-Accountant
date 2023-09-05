package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pterm/pterm"

	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/walletmiddleware"
)

func main() {
	serverAddress := "http://192.168.0.206"
	var timestep time.Duration = 100 // [ ms ]
	type connection struct {
		port   string
		tokens []string
	}
	connections := []connection{
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

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	for _, c := range connections {
		go func(ctx context.Context, c connection) {
			addr := fmt.Sprintf("%s:%s", serverAddress, c.port)
			issuer := walletmiddleware.NewClient(addr, 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
			err := issuer.ValidateApiVersion()
			if err != nil {
				fmt.Println(err)
				cancel()
			}
			err = issuer.NewWallet(c.tokens[0])
			if err != nil {
				fmt.Println(err)
				cancel()
			}

			receiver := walletmiddleware.NewClient(addr, 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
			err = receiver.ValidateApiVersion()
			if err != nil {
				fmt.Println(err)
				cancel()
			}
			err = receiver.NewWallet(c.tokens[1])
			if err != nil {
				fmt.Println(err)
				cancel()
			}

			tc := time.NewTicker(time.Millisecond * timestep)
			defer tc.Stop()
		Trxs:
			for {
				select {
				case <-ctx.Done():
					break Trxs
				case <-tc.C:
					now := time.Now()
					pterm.Info.Printf("NEXT TRANSACTION [ %v ]\n", now.UnixMicro())
					receiverAddr, err := receiver.Address()
					if err != nil {
						fmt.Println(err)
						cancel()
					}
					err = issuer.ProposeTransaction(receiverAddr, "text", []byte(fmt.Sprintf("test_transaction_data:%v:%s", now.UnixMicro(), receiverAddr)))
					if err != nil {
						fmt.Println(err)
						cancel()
					}

					awaitedTrx, err := receiver.ReadWaitingTransactions()
					if err != nil {
						fmt.Println(err)
						cancel()
					}

					for i := range awaitedTrx {
						receiver.ConfirmTransaction(&awaitedTrx[i])
						if err != nil {
							fmt.Println(err)
							cancel()
						}
					}
				}
			}

			issuer.FlushWalletFromMemory()
			receiver.FlushWalletFromMemory()
		}(ctx, c)
	}

	<-ctx.Done()
	pterm.Warning.Println("STRESS TEST STOPED")
}
