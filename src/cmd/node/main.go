package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"

	"github.com/bartossh/Computantis/src/accountant"
	"github.com/bartossh/Computantis/src/aeswrapper"
	"github.com/bartossh/Computantis/src/cache"
	"github.com/bartossh/Computantis/src/configuration"
	"github.com/bartossh/Computantis/src/dataprovider"
	"github.com/bartossh/Computantis/src/fileoperations"
	"github.com/bartossh/Computantis/src/gossip"
	"github.com/bartossh/Computantis/src/logging"
	"github.com/bartossh/Computantis/src/logo"
	"github.com/bartossh/Computantis/src/natsclient"
	"github.com/bartossh/Computantis/src/notaryserver"
	"github.com/bartossh/Computantis/src/pipe"
	"github.com/bartossh/Computantis/src/stdoutwriter"
	"github.com/bartossh/Computantis/src/telemetry"
	"github.com/bartossh/Computantis/src/wallet"
	"github.com/bartossh/Computantis/src/zincaddapter"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

const usage = `runs the Computantis node that connects in to the Computantis network`

const (
	trxChSize = 800 // set it bigger then expected transaction throughput
	vrxChSize = 800 // set it bigger then expected transaction throughput
)

const (
	maxCacheSizeMB = 128
	maxEntrySize   = 32 * 10_000
)

const gossipTimeout = time.Second * 5

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
		f, _ := os.Create("default.pgo")
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	var writer io.Writer
	zinc, err := zincaddapter.New(cfg.ZincLogger)
	if err != nil {
		fmt.Printf("Failed to connect to zincsearch due to %s, logging to stdout.\n", err)
		writer = &stdoutwriter.Logger{}
		if !errors.Is(err, zincaddapter.ErrEmptyAddressProvided) {
			c <- os.Interrupt
			return
		}
	} else {
		writer = &zinc
	}
	log := logging.New(callbackOnErr, callbackOnFatal, writer)
	dataProvider := dataprovider.New(ctx, cfg.DataProvider)
	verifier := wallet.NewVerifier()
	h := fileoperations.New(cfg.FileOperator, aeswrapper.New())
	wlt, err := h.ReadWallet()
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}

	acc, err := accountant.NewAccountingBook(ctx, cfg.Accountant, &verifier, &wlt, &log)
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}

	tele, err := telemetry.Run(ctx, cancel, 2112)
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}

	juggler := pipe.New(trxChSize, vrxChSize)
	defer juggler.Close()
	hippo, err := cache.New(maxEntrySize, maxCacheSizeMB)
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}
	defer hippo.Close()

	flash, err := cache.NewFlash()
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}
	defer flash.Close()

	pub, err := natsclient.PublisherConnect(cfg.Nats)
	switch err {
	case nil:
		defer func() {
			if err := pub.Disconnect(); err != nil {
				log.Error(err.Error())
			}
		}()
	case natsclient.ErrEmptyAddressProvided:
		log.Error(err.Error())
		time.Sleep(time.Second)
	default:
		log.Error(err.Error())
		time.Sleep(time.Second)
		c <- os.Interrupt
		return
	}

	go func() {
		err = gossip.RunGRPC(ctx, cfg.Gossip, &log, gossipTimeout, &wlt, &verifier, acc, hippo, flash, juggler)
		if err != nil {
			log.Error(err.Error())
			time.Sleep(time.Second)
			c <- os.Interrupt
			return
		}
	}()

	err = notaryserver.Run(ctx, cfg.NotaryServer, pub, dataProvider, tele, &log, &verifier, acc, hippo, juggler)
	if err != nil {
		log.Error(err.Error())
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second)
}
