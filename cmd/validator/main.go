package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bartossh/Computantis/logo"
	"github.com/bartossh/Computantis/telemetry"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/validator"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/webhooks"
)

const usage = `The Validator Computantis API server validates transactions and blocks. In additions Validator offers
webhook endpoint where any application with valid address can register to listen for new blocks or transactions for 
given wallet public address.`

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
		Name:  "validator",
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
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	db, err := repository.Connect(ctx, cfg.Database)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	ctxx, cancelClose := context.WithTimeout(context.Background(), time.Second*1)
	defer cancelClose()
	defer db.Disconnect(ctxx)

	callbackOnErr := func(err error) {
		fmt.Println("logger error: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("fatal error: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, db, stdoutwriter.Logger{})

	go func() {
		<-c
		cancel()
	}()

	verify := wallet.NewVerifier()

	seal := aeswrapper.New()
	fo := fileoperations.New(cfg.FileOperator, seal)

	wh := webhooks.New(log)

	wl, err := fo.ReadWallet()
	if err != nil {
		log.Error(err.Error())
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	go func() {
		if err := telemetry.Run(ctx, cancel); err != nil {
			log.Error(err.Error())
		}
	}()

	if err := validator.Run(ctx, cfg.Validator, db, log, verify, wh, &wl, dataProvider); err != nil {
		log.Error(err.Error())
	}
}
