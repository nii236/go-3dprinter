package agent

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
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var log *zap.SugaredLogger

func init() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	log = logger.Sugar()
}

// New agent
func New(ctx context.Context, serialConn serial.Port, wsConn *websocket.Conn, wshost, wsport string) (*Agent, error) {
	a := &Agent{
		wsConn,
		serialConn,
		false,
		&sync.Mutex{},
		wshost,
		wsport,
	}
	return a, nil
}

type Agent struct {
	Conn   *websocket.Conn
	Serial serial.Port
	Busy   bool // No print commands allowed
	*sync.Mutex
	WebsocketHost string
	WebsocketPort string
}

func (a *Agent) Reconnect(ctx context.Context) error {
	wsconn, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s:%s", a.WebsocketHost, a.WebsocketPort), nil)
	if err != nil {
		return err
	}
	a.Conn = wsconn
	return nil
}

// Subscribe to messages from the server
func (a *Agent) Subscribe(ctx context.Context) error {
	for {
		time.Sleep(500 * time.Millisecond)
		result := &messages.AsyncCommand{}
		err := wsjson.Read(ctx, a.Conn, result)
		if err != nil {
			terror.Echo(err)
			log.Info("reconnecting...")
			err = a.Reconnect(ctx)
			if err != nil {
				terror.Echo(err)
				continue
			}
		}
		log.Info("RECV:", result)
		go a.ProcessMessage(result)
	}
}

// ProcessMessage prints stuff, mutex locked
func (a *Agent) ProcessMessage(result *messages.AsyncCommand) {
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

	if result.Type == messages.LevelBedTest {
		err := a.Print(ctx, strings.NewReader(GCodeLevelBedTest))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.Type == messages.AutoHome {
		err := a.Print(ctx, strings.NewReader(GCodeAutoHome))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.Type == messages.UnlockPrinter {
		a.Busy = false
	}
}

// Print the gcode
func (a *Agent) Print(ctx context.Context, r io.Reader) error {

	return print(ctx, a.Serial, r)
}
