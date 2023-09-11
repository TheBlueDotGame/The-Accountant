package main

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/bartossh/Computantis/generator"
	"github.com/bartossh/Computantis/logo"
)

func main() {
	logo.Display()

	var count, vmin, vmax, mamin, mamax int
	var file string

	app := &cli.App{
		Name:  "generator",
		Usage: "Generates data required by emulator.",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "count",
				Aliases:     []string{"c"},
				Usage:       "Count represents number of data points to generate.",
				Destination: &count,
			},
			&cli.StringFlag{
				Name:        "file",
				Aliases:     []string{"f"},
				Usage:       "Save data file `FILE` in json format.",
				Destination: &file,
			},
			&cli.IntFlag{
				Name:        "vmin",
				Usage:       "Minimum voltage data allowed.",
				Destination: &vmin,
			},
			&cli.IntFlag{
				Name:        "vmax",
				Usage:       "Maximum voltage data allowed.",
				Destination: &vmax,
			},
			&cli.IntFlag{
				Name:        "mamin",
				Usage:       "Minimum mili amps data allowed.",
				Destination: &mamin,
			},
			&cli.IntFlag{
				Name:        "mamax",
				Usage:       "Maximum mili amps data allowed.",
				Destination: &mamax,
			},
		},
		Action: func(cCtx *cli.Context) error {
			return generator.ToJSONFile(file, int64(count), int64(vmin), int64(vmax), int64(mamin), int64(mamax))
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Println(err.Error())
	}
}
