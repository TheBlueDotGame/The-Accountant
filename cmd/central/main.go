package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bartossh/Computantis/blockchain"
	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/repo"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/wallet"
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

	db, err := repo.Connect(ctx, cfg.Database)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	callbackOnErr := func(err error) {
		fmt.Println("error with logger: ", err)
	}

	callbackOnFatal := func(err error) {
		panic(fmt.Sprintf("error with logger: %s", err))
	}

	log := logging.New(callbackOnErr, callbackOnFatal, db)

	if err := blockchain.GenesisBlock(ctx, db); err != nil {
		fmt.Println(err)
	}

	blc, err := blockchain.New(ctx, db)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	verifier := wallet.Helper{}

	ladger, err := bookkeeping.New(cfg.Bookkeeper, blc, db, db, verifier, db, log)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	err = server.Run(ctx, cfg.Server, db, ladger, dataProvider, log)
	if err != nil {
		log.Error(err.Error())
		fmt.Println(err)
	}
}
