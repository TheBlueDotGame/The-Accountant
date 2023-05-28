package main

import (
	"errors"
	"os"

	"github.com/bartossh/Computantis/configuration"
	"github.com/bartossh/Computantis/logo"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

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
		Name:  "emulator",
		Usage: "Emulates device publisher or subscriber.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Destination: &file,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "publisher",
				Aliases: []string{"p"},
				Usage:   "starts publisher",
				Action: func(cCtx *cli.Context) error {
					cfg, err := configurator()
					pterm.Info.Println(cfg)
					return err
				},
			},
			{
				Name:    "subscriber",
				Aliases: []string{"s"},
				Usage:   "starts subscriber",
				Action: func(cCtx *cli.Context) error {
					cfg, err := configurator()
					pterm.Info.Println(cfg)
					return err
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}
