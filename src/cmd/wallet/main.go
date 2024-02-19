package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/bartossh/Computantis/src/walletmiddleware"
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
	var (
		pemFile      string
		walletFile   string
		walletPasswd string
		receiver     string
		nodeURL      string
	)

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
					if err := runFileOp(actionNewWallet, cfg.FileOperator); err != nil {
						return err
					}
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
					if err := runFileOp(actionFromGobToPem, cfg.FileOperator); err != nil {
						return err
					}
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
					if err := runFileOp(actionFromPemToGob, cfg.FileOperator); err != nil {
						return err
					}
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
					if err := runFileOp(actionReadAddress, cfg.FileOperator); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:    "send",
				Aliases: []string{"s"},
				Usage:   "Sends transaction.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd)
					if err != nil {
						return err
					}
					if err := runTransferOp(cfg.FileOperator, receiver, nodeURL); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "receiver",
						Aliases:     []string{"r"},
						Usage:       "Receiver wallet public address.",
						Destination: &receiver,
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "node",
						Aliases:     []string{"n"},
						Usage:       "Node URL address.",
						Destination: &nodeURL,
						Required:    true,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		printError(err)
		return
	}
}

func runTransferOp(cfg fileoperations.Config, receiver, nodeURL string) error {
	ctx := context.Background()
	spiceStr, _ := pterm.DefaultInteractiveTextInput.Show("Provide Spice amount")
	spiceVal, err := strconv.Atoi(spiceStr)
	if err != nil || spiceVal < 0 {
		return errors.New("spice can only be provided as a positive integer")
	}
	suplStr, _ := pterm.DefaultInteractiveTextInput.Show("Provide Supl amount")
	suplVal, err := strconv.Atoi(suplStr)
	if err != nil || suplVal < 0 {
		return errors.New("supl can only be provided as a positive integer")
	}
	result, _ := pterm.DefaultInteractiveConfirm.Show(
		fmt.Sprintf(
			"Are you sure you want to transfer [ %v ] of Spice [ %v ] of Supl Spice to [ %s ].\n",
			spiceVal, suplVal, receiver,
		),
	)
	pterm.Println()
	if !result {
		printWarning("Transaction has been rejected.")
		return nil
	}
	melange := spice.New(uint64(spiceVal), uint64(suplVal))
	h := fileoperations.New(cfg, aeswrapper.New())
	verify := wallet.NewVerifier()
	c, err := walletmiddleware.NewClient(nodeURL, &verify, &h, wallet.New)
	if err != nil {
		return err
	}
	if err := c.ReadWalletFromFile(); err != nil {
		return err
	}
	spinnerInfo, _ := pterm.DefaultSpinner.Start("Sending transaction ...")
	time.Sleep(time.Second * 2)
	if err := c.ProposeTransaction(ctx, receiver, "Spice transfer", melange, []byte{}); err != nil {
		return err
	}
	spinnerInfo.Info("Transaction send.")
	printSuccess()
	return nil
}

func runFileOp(action int, cfg fileoperations.Config) error {
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
		printSuccess()
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
		printSuccess()
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
		printSuccess()
		return nil
	case actionReadAddress:
		pterm.Info.Println(" READING WALLET PUBLIC ADDRESS ")
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadWallet()
		if err != nil {
			return err
		}
		printWalletPublicAddress(w.Address())
		printSuccess()
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
	pterm.Error.Println("")
	pterm.Error.Printf(" %s\n", err.Error())
	pterm.Error.Println("")
}

func printWarning(warning string) {
	pterm.Warning.Println("")
	pterm.Warning.Printf(" %s\n", warning)
	pterm.Warning.Println("")
}
