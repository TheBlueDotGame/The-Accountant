package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/logging"
	"github.com/bartossh/Computantis/repo"
	"github.com/bartossh/Computantis/validator"
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

	go func() {
		<-c
		cancel()
	}()

	if err := validator.Run(ctx, cfg.Validator, db, log); err != nil {
		log.Error(err.Error())
		fmt.Println(err.Error())
	}
}