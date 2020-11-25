package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"

	"github.com/ninja-software/terror"

	"github.com/256dpi/gcode"

	"github.com/urfave/cli/v2"
)

// GCodeHome is the script for testing and autohoming
const GCodeHome = `M201 X500 Y500 Z100 E5000 ; sets maximum accelerations, mm/sec^2
M203 X500 Y500 Z10 E60 ; sets maximum feedrates, mm/sec
M204 P500 R1000 T500 ; sets acceleration (P, T) and retract acceleration (R), mm/sec^2
M205 X8.00 Y8.00 Z0.40 E5.00 ; sets the jerk limits, mm/sec
M205 S0 T0 ; sets the minimum extruding and travel feed rate, mm/sec
M107 ; disable fan
G90 ; use absolute coordinatsces
M83 ; extruder relative mode
G28 ; home all`

// RespBusy returns from the printer if its busy
const RespBusy = "echo:busy: processing\n"

// RespOK returns from the printer if its ready for the next command
const RespOK = "ok\n"

func main() {

	err := (&cli.App{
		Commands: []*cli.Command{
			{
				Name: "print",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "serial-port",
						Aliases:  []string{"p"},
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "input-file",
						Aliases:  []string{"i"},
						Usage:    "Set the gcode file",
						EnvVars:  []string{"INPUT_FILE"},
						Required: true,
					},
				},
				Aliases: []string{"p"},
				Usage:   "Print a gcode file",
				Action: func(c *cli.Context) error {
					mode := &serial.Mode{BaudRate: 115200}
					s, err := serial.Open(c.String("serial-port"), mode)
					if err != nil {
						return err
					}
					f, err := os.Open(c.String("input-file"))
					if err != nil {
						return terror.New(err, "")
					}

					return print(s, f)
				},
			}, {
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "serial-port",
						Aliases:  []string{"p"},
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
				},
				Aliases: []string{"l"},
				Usage:   "Print a level test",
				Action: func(c *cli.Context) error {
					mode := &serial.Mode{BaudRate: 115200}
					s, err := serial.Open(c.String("serial-port"), mode)
					if err != nil {
						return err
					}

					return print(s, strings.NewReader(GCodeBedLevel))
				},
			},
			{
				Name: "home",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "serial-port",
						Aliases:  []string{"p"},
						Usage:    "Set the serial port",
						EnvVars:  []string{"SERIAL_PORT"},
						Required: true,
					},
				},
				Aliases: []string{"h"},
				Usage:   "auto home printer",
				Action: func(c *cli.Context) error {
					mode := &serial.Mode{BaudRate: 115200}
					s, err := serial.Open(c.String("serial-port"), mode)
					if err != nil {
						return err
					}

					return print(s, strings.NewReader(GCodeHome))
				},
			},
			{
				Name:    "devices",
				Aliases: []string{"d"},
				Usage:   "list devices and baudrates",
				Action: func(c *cli.Context) error {

					ports, err := enumerator.GetDetailedPortsList()
					if err != nil {
						return terror.New(err, "")
					}
					if len(ports) == 0 {
						return terror.New(errors.New("no serial ports found"), "")
					}
					for _, port := range ports {
						fmt.Printf("Found port: %s\n", port.Name)
						if port.IsUSB {
							fmt.Printf("   USB ID     %s:%s\n", port.VID, port.PID)
							fmt.Printf("   USB serial %s\n", port.SerialNumber)
						}
					}
					return nil
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		terror.Echo(err)
	}

}

func print(s serial.Port, f io.Reader) error {

	gfile, err := gcode.ParseFile(f)
	if err != nil {
		return terror.New(err, "")
	}

	for _, l := range gfile.Lines {
		if strings.HasPrefix(l.String(), ";") {
			continue
		}
		if l.String() == "" {
			continue
		}
		if len(l.String()) == 0 {
			continue
		}
		if !unicode.IsLetter(rune(l.String()[0])) {
			continue
		}
		fmt.Print("SEND: ", l.String())
		_, err = s.Write([]byte(l.String()))
		if err != nil {
			return terror.New(err, "")
		}

		brdr := bufio.NewReader(s)
		result, err := brdr.ReadString('\n')
		if err != nil {
			return terror.New(err, "")
		}
		fmt.Print("RECV: ", result)
		if unicode.IsLetter(rune(result[0])) && unicode.IsUpper(rune(result[0])) {
			continue
		}

		if result != RespOK {
			for {
				brdr := bufio.NewReader(s)
				result, err := brdr.ReadString('\n')
				if err != nil {
					return terror.New(err, "")
				}
				fmt.Print("RECV: ", result)
				if result == RespOK {
					break
				}
				if unicode.IsLetter(rune(result[0])) && unicode.IsUpper(rune(result[0])) {
					break
				}
			}
		}
	}
	return nil
}
