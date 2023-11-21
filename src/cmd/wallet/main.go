package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/logo"
	"github.com/bartossh/Computantis/src/wallet"
)

const (
	actionFromPemToGob = iota
	actionFromGobToPem
	actionNewWallet
)

const usage = `Wallet CLI tool allows to create a new Wallet or act on the local Wallet by using keys from different formats and transforming them between formats.
Use with the best security practices. GOBINARY is safer to move between machines as this file format is encrypted with AES key.`

const configFlagDescryption = `Load configuration from 'FILE',
configuration file is required to be in yaml format.
In case configuration file isn't provided,
GOB binary is saved to 'artefacts/wallet' file,
and PEM public and private keys are seved to 'artefacts/ed25519' file.
Wallet password is in hex format.
Please provide arguments like the example:
--- YAML FILE EXAMPLE ---
file_operator:
    wallet_path: "test_wallet"
    wallet_passwd: "dc6b5b1635453e0eb57344ffb6cb293e8300fc4001fad3518e721d548459c09d"
    pem_path: "ed25519"
--- YAML FILE EXAMPLE ---
`

func main() {
	logo.Display()

	var configFilePath string

	configurator := func() (configuration.Configuration, error) {
		var cfg configuration.Configuration
		var err error

		switch configFilePath {
		case "":
			cfg.FileOperator.WalletPath = "./artefacts/wallet"
			cfg.FileOperator.WalletPemPath = "./artefacts/ed25519"
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				return cfg, fmt.Errorf("failed to generate password: %w", err)
			}
			cfg.FileOperator.WalletPasswd = hex.EncodeToString(b)
			pterm.Warning.Println("Wallet creator is using default configuration.")
			pterm.Warning.Printf("Wallet GOB file path: [ %s ].\n", cfg.FileOperator.WalletPath)
			pterm.Warning.Printf("Wallet GOB password: [ %s ]. SAVE ME SOMEWHERE SAFE!\n", cfg.FileOperator.WalletPasswd)
			pterm.Warning.Printf("Wallet PEM file path: [ %s ].\n", cfg.FileOperator.WalletPemPath)
		default:
			cfg, err = configuration.Read(configFilePath)
			if err != nil {
				return cfg, err
			}
			if cfg.FileOperator.WalletPath == "" || cfg.FileOperator.WalletPemPath == "" || cfg.FileOperator.WalletPasswd == "" {
				return cfg, errors.New("cannot read arguments from the configuration file, validate file format, argument names and values")
			}
			pterm.Info.Printf("Wallet creator is using given configuration from file [ %s ].\n", configFilePath)
			pterm.Info.Printf("Wallet GOB file path: [ %s ].\n", cfg.FileOperator.WalletPath)
			pterm.Info.Printf("Wallet PEM file path: [ %s ].\n", cfg.FileOperator.WalletPemPath)
		}

		return cfg, nil
	}

	app := &cli.App{
		Name:  "wallet",
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       configFlagDescryption,
				Destination: &configFilePath,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Creates new wallet and saves it to encrypted GOBINARY file and PEM format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator()
					if err != nil {
						return err
					}
					if err := run(actionNewWallet, cfg.FileOperator); err != nil {
						return err
					}
					printSuccess()
					return nil
				},
			},
			{
				Name:    "topem",
				Aliases: []string{"tp"},
				Usage:   "Reads GOBINARY and saves it to PEM file format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator()
					if err != nil {
						return err
					}
					if err := run(actionFromGobToPem, cfg.FileOperator); err != nil {
						return err
					}
					printSuccess()
					return nil
				},
			},
			{
				Name:    "togob",
				Aliases: []string{"tg"},
				Usage:   "Reads PEM file format and saves it to GOBINARY encrypted file format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator()
					if err != nil {
						return err
					}
					if err := run(actionFromPemToGob, cfg.FileOperator); err != nil {
						return err
					}
					printSuccess()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}

func run(action int, cfg fileoperations.Config) error {
	switch action {
	case actionNewWallet:
		w, err := wallet.New()
		if err != nil {
			return err
		}
		h := fileoperations.New(cfg, aeswrapper.New())
		if err := h.SaveWallet(&w); err != nil {
			return err
		}
		if err := h.SaveToPem(&w); err != nil {
			return err
		}
		printWalletPublicAddress(w.Address())
		return nil
	case actionFromGobToPem:
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadWallet()
		if err != nil {
			return err
		}
		if err := h.SaveToPem(&w); err != nil {
			return err
		}
		printWalletPublicAddress(w.Address())
		return nil

	case actionFromPemToGob:
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadFromPem()
		if err != nil {
			return err
		}
		if err := h.SaveWallet(&w); err != nil {
			return err
		}
		printWalletPublicAddress(w.Address())
		return nil
	default:
		return errors.New("unimplemented action")

	}
}

func printWalletPublicAddress(address string) {
	pterm.Info.Printf("Wallet public address is [ %s ]\n", address)
}

func printSuccess() {
	pterm.Info.Println("----------")
	pterm.Info.Println(" SUCCESS !")
	pterm.Info.Println("----------")
}
