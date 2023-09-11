package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"

	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/walletmiddleware"
)

type config struct {
	CentralNodeIP   string        `yaml:"central_node_ip"`
	CentralNodePort int           `yaml:"central_node_port"`
	ProcessTick     time.Duration `yaml:"process_tick_ms"`
}

func read(path string) (config, error) {
	var cfg config
	buf, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("in file %q: %w", path, err)
	}
	if ip := net.ParseIP(cfg.CentralNodeIP); ip == nil {
		return cfg, fmt.Errorf("provided ip [ %s ] is incorrect", cfg.CentralNodeIP)
	}

	if cfg.CentralNodePort < 0 || cfg.CentralNodePort > 65535 {
		return cfg, fmt.Errorf("provided port [ %v ] is out of range 0 to 65535", cfg.CentralNodePort)
	}

	if cfg.ProcessTick > 1000 {
		return cfg, fmt.Errorf("provided process tick [ %v ] is to large to stress the device", cfg.ProcessTick)
	}

	return cfg, err
}

func main() {
	cfg, err := read("stress_test_setup.yaml")
	if err != nil {
		pterm.Error.Printf("Failed with error: %s\n", err)
	}

	type connection struct {
		tokens []string
	}
	connections := []connection{
		{
			tokens: []string{
				"ykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ",
				"8CdWLXrx5GGSSu3je0m6SbCqIuEj7emrsrt7lvm6AeaIQl8d6MCNZKMS00ODA6TrjVYKg4NB9Js4xlSetRdZ4edYupHgBKwX",
			},
		},
		{
			tokens: []string{
				"G8OH7lHu5qfWVumWom0ySN29lakog8nhzSPEwROMjvhdI6VgZ6GoPcdJmoIo7sF3lxQNJMOTKxpYBr6zF992WN86uB7xTEJZ",
				"jykkeD6Tr6xikkYwC805kVoFThm8VGEHStTFk1lIU6RgEf7p3vjFpPQFI3VP9SYeARjYh2jecMSYsmgddjZZcy32iySHijJQ",
			},
		},
		{
			tokens: []string{
				"bIJZyIQLw9hTP0rnbOwmK1G4xlcAXT46IPEkqFdF03gpb2YDuASjWyYVtJIDFdbJm5cRueIbEozhxN8DeevIuapj4BPwfK3d",
				"wGrKWMTNzVT5kqtBWPAlRz58L2AOY3BSZ9PN7WGm1EonyGStnOFNX9y3Tr0p635vbe5dD1TiONgCGiP7yIVc2tVEzfCnYL15",
			},
		},
		{
			tokens: []string{
				"ZepH88DsFcoPoZUzIE0AI3gRcCrQ8KhDpzESbxoQiyrB77CtKn7MZnjcj9cRla4aucjrgpnTMtM1AtkegwhXnE6iAKRv6hON",
				"w4NXZ8H5vebzhfgvfanFXzEIaoPwyWeZpZjRheo4LnG8vjWlMQeNVBz9lCMhTiBbj1PjVFWXHiUyZW21P7o6DkTlrx5x3tJ1",
			},
		},
		{
			tokens: []string{
				"80fda91a43989fa81347aa011e0f1e0fdde4eaabb408bf426166a62c80456c30",
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bacc9",
			},
		},
		{
			tokens: []string{
				"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bacc1",
			},
		},
		{
			tokens: []string{
				"7147a8f255f49cb7693dcd19b6b46e139680d48a03e0a075ea237deb7e6bac22",
				"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b543",
			},
		},
		{
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

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting [ %v ] independent connections to stress test central node", len(connections)))
	time.Sleep(time.Second * 5)
	spinner.UpdateText("Running stress test ...")
	for _, c := range connections {
		go func(ctx context.Context, c connection) {
			addr := fmt.Sprintf("http://%s:%v", cfg.CentralNodeIP, cfg.CentralNodePort)
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

			tc := time.NewTicker(time.Millisecond * cfg.ProcessTick)
			defer tc.Stop()
		Trxs:
			for {
				select {
				case <-ctx.Done():
					break Trxs
				case <-tc.C:
					now := time.Now()
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
	spinner.UpdateText("STRESS TEST STOPED")
	spinner.Stop()
}
