package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"go.bug.st/serial"

	"github.com/ninja-software/terror"

	"github.com/256dpi/gcode"
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

func print(ctx context.Context, s serial.Port, f io.Reader) error {
	fmt.Println("Start print")

	gfile, err := gcode.ParseFile(f)
	if err != nil {
		return terror.New(err, "")
	}
	fmt.Println("Start sending gcode")
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
