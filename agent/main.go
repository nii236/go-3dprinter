package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/ninja-software/terror"
	"github.com/urfave/cli/v2"
	"go.bug.st/serial"
	"nhooyr.io/websocket"
)

func main() {

	app := (&cli.App{
		Commands: []*cli.Command{
			{
				Name: "agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "websocket-host",
						Usage:   "Set the websocket host",
						EnvVars: []string{"WEBSOCKET_HOST"},
						Value:   "localhost",
					},
					&cli.StringFlag{
						Name:    "websocket-port",
						Usage:   "Set the websocket port",
						EnvVars: []string{"WEBSOCKET_PORT"},
						Value:   "8080",
					},
					&cli.StringFlag{
						Name:     "serial-device",
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
				},
				Usage: "Print a gcode file",
				Action: func(c *cli.Context) error {
					fmt.Println("serial device:", c.String("serial-device"))
					fmt.Println("websocket-host:", c.String("websocket-host"))
					fmt.Println("websocket-port:", c.String("websocket-port"))
					mode := &serial.Mode{BaudRate: 115200}
					fmt.Println("open serial device")
					serialconn, err := serial.Open(c.String("serial-device"), mode)
					if err != nil {
						return terror.New(err, "")
					}
					ctx := context.Background()
					wsconn, err := connect(ctx, c.String("websocket-host"), c.String("websocket-port"))
					if err != nil {
						return terror.New(err, "")
					}
					defer wsconn.Close(websocket.StatusNormalClosure, "")
					a := &Agent{
						wsconn,
						serialconn,
						false,
						&sync.Mutex{},
						c.String("websocket-host"),
						c.String("websocket-port"),
					}
					err = a.Subscribe(ctx)
					if err != nil {
						return terror.New(err, "")
					}
					return nil
				},
			},
		},
	})
	err := app.Run(os.Args)
	if err != nil {
		terror.Echo(err)
	}
}

func connect(ctx context.Context, wshost, wsport string) (*websocket.Conn, error) {
	c, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s:%s", wshost, wsport), nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}
