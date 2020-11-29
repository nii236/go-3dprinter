package main

import (
	"context"
	"errors"
	"fmt"
	"go-3dprint/agent"
	"go-3dprint/seed"
	"go-3dprint/server"
	"net/http"
	"os"
	"time"

	"github.com/avast/retry-go"

	_ "github.com/lib/pq"

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
				Name:  "db",
				Usage: "DB admin commands",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "seed", EnvVars: []string{"GOPRINT_DATABASE_SEED"}, Usage: "Seed the database"},
					&cli.StringFlag{Name: "database_user", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_USER"}, Usage: "The database user"},
					&cli.StringFlag{Name: "database_pass", Value: "dev", EnvVars: []string{"GOPRINT_DATABASE_PASS"}, Usage: "The database pass"},
					&cli.StringFlag{Name: "database_host", Value: "localhost", EnvVars: []string{"GOPRINT_DATABASE_HOST"}, Usage: "The database host"},
					&cli.StringFlag{Name: "database_port", Value: "5432", EnvVars: []string{"GOPRINT_DATABASE_PORT"}, Usage: "The database port"},
					&cli.StringFlag{Name: "database_name", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_NAME"}, Usage: "The database name"},
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
					if c.Bool("seed") {
						log.Infow("Seeding database")
						return seed.Run()
					}
					return terror.New(errors.New("no command provided"), "")
				},
			},
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
						Name:    "websocket_host",
						Usage:   "Set the websocket host",
						EnvVars: []string{"WEBSOCKET_HOST"},
						Value:   "localhost",
					},
					&cli.StringFlag{
						Name:    "websocket_port",
						Usage:   "Set the websocket port",
						EnvVars: []string{"WEBSOCKET_PORT"},
						Value:   "8080",
					},
					&cli.IntFlag{
						Name:    "baud_rate",
						Usage:   "Set the baud rate",
						EnvVars: []string{"BAUD_RATE"},
						Value:   115200,
					},
					&cli.StringFlag{
						Name:     "serial_device",
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
					&cli.StringFlag{Name: "database_user", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_USER"}, Usage: "The database user"},
					&cli.StringFlag{Name: "database_pass", Value: "dev", EnvVars: []string{"GOPRINT_DATABASE_PASS"}, Usage: "The database pass"},
					&cli.StringFlag{Name: "database_host", Value: "localhost", EnvVars: []string{"GOPRINT_DATABASE_HOST"}, Usage: "The database host"},
					&cli.StringFlag{Name: "database_port", Value: "5432", EnvVars: []string{"GOPRINT_DATABASE_PORT"}, Usage: "The database port"},
					&cli.StringFlag{Name: "database_name", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_NAME"}, Usage: "The database name"},
				},
				Action: func(c *cli.Context) error {
					log.Infow("Starting dev mode", "addr", c.String("addr"), "baud_rate", c.Int("baud_rate"), "serial_device", c.String("serial_device"), "websocket_host", c.String("websocket_host"), "websocket_port", c.String("websocket_port"))
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
					return devCommand(
						c.Context,
						c.String(("addr")),
						c.Int("baud_rate"),
						c.String("serial_device"),
						c.String("websocket_host"),
						c.String("websocket_port"),
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
					&cli.StringFlag{Name: "database_user", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_USER"}, Usage: "The database user"},
					&cli.StringFlag{Name: "database_pass", Value: "dev", EnvVars: []string{"GOPRINT_DATABASE_PASS"}, Usage: "The database pass"},
					&cli.StringFlag{Name: "database_host", Value: "localhost", EnvVars: []string{"GOPRINT_DATABASE_HOST"}, Usage: "The database host"},
					&cli.StringFlag{Name: "database_port", Value: "5432", EnvVars: []string{"GOPRINT_DATABASE_PORT"}, Usage: "The database port"},
					&cli.StringFlag{Name: "database_name", Value: "goprint", EnvVars: []string{"GOPRINT_DATABASE_NAME"}, Usage: "The database name"},
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
						Name:    "baud_rate",
						Usage:   "Set the baud rate",
						EnvVars: []string{"BAUD_RATE"},
						Value:   115200,
					},
					&cli.StringFlag{
						Name:    "websocket_host",
						Usage:   "Set the websocket host",
						EnvVars: []string{"WEBSOCKET_HOST"},
						Value:   "localhost",
					},
					&cli.StringFlag{
						Name:    "websocket_port",
						Usage:   "Set the websocket port",
						EnvVars: []string{"WEBSOCKET_PORT"},
						Value:   "8080",
					},
					&cli.StringFlag{
						Name:     "serial_device",
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
				},
				Usage: "Print a gcode file",
				Action: func(c *cli.Context) error {
					return agentCommand(
						c.Context,
						c.Int("baud_rate"),
						c.String("serial_device"),
						c.String("websocket_host"),
						c.String("websocket_port"),
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

	logW := log.With("service", "agent")
	return retry.Do(
		func() error {
			mode := &serial.Mode{BaudRate: baudRate}
			logW.Info("Connecting to serial device...")
			serialconn, err := serial.Open(serialDevice, mode)
			if err != nil {
				return terror.New(err, "")
			}
			logW.Info("Connecting to websocket...")
			wsconn, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s:%s/api/websocket", websocketHost, websocketPort), nil)
			if err != nil {
				return terror.New(err, "")
			}
			defer wsconn.Close(websocket.StatusNormalClosure, "")
			a := agent.New(
				ctx,
				serialconn,
				wsconn,
				websocketHost,
				websocketPort,
			)
			logW.Info("Starting agent...")
			a.Subscribe(ctx)
			return nil
		},
		retry.Attempts(99),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
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
