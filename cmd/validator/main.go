package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/client"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/validator"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/webhooks"
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

	db, _, err := cfg.Database.Connect(ctx)
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

	log := logging.New(callbackOnErr, callbackOnFatal, db)

	go func() {
		<-c
		cancel()
	}()

	verify := wallet.NewVerifier()

	seal := aeswrapper.New()
	fo := fileoperations.New(cfg.FileOperator, seal)

	httpClient := client.NewClient("", time.Second*5, verify, fo, wallet.New)

	wh := webhooks.New(httpClient, log)

	wl, err := fo.ReadWallet()
	if err != nil {
		log.Error(err.Error())
	}

	if err := validator.Run(ctx, cfg.Validator, db, log, verify, wh, &wl); err != nil {
		log.Error(err.Error())
		fmt.Println(err.Error())
	}
}
