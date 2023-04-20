package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bartossh/The-Accountant/blockchain"
	"github.com/bartossh/The-Accountant/bookkeeping"
	"github.com/bartossh/The-Accountant/configuration"
	"github.com/bartossh/The-Accountant/dataprovider"
	"github.com/bartossh/The-Accountant/repo"
	"github.com/bartossh/The-Accountant/server"
	"github.com/bartossh/The-Accountant/wallet"
)

func main() {
	cfg, err := configuration.Read("server_settings.yaml") // TODO: take from stdin
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

	if err := blockchain.GenesisBlock(ctx, db); err != nil {
		fmt.Println(err)
	}

	blc, err := blockchain.NewBlockchain(ctx, db)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	verifier := wallet.Helper{}

	ladger, err := bookkeeping.NewLedger(cfg.Bookkeeper, blc, db, db, verifier, db)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, cfg.DataProvider)

	err = server.Run(ctx, cfg.Server, db, ladger, dataProvider)
	if err != nil {
		fmt.Println(err)
	}

}
