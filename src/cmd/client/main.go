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

	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/logging"
	"github.com/bartossh/Computantis/src/logo"
	"github.com/bartossh/Computantis/src/stdoutwriter"
	"github.com/bartossh/Computantis/src/telemetry"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/bartossh/Computantis/src/walletapi"
	"github.com/bartossh/Computantis/src/zincaddapter"
)

const usage = `client runs the Computantis wallet API service that serves as a middleware between your application and central node`

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

	_, err = telemetry.Run(ctx, cancel, 2114)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	err = walletapi.Run(ctx, cfg.Client, log, verify, fo, wallet.New) // TODO: Indicate the client wallet source: (new, pem, gob)
	if err != nil {
		log.Error(err.Error())
	}
	time.Sleep(time.Second)
}
