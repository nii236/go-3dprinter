package main

import (
	"net/http"
	"os"

	"github.com/ninja-software/terror"

	"github.com/urfave/cli/v2"
)

func main() {
	err := (&cli.App{
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Server",
				Action: func(c *cli.Context) error {
					r := Routes()
					return http.ListenAndServe(":8080", r)
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		terror.Echo(err)
	}
}
