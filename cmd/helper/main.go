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

	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/helperserver"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/logo"
	"github.com/bartossh/Computantis/natsclient"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/telemetry"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/webhooks"
	"github.com/bartossh/Computantis/zincaddapter"
)

const usage = `The Helper Computantis API server validates transactions and blocks. In additions Helper offers
web-hook endpoint where any application with valid address can register to listen for new blocks or transactions for 
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
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	statusDB, err := repository.Connect(ctx, cfg.StorageConfig.HelperStatusDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	ctxx, cancelClose := context.WithTimeout(context.Background(), time.Second*1)
	defer cancelClose()
	defer statusDB.Disconnect(ctxx)

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

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	_, err = telemetry.Run(ctx, cancel, 0)
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

	if err := helperserver.Run(ctx, cfg.HelperServer, sub, statusDB, log, verify, wh, dataProvider); err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
	}
}
