package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bartossh/Computantis/logo"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/blockchain"
	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/reactive"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/wallet"
)

const (
	rxBufferSize = 100
)

const usage = `The Central Computantis API server is responsible for validating, storing transactions and
forging blocks in immutable blockchain history. The Central Computantis Node is a heart of the whole Computantis system.`

func main() {
	logo.Display()

	var file string
	configurator := func() (configuration.Configuration, error) {
		if file == "" {
			return configuration.Configuration{}, errors.New("please specify configuration file path with -c <path to file>")
		}

		cfg, err := configuration.Read(file)
		if err != nil {
			return cfg, err
		}

		return cfg, nil
	}

	app := &cli.App{
		Name:  "computantis",
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Destination: &file,
			},
		},
		Action: func(cCtx *cli.Context) error {
			cfg, err := configurator()
			if err != nil {
				return err
			}
			run(cfg)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}

func run(cfg configuration.Configuration) {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	db, err := repository.Connect(ctx, cfg.Database)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	sub, err := repository.Subscribe(ctx, cfg.Database)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	ctxx, cancelClose := context.WithTimeout(context.Background(), time.Second*1)
	defer cancelClose()
	defer db.Disconnect(ctxx)

	callbackOnErr := func(err error) {
		fmt.Println("error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, db, stdoutwriter.Logger{})

	if err := blockchain.GenesisBlock(ctx, db); err != nil {
		fmt.Println(err)
	}

	blc, err := blockchain.New(ctx, db)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	verifier := wallet.NewVerifier()
	rxBlock := reactive.New[block.Block](rxBufferSize)
	rxTrxIssuer := reactive.New[string](rxBufferSize)

	ladger, err := bookkeeping.New(cfg.Bookkeeper, blc, db, db, verifier, db, log, rxBlock, rxTrxIssuer, sub)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	err = server.Run(ctx, cfg.Server, db, ladger, dataProvider, log, rxBlock.Subscribe(), rxTrxIssuer.Subscribe())
	if err != nil {
		log.Error(err.Error())
	}
}
