package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-3dprint/messages"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
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

// Agent holds state of the printer
type Agent struct {
	Conn       *websocket.Conn
	Serial     serial.Port
	LoadedFile []byte
	Busy       bool                 // No print commands allowed
	Status     messages.AgentStatus // What printer is currently doing
	*sync.Mutex
	WebsocketHost string
	WebsocketPort string
}

// New agent
func New(ctx context.Context, serialConn serial.Port, wsConn *websocket.Conn, wshost, wsport string) *Agent {
	a := &Agent{
		wsConn,
		serialConn,
		[]byte{},
		false,
		messages.StatusIdle,
		&sync.Mutex{},
		wshost,
		wsport,
	}
	return a
}

// Reconnect creates a new conn to websocket
func (a *Agent) Reconnect(ctx context.Context) error {
	wsconn, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s:%s/api/websocket", a.WebsocketHost, a.WebsocketPort), nil)
	if err != nil {
		return err
	}
	a.Conn = wsconn
	return nil
}

// Subscribe to messages from the server
func (a *Agent) Subscribe(ctx context.Context) {
	// Send agent info to server
	go func() {
		for {
			time.Sleep(1 * time.Second)
			msg := &messages.AsyncCommand{
				RequestID:   uuid.Must(uuid.NewV4()).String(),
				MessageType: messages.TypeInfo,
				RequestType: messages.InfoAgentStatus,
				Payload:     nil,
			}
			b, err := json.Marshal(&messages.AgentInfo{Busy: a.Busy, Status: a.Status})
			if err != nil {
				terror.Echo(err)
				continue
			}
			msg.Payload = b
			err = wsjson.Write(ctx, a.Conn, msg)
			if err != nil {
				terror.Echo(err)
				continue
			}
		}
	}()
	for {
		time.Sleep(500 * time.Millisecond)
		result := &messages.AsyncCommand{}
		err := wsjson.Read(ctx, a.Conn, result)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			fmt.Println("websocket closed")
			return
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		switch result.RequestType {
		case messages.CommandLoad:
			fmt.Println("AGENT LOAD RECEIVED ")
			payload := &messages.PayloadLoadFile{}
			err = json.Unmarshal(result.Payload, payload)
			if err != nil {
				fmt.Println(err)
				continue
			}
			resp, err := http.Get(payload.URL)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if resp.StatusCode != 200 {
				fmt.Println("non 200 code:", resp.StatusCode)
				continue
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				continue
			}
			log.Infow("Downloaded gcode", "bytes", len(a.LoadedFile), "url", payload.URL)
			a.LoadedFile = b
			a.Status = messages.StatusReady

		case messages.CommandStart:
			fmt.Println("AGENT START RECEIVED")
			log.Infow("Loaded gcode", "bytes", len(a.LoadedFile))
			err = print(ctx, a.Serial, bytes.NewReader(a.LoadedFile))
			if err != nil {
				fmt.Println(err)
				continue
			}
		}

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

	if result.RequestType == messages.CommandLevelBedTest {
		err := a.Print(ctx, strings.NewReader(GCodeLevelBedTest))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.RequestType == messages.CommandAutoHome {
		err := a.Print(ctx, strings.NewReader(GCodeAutoHome))
		if err != nil {
			terror.Echo(err)
			return
		}
	}
	if result.RequestType == messages.CommandUnlockPrinter {
		a.Busy = false
	}
}

// Print the gcode
func (a *Agent) Print(ctx context.Context, r io.Reader) error {

	return print(ctx, a.Serial, r)
}
