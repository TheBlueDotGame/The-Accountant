package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/bartossh/Computantis/aeswrapper"
	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/signerservice"
	"github.com/bartossh/Computantis/stdoutwriter"
	"github.com/bartossh/Computantis/wallet"
)

const timeout = time.Second * 5

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

	log := logging.New(callbackOnErr, callbackOnFatal, db, stdoutwriter.Logger{})

	seal := aeswrapper.New()
	fo := fileoperations.New(cfg.FileOperator, seal)

	verify := wallet.NewVerifier()

	err = signerservice.Run(ctx, cfg.SignerService, log, timeout, verify, fo, wallet.New)

	if err != nil {
		log.Error(err.Error())
	}
}
