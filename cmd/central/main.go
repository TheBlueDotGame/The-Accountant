package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bartossh/The-Accountant/blockchain"
	"github.com/bartossh/The-Accountant/bookkeeping"
	"github.com/bartossh/The-Accountant/dataprovider"
	"github.com/bartossh/The-Accountant/repo"
	"github.com/bartossh/The-Accountant/server"
	"github.com/bartossh/The-Accountant/wallet"
)

// TODO: load config from file

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		cancel()
	}()

	db, err := repo.Connect(ctx, "mongodb://root:root@localhost:27017", "accountant")
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

	bookCfg := bookkeeping.Config{
		BlockTransactionsSize: 20,
		BlockWriteTimestamp:   time.Minute,
		Difficulty:            3,
	}

	ladger, err := bookkeeping.NewLedger(bookCfg, blc, db, db, verifier, db)
	if err != nil {
		fmt.Println(err)
		c <- os.Interrupt
		return
	}

	dataProvider := dataprovider.New(ctx, time.Minute*5)

	serverCfg := server.Config{Port: 8080}
	err = server.Run(ctx, &serverCfg, db, ladger, dataProvider)
	if err != nil {
		fmt.Println(err)
	}

}
