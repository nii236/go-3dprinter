package main

import (
	"context"
	"errors"
	"fmt"
	"go-3dprint/agent"
	"go-3dprint/server"
	"net/http"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/ninja-software/terror"
	"github.com/oklog/run"
	"github.com/urfave/cli/v2"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"go.bug.st/serial"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

var log *zap.SugaredLogger

func init() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	log = logger.Sugar()
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "dev",
				Aliases: []string{"d"},
				Usage:   "Dev defaults",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "addr",
						Usage:   "Addr to host",
						EnvVars: []string{"SERVER_ADDR"},
						Value:   ":8080",
					},
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
					&cli.IntFlag{
						Name:    "baud-rate",
						Usage:   "Set the baud rate",
						EnvVars: []string{"BAUD_RATE"},
						Value:   115200,
					},
					&cli.StringFlag{
						Name:     "serial-device",
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					log.Infow("Starting dev mode", "addr", c.String("addr"), "baud_rate", c.Int("baud-rate"), "serial_device", c.String("serial-device"), "websocket_host", c.String("websocket-host"), "websocket_port", c.String("websocket-port"))
					return devCommand(
						c.Context,
						c.String(("addr")),
						c.Int("baud-rate"),
						c.String("serial-device"),
						c.String("websocket-host"),
						c.String("websocket-port"),
					)
				},
			},
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "addr",
						Usage:   "Addr to host",
						EnvVars: []string{"SERVER_ADDR"},
						Value:   ":8080",
					},
					&cli.StringFlag{Name: "database_user", Value: "totalex", EnvVars: []string{"GOPRINT_DATABASE_USER"}, Usage: "The database user"},
					&cli.StringFlag{Name: "database_pass", Value: "dev", EnvVars: []string{"GOPRINT_DATABASE_PASS"}, Usage: "The database pass"},
					&cli.StringFlag{Name: "database_host", Value: "localhost", EnvVars: []string{"GOPRINT_DATABASE_HOST"}, Usage: "The database host"},
					&cli.StringFlag{Name: "database_port", Value: "5445", EnvVars: []string{"GOPRINT_DATABASE_PORT"}, Usage: "The database port"},
					&cli.StringFlag{Name: "database_name", Value: "totalex", EnvVars: []string{"GOPRINT_DATABASE_NAME"}, Usage: "The database name"},
				},
				Action: func(c *cli.Context) error {
					databaseUser := c.String("database_user")
					databasePass := c.String("database_pass")
					databaseHost := c.String("database_host")
					databasePort := c.String("database_port")
					databaseName := c.String("database_name")
					conn, err := connect(
						databaseUser,
						databasePass,
						databaseHost,
						databasePort,
						databaseName,
					)
					if err != nil {
						return terror.New(err, "")
					}
					boil.SetDB(conn)

					return serveCommand(c.Context, c.String(("addr")))
				},
			},
			{
				Name: "agent",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "baud-rate",
						Usage:   "Set the baud rate",
						EnvVars: []string{"BAUD_RATE"},
						Value:   115200,
					},
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
					return agentCommand(
						c.Context,
						c.Int("baud-rate"),
						c.String("serial-device"),
						c.String("websocket-host"),
						c.String("websocket-port"),
					)
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		terror.Echo(err)
	}

}

func agentCommand(ctx context.Context, baudRate int, serialDevice, websocketHost, websocketPort string) error {
	mode := &serial.Mode{BaudRate: baudRate}
	log.Info("Connecting to serial device...")
	serialconn, err := serial.Open(serialDevice, mode)
	if err != nil {
		return terror.New(err, "")
	}
	wsconn, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s:%s", websocketHost, websocketPort), nil)
	if err != nil {
		return terror.New(err, "")
	}
	defer wsconn.Close(websocket.StatusNormalClosure, "")
	a := &agent.Agent{
		Conn:          wsconn,
		Serial:        serialconn,
		Busy:          false,
		Mutex:         &sync.Mutex{},
		WebsocketHost: websocketHost,
		WebsocketPort: websocketPort,
	}
	err = a.Subscribe(ctx)
	if err != nil {
		return terror.New(err, "")
	}
	return nil
}
func serveCommand(ctx context.Context, addr string) error {
	r := server.Routes()
	return http.ListenAndServe(addr, r)
}
func devCommand(ctx context.Context, addr string, baudRate int, serialDevice, websocketHost, websocketPort string) error {
	ctx, cancel := context.WithCancel(ctx)
	g := &run.Group{}
	g.Add(func() error {
		return serveCommand(ctx, addr)
	}, func(error) {
		cancel()
	})
	g.Add(func() error {
		return agentCommand(ctx, baudRate, serialDevice, websocketHost, websocketPort)
	}, func(error) {
		cancel()
	})
	return g.Run()
}

func connect(
	DatabaseUser string,
	DatabasePass string,
	DatabaseHost string,
	DatabasePort string,
	DatabaseName string,

) (*sqlx.DB, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		DatabaseUser,
		DatabasePass,
		DatabaseHost,
		DatabasePort,
		DatabaseName,
	)
	conn, err := sqlx.Connect("postgres", connString)
	if err != nil {
		return nil, terror.New(err, "could not initialise database")
	}
	if conn == nil {
		err := errors.New("conn is nil")
		return nil, terror.New(err, "could not initialise database")
	}

	return conn, nil
}
