package main

import (
	"context"
	"errors"
	"fmt"
	"go-3dprint/messages"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ninja-software/terror"
	"go.bug.st/serial"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Agent struct {
	Conn   *websocket.Conn
	Serial serial.Port
	Busy   bool // No print commands allowed
	*sync.Mutex
	WebsocketHost string
	WebsocketPort string
}

// Subscribe to messages from the server
func (a *Agent) Subscribe(ctx context.Context) error {
	for {
		time.Sleep(500 * time.Millisecond)
		result := &messages.Command{}
		err := wsjson.Read(ctx, a.Conn, result)
		if err != nil {
			terror.Echo(err)
			fmt.Println("reconnecting...")
			wsconn, err := connect(ctx, a.WebsocketHost, a.WebsocketPort)
			if err != nil {
				terror.Echo(err)
				continue
			}
			a.Conn = wsconn
		}
		fmt.Println("RECV:", result)
		go a.ProcessMessage(result)
	}
}

// ProcessMessage prints stuff, mutex locked
func (a *Agent) ProcessMessage(result *messages.Command) {
	if a.Busy {
		terror.Echo(errors.New("agent is busy, try again later"))
		return
	}

	a.Lock()
	ctx := context.Background()
	a.Busy = true
	defer func() {
		a.Busy = false
		a.Unlock()
	}()

	if result.Type == messages.TypeLevelBedTest {
		err := a.Print(ctx, strings.NewReader(GCodeLevelBedTest))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.Type == messages.TypeAutoHome {
		err := a.Print(ctx, strings.NewReader(GCodeAutoHome))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.Type == messages.TypeUnlockPrinter {
		a.Busy = false
	}
}

// Print the gcode
func (a *Agent) Print(ctx context.Context, r io.Reader) error {

	return print(ctx, a.Serial, r)
}
