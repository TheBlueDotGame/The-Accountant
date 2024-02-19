package main

import (
	"errors"
	"os"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/wallet"
)

const (
	actionFromPemToGob = iota
	actionFromGobToPem
	actionNewWallet
	actionReadAddress
)

const usage = `Wallet CLI tool allows to create a new Wallet or act on the local Wallet by using keys from different formats and transforming them between formats.
Please use with the best security practices. GOBINARY is safer to move between machines as this file format is encrypted with AES key.
Tool provides Spice and Contract transfer, reading balance, reading contracts, approving and rejecting contracts.`

func main() {
	pterm.DefaultHeader.WithFullWidth().Println("Computantis")
	var pemFile string
	var walletFile string
	var walletPasswd string

	configurator := func(pemFile, walletFile, walletPasswd string) (configuration.Configuration, error) {
		var cfg configuration.Configuration

		if pemFile == "" {
			cfg.FileOperator.WalletPemPath = "~/.ssh/id_ed25519"
		}

		if walletFile != "" && walletPasswd == "" {
			return cfg, errors.New("wallet file requires a password")
		}

		cfg.FileOperator.WalletPemPath = pemFile
		cfg.FileOperator.WalletPath = walletFile
		cfg.FileOperator.WalletPasswd = walletPasswd

		return cfg, nil
	}

	app := &cli.App{
		Name:  "wallet",
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pem",
				Aliases:     []string{"e"},
				Usage:       "Path to PEM file of ED25519 asymmetric key. Required for creating a new wallet.",
				Destination: &pemFile,
			},
			&cli.StringFlag{
				Name:        "wallet",
				Aliases:     []string{"w"},
				Usage:       "Path to encrypted with AES password wallet file of ED25519 asymmetric key.",
				Destination: &walletFile,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "passwd",
				Aliases:     []string{"p"},
				Usage:       "32 long password key in hex format to open the wallet file.",
				Destination: &walletPasswd,
				Required:    true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Creates new wallet and saves it to encrypted GOBINARY file and PEM format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd)
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
					cfg, err := configurator(pemFile, walletFile, walletPasswd)
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
					cfg, err := configurator(pemFile, walletFile, walletPasswd)
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
			{
				Name:    "address",
				Aliases: []string{"a"},
				Usage:   "Reads wallet public address.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd)
					if err != nil {
						return err
					}
					if err := run(actionReadAddress, cfg.FileOperator); err != nil {
						return err
					}
					printSuccess()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		printError(err)
		return
	}
}

func run(action int, cfg fileoperations.Config) error {
	switch action {
	case actionNewWallet:
		pterm.Info.Println(" CREATING A NEW WALLET ")
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
		pterm.Info.Println(" MOVING WALLET TO PEM KEYS ")
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
		pterm.Info.Println(" MOVING PEM KEYS TO WALLET ")
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
	case actionReadAddress:
		pterm.Info.Println(" READING WALLET PUBLIC ADDRESS ")
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadWallet()
		if err != nil {
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

func printError(err error) {
	pterm.Error.Println("----------")
	pterm.Error.Printf(" Error: %s\n", err.Error())
	pterm.Error.Println("----------")
}

func printWarning(warning string) {
	pterm.Warning.Println("----------")
	pterm.Warning.Printf(" Warning: %s\n", warning)
	pterm.Warning.Println("----------")
}
