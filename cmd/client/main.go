package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/logo"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/telemetry"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/walletapi"
	"github.com/bartossh/Computantis/zincaddapter"
)

const usage = `Client runs wallet API service that serves as a middleware between your application and central node.
Wallet has cryptographic capabilities and uses GOB encoded and EAS encrypted wallet.`

const timeout = time.Second * 5

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
		Name:  "client",
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Destination: &file,
			},
		},
		Action: func(_ *cli.Context) error {
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

	callbackOnErr := func(err error) {
		fmt.Println("error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("error with logger: %s", err))
	}

	zinc, err := zincaddapter.New(cfg.ZincLogger)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{}, &zinc)

	seal := aeswrapper.New()
	fo := fileoperations.New(cfg.FileOperator, seal)

	verify := wallet.NewVerifier()

	_, err = telemetry.Run(ctx, cancel, 0)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	err = walletapi.Run(ctx, cfg.Client, log, timeout, verify, fo, wallet.New) // TODO: Indicate the client wallet source: (new, pem, gob)

	if err != nil {
		log.Error(err.Error())
	}
}
