package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/bartossh/Computantis/src/walletmiddleware"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/rand"
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
	primary := pterm.NewStyle(pterm.FgLightCyan, pterm.BgGray, pterm.Bold)
	primary.Println("")
	primary.Println("  Hello Computantis  ")
	primary.Println("")
	var (
		pemFile      string
		walletFile   string
		walletPasswd string
		nodeURL      string
		cert         string
	)

	configurator := func(pemFile, walletFile, walletPasswd, cert string) (configuration.Configuration, error) {
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
		cfg.FileOperator.CAPath = cert
		return cfg, nil
	}

	app := &cli.App{
		Name:  "wallet",
		Usage: usage,
		Commands: []*cli.Command{
			{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Creates new wallet and saves it to encrypted GOBINARY file and PEM format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd, cert)
					if err != nil {
						return err
					}
					if err := runFileOp(actionNewWallet, cfg.FileOperator); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "pem",
						Aliases:     []string{"e"},
						Usage:       "Path to PEM file of ED25519 asymmetric key. Required for creating a new wallet.",
						Destination: &pemFile,
						Required:    true,
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
			},
			{
				Name:    "topem",
				Aliases: []string{"tp"},
				Usage:   "Reads GOBINARY and saves it to PEM file format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd, cert)
					if err != nil {
						return err
					}
					if err := runFileOp(actionFromGobToPem, cfg.FileOperator); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "pem",
						Aliases:     []string{"e"},
						Usage:       "Path to PEM file of ED25519 asymmetric key. Required for creating a new wallet.",
						Destination: &pemFile,
						Required:    true,
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
			},
			{
				Name:    "togob",
				Aliases: []string{"tg"},
				Usage:   "Reads PEM file format and saves it to GOBINARY encrypted file format.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd, cert)
					if err != nil {
						return err
					}
					if err := runFileOp(actionFromPemToGob, cfg.FileOperator); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "pem",
						Aliases:     []string{"e"},
						Usage:       "Path to PEM file of ED25519 asymmetric key. Required for creating a new wallet.",
						Destination: &pemFile,
						Required:    true,
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
			},
			{
				Name:    "address",
				Aliases: []string{"a"},
				Usage:   "Reads wallet public address.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd, cert)
					if err != nil {
						return err
					}
					if err := runFileOp(actionReadAddress, cfg.FileOperator); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
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
			},
			{
				Name:    "connect",
				Aliases: []string{"c"},
				Usage:   "Establish connection with node.",
				Action: func(_ *cli.Context) error {
					cfg, err := configurator(pemFile, walletFile, walletPasswd, cert)
					if err != nil {
						return err
					}
					if err := runTransactionOps(cfg.FileOperator, nodeURL); err != nil {
						return err
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "node",
						Aliases:     []string{"n"},
						Usage:       "Node URL address.",
						Destination: &nodeURL,
						Required:    true,
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
					&cli.StringFlag{
						Name:        "cert",
						Aliases:     []string{"c"},
						Usage:       "Path to certificate authority file.",
						Destination: &cert,
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

func runTransactionOps(cfg fileoperations.Config, nodeURL string) error {
	ctx := context.Background()
	h := fileoperations.New(cfg, aeswrapper.New())
	verify := wallet.NewVerifier()
	c, err := walletmiddleware.NewClient(nodeURL, cfg.CAPath, &verify, &h, wallet.New)
	if err != nil {
		return fmt.Errorf("cannot establish connection to the node %s", nodeURL)
	}
	if err := c.ReadWalletFromFile(); err != nil {
		return fmt.Errorf("cannot read wallet %s", cfg.WalletPath)
	}

	options := []string{"Send tokens", "Check balance", "Read Transactions", "Quit"}

	for {
		selectedOption, _ := pterm.DefaultInteractiveSelect.WithOptions(options).Show()

		switch selectedOption {
		case "Send tokens":
			receiver, _ := pterm.DefaultInteractiveTextInput.Show("Provide receiver public wallet address")
			subject, _ := pterm.DefaultInteractiveTextInput.Show("Provide the subject")
			if subject == "" {
				printError(errors.New("subject cannot be empty"))
				continue
			}
			value, _ := pterm.DefaultInteractiveTextInput.Show("Provide amount of tokans")
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				printError(err)
				continue
			}
			if v == 0.0 {
				printError(errors.New("transfer value cannot be 0"))
				continue
			}
			melange := spice.FromFloat(v)
			result, _ := pterm.DefaultInteractiveConfirm.Show(
				fmt.Sprintf(
					"Are you sure you want to transfer [ %s ] tokens to [ %s ].\n",
					melange.String(), receiver,
				),
			)
			pterm.Println()
			if !result {
				printWarning("Transaction has been rejected.")
				continue
			}
			spinnerInfo, _ := pterm.DefaultSpinner.Start("Sending transaction ...")
			time.Sleep(getPause())
			if err := c.ProposeTransaction(ctx, receiver, subject, melange, []byte{}); err != nil {
				spinnerInfo.Stop()
				printError(fmt.Errorf("cannot propose transaction due to, %e", err))
				continue
			}
			spinnerInfo.Info("Transaction send.")
			printSuccess()
		case "Check balance":
			spinnerInfo, _ := pterm.DefaultSpinner.Start("Checking balance ...")
			time.Sleep(getPause())
			melange, err := c.ReadBalance(ctx)
			if err != nil {
				spinnerInfo.Stop()
				printError(fmt.Errorf("cannot read balance due to, %w", err))
				continue
			}
			addr, err := c.Address()
			if err != nil {
				spinnerInfo.Stop()
				printError(fmt.Errorf("cannot read wallet address due to, %w", err))
				continue
			}
			spinnerInfo.Info(fmt.Sprintf("Account [ %s ] balance is [ %v ]", addr, melange.String()))
			printSuccess()
		case "Read Transactions":
			spinnerInfo, _ := pterm.DefaultSpinner.Start("Reading transactions ...")
			time.Sleep(getPause())
			transactions, err := c.ReadDAGTransactions(ctx)
			if err != nil {
				spinnerInfo.Stop()
				printError(fmt.Errorf("cannot read transactions due to, %w", err))
				continue
			}
			addr, err := c.Address()
			if err != nil {
				spinnerInfo.Stop()
				printError(fmt.Errorf("cannot read wallet address due to, %w", err))
				continue
			}
			spinnerInfo.Info(fmt.Sprintf("Account [ %s ] received [ %v ] transactions", addr, len(transactions)))

			if len(transactions) == 0 {
				printSuccess()
				continue
			}
			tableData := pterm.TableData{
				{"Subject", "From", "To", "Transfer", "Time"},
			}
			for _, trx := range transactions {
				tableData = append(tableData, []string{
					trx.Subject, trx.IssuerAddress, trx.ReceiverAddress,
					trx.Spice.String(), trx.CreatedAt.UTC().Format("2006-01-02T15:04:05"),
				})
			}

			pterm.DefaultTable.WithHasHeader().WithRightAlignment().WithData(tableData).Render()
			printSuccess()

		case "Quit":
			return nil
		default:
			return errors.New("unimplemented action")
		}
	}
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

func getPause() time.Duration {
	p := rand.Intn(5)
	return time.Duration(p) * time.Second
}
