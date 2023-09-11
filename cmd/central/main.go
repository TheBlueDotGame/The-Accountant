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

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/blockchain"
	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/logo"
	"github.com/bartossh/Computantis/reactive"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/telemetry"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/zincaddapter"
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

	ctxx, cancelClose := context.WithTimeout(context.Background(), time.Second*1)
	defer cancelClose()

	trxDB, err := repository.Connect(ctx, cfg.StorageConfig.TransactionDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer trxDB.Disconnect(ctxx)

	blockchainDB, err := repository.Connect(ctx, cfg.StorageConfig.BlockchainDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer blockchainDB.Disconnect(ctxx)

	blockchainNotifier, err := repository.Subscribe(ctx, cfg.StorageConfig.BlockchainDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer blockchainNotifier.Close()

	nodeRegisterDB, err := repository.Connect(ctx, cfg.StorageConfig.NodeRegisterDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer nodeRegisterDB.Disconnect(ctxx)

	addressDB, err := repository.Connect(ctx, cfg.StorageConfig.AddressDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer addressDB.Disconnect(ctxx)

	tokenDB, err := repository.Connect(ctx, cfg.StorageConfig.TokenDatabase)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	defer tokenDB.Disconnect(ctxx)

	callbackOnErr := func(err error) {
		fmt.Println("Error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("Error with logger: %s", err))
	}

	zinc, err := zincaddapter.New(cfg.ZincLogger)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	log := logging.New(callbackOnErr, callbackOnFatal, stdoutwriter.Logger{}, &zinc)

	if err := blockchain.GenesisBlock(ctx, blockchainDB); err != nil {
		fmt.Printf("Mining genesis block error: %s\n", err)
	}

	blc, err := blockchain.New(ctx, blockchainDB)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	verifier := wallet.NewVerifier()
	rxBlock := reactive.New[block.Block](rxBufferSize)
	rxTrxIssuer := reactive.New[string](rxBufferSize)

	ladger, err := bookkeeping.New(
		cfg.Bookkeeper, trxDB, blc, nodeRegisterDB, blockchainNotifier,
		addressDB, verifier, log, rxBlock, rxTrxIssuer)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	tele, err := telemetry.Run(ctx, cancel, 0)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	err = server.Run(
		ctx, cfg.Server, trxDB, nodeRegisterDB, addressDB, tokenDB, ladger,
		dataProvider, tele, log, rxBlock.Subscribe(), rxTrxIssuer.Subscribe())
	if err != nil {
		log.Error(err.Error())
	}
}
