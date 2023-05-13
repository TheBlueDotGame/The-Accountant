package main

import (
	"context"
	"fmt"
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
	"github.com/bartossh/Computantis/repopostgre"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/wallet"
)

const (
	rxBufferSize = 100
)

func main() {
	cfg, err := configuration.Read("server_settings.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	db, err := repopostgre.Connect(ctx, cfg.Database)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}
	sub, err := repopostgre.Subscribe(ctx, cfg.Database)
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
	rx := reactive.New[block.Block](rxBufferSize)

	ladger, err := bookkeeping.New(cfg.Bookkeeper, blc, db, db, verifier, db, log, rx, sub)
	if err != nil {
		log.Error(err.Error())
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	err = server.Run(ctx, cfg.Server, db, ladger, dataProvider, log, rx.Subscribe())
	if err != nil {
		log.Error(err.Error())
	}
}
