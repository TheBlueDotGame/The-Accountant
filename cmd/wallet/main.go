package main

import (
	"errors"
	"os"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/logo"
	"github.com/bartossh/Computantis/wallet"
)

const (
	actionFromPemToGob = iota
	actionFromGobToPem
	acctionNewWallet
)

const usage = `Wallet CLI tool allows to create a new Wallet or act on the local Wallet by using keys from different formats and transforming them between formats.
Use with the best seciurity practices. GOBINARY is safer to move between machines as this file format is encrypted with AES key.`

func main() {
	logo.Display()

	var pem, config string

	configurator := func() (configuration.Configuration, error) {
		if config == "" {
			return configuration.Configuration{}, errors.New("please specify configuration file path with -c <path to file>")
		}

		cfg, err := configuration.Read(config)
		if err != nil {
			return cfg, err
		}

		return cfg, nil
	}

	app := &cli.App{
		Name:  "wallet",
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pem",
				Aliases:     []string{"p"},
				Usage:       "Load wallet from PEM `FILE` path. Your path shall look like that 'path/to/wallet' and the files are 'wallet' and 'wallet.pem'.",
				Destination: &pem,
			},
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Destination: &config,
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
					if err := run(acctionNewWallet, pem, cfg.FileOperator); err != nil {
						return err
					}
					pterm.Info.Println("----------")
					pterm.Info.Println(" SUCCESS !")
					pterm.Info.Println("----------")
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
					if err := run(actionFromGobToPem, pem, cfg.FileOperator); err != nil {
						return err
					}
					pterm.Info.Println("----------")
					pterm.Info.Println(" SUCCESS !")
					pterm.Info.Println("----------")
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
					if err := run(actionFromPemToGob, pem, cfg.FileOperator); err != nil {
						return err
					}
					pterm.Info.Println("----------")
					pterm.Info.Println(" SUCCESS !")
					pterm.Info.Println("----------")
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}

func run(action int, pem string, cfg fileoperations.Config) error {
	switch action {
	case acctionNewWallet:
		w, err := wallet.New()
		if err != nil {
			return err
		}
		h := fileoperations.New(cfg, aeswrapper.New())
		if err := h.SaveWallet(&w); err != nil {
			return err
		}
		if err := h.SaveToPem(&w, pem); err != nil {
			return err
		}
		return nil
	case actionFromGobToPem:
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadWallet()
		if err != nil {
			return err
		}
		if err := h.SaveToPem(&w, pem); err != nil {
			return err
		}
		return nil

	case actionFromPemToGob:
		h := fileoperations.New(cfg, aeswrapper.New())
		w, err := h.ReadFromPem(pem)
		if err != nil {
			return err
		}
		if err := h.SaveWallet(&w); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("unimplemented action")

	}
}
