package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/emulator"
	"github.com/bartossh/Computantis/logo"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

func configReader(confFile string) (configuration.Configuration, error) {
	if confFile == "" {
		return configuration.Configuration{}, errors.New("please specify configuration file path with -c <path to file>")
	}

	cfg, err := configuration.Read(confFile)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func closerContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func(cancel context.CancelFunc) {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		cancel()
	}(cancel)

	return ctx, cancel
}

func main() {
	logo.Display()

	var confFile string
	var dataFile string

	app := &cli.App{
		Name:  "emulator",
		Usage: "Emulates device publisher or subscriber.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Destination: &confFile,
			},
			&cli.StringFlag{
				Name:        "data",
				Aliases:     []string{"d"},
				Usage:       "Load data from `FILE` line by line",
				Destination: &dataFile,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "publisher",
				Aliases: []string{"p"},
				Usage:   "Starts publisher emulator",
				Action: func(cCtx *cli.Context) error {
					cfg, err := configReader(confFile)
					if err != nil {
						return err
					}
					data, err := os.ReadFile(dataFile)
					if err != nil {
						return err
					}
					ctx, cancel := closerContext()
					if err := emulator.RunPublisher(ctx, cancel, cfg.Emulator, data); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:    "subscriber",
				Aliases: []string{"s"},
				Usage:   "Starts subscriber emulator",
				Action: func(cCtx *cli.Context) error {
					cfg, err := configReader(confFile)
					if err != nil {
						return err
					}
					data, err := os.ReadFile(dataFile)
					if err != nil {
						return err
					}
					ctx, cancel := closerContext()
					if err := emulator.RunSubscriber(ctx, cancel, cfg.Emulator, data); err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}
