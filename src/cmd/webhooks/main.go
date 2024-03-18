package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/logging"
	"github.com/bartossh/Computantis/src/logo"
	"github.com/bartossh/Computantis/src/natsclient"
	"github.com/bartossh/Computantis/src/stdoutwriter"
	"github.com/bartossh/Computantis/src/telemetry"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/bartossh/Computantis/src/webhooks"
	"github.com/bartossh/Computantis/src/webhooksserver"
	"github.com/bartossh/Computantis/src/zincaddapter"
)

const usage = `runs The Computantis-Web-Hooks node that is responsible for informing subscribed clients about awiated transactions`

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
		Name:  "webhooks",
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
	if cfg.IsProfiling {
		f, _ := os.Create("default_validator.pgo")
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	callbackOnErr := func(err error) {
		fmt.Println("logger error: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("fatal error: %s", err))
	}

	zinc, err := zincaddapter.New(cfg.ZincLogger)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{}, &zinc)

	go func() {
		<-c
		cancel()
	}()

	verify := wallet.NewVerifier()

	wh := webhooks.New(log)

	_, err = telemetry.Run(ctx, cancel, 2113)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	sub, err := natsclient.SubscriberConnect(cfg.Nats)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}
	defer func() {
		if err := sub.Disconnect(); err != nil {
			log.Error(err.Error())
		}
	}()

	if err := webhooksserver.Run(ctx, cfg.WebhooksServer, sub, &log, &verify, wh); err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
	}
	time.Sleep(time.Second) // Sleep one seccond so all the goroutines will finish. It is important for logging to the external micorservice.
}
