package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-3dprint/db"
	"go-3dprint/messages"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/cors"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gofrs/uuid"
	"github.com/ninja-software/terror"
	"github.com/volatiletech/sqlboiler/v4/boil"
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

// Controller holds routes and channels
type Controller struct {
	Host       string
	Aggregator chan *messages.AsyncCommand
	Sessions   map[string]*Session
	*sync.Mutex
}

// Session holds two channels for bidirectional communication
type Session struct {
	Info   *messages.AgentInfo
	Agent  chan *messages.AsyncCommand
	Server chan *messages.AsyncCommand
}

// Routes for the master server
func Routes(serverHost string) chi.Router {
	c := &Controller{
		Host:     serverHost,
		Sessions: map[string]*Session{},
		Mutex:    &sync.Mutex{},
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Route("/api", func(r chi.Router) {

		r.HandleFunc("/websocket", WithError(c.websocketHandler))
		r.Get("/printer/sessions", WithError(c.printerSessions))
		r.Get("/printer/info", WithError(c.printerInfo))

		r.Post("/command/levelbedtest", WithError(c.commandLevelBedTest))
		r.Post("/command/autohome", WithError(c.commandAutoHome))
		r.Post("/command/unlock", WithError(c.commandUnlock))

		r.Post("/command/load", WithError(c.commandLoad))
		r.Post("/command/start", WithError(c.commandStart))
		r.Post("/command/pause", WithError(c.commandPause))
		r.Post("/command/cancel", WithError(c.commandCancel))

		r.Get("/gcodes", WithError(c.gcodesList))
		r.Post("/gcodes/upload", WithError(c.gcodesUpload))
		r.Get("/gcodes/download", WithError(c.gcodesDownload))
	})

	return r
}

func (c *Controller) printerInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		return http.StatusBadRequest, terror.New(errors.New("session id not provided"), "")
	}
	currentSession := c.Sessions[sessionID]
	resp := &APIResponse{}
	b, err := json.Marshal(currentSession.Info)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	resp.Payload = b
	err = json.NewEncoder(w).Encode(resp)
	if sessionID == "" {
		return http.StatusBadRequest, terror.New(err, "")
	}
	return http.StatusOK, nil
}

// APIResponse generic container for api response
type APIResponse struct {
	Payload json.RawMessage `json:"payload"`
}

func (c *Controller) printerSessions(w http.ResponseWriter, r *http.Request) (int, error) {
	result := []string{}
	for id := range c.Sessions {
		result = append(result, id)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	err = json.NewEncoder(w).Encode(&APIResponse{Payload: b})
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	return http.StatusOK, nil
}

// LoadCommand instructs printer on session ID to download file ID into memory
type LoadCommand struct {
	SessionID string `json:"session_id"`
	FileID    string `json:"file_id"`
}

func (c *Controller) commandLoad(w http.ResponseWriter, r *http.Request) (int, error) {
	req := &LoadCommand{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	if req.SessionID == "" || req.FileID == "" {
		fmt.Printf("%+v", req)
		return http.StatusBadRequest, terror.New(errors.New("session id or file id not provided"), "")
	}

	payload := &messages.PayloadLoadFile{
		ID:  req.FileID,
		URL: fmt.Sprintf("%s/api/gcodes/download?file_id=%s", c.Host, req.FileID),
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	chs := c.Sessions[req.SessionID]
	chs.Agent <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), MessageType: messages.TypeCommand, RequestType: messages.CommandLoad, Payload: b}

	w.Write([]byte("OK"))
	return http.StatusOK, nil
}

// SessionRequest is a generic struct that holds session ID
type SessionRequest struct {
	SessionID string `json:"sessionId"`
}

// StartRequest tells printer to start printing
type StartRequest struct {
	SessionID string `json:"sessionId"`
	FileID    string `json:"fileId"`
}

func (c *Controller) commandStart(w http.ResponseWriter, r *http.Request) (int, error) {
	req := &SessionRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	if req.SessionID == "" {
		fmt.Printf("%+v", req)
		return http.StatusBadRequest, terror.New(errors.New("session id or file id not provided"), "")
	}
	chs := c.Sessions[req.SessionID]
	chs.Agent <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), MessageType: messages.TypeCommand, RequestType: messages.CommandStart}
	return http.StatusOK, nil
}
func (c *Controller) commandPause(w http.ResponseWriter, r *http.Request) (int, error) {
	// c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandPause}
	return http.StatusOK, nil
}
func (c *Controller) commandCancel(w http.ResponseWriter, r *http.Request) (int, error) {
	// c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandCancel}
	return http.StatusOK, nil
}
func (c *Controller) gcodesList(w http.ResponseWriter, r *http.Request) (int, error) {
	result, err := db.Gcodes().AllG()
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	b, err := json.Marshal(result)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}

	err = json.NewEncoder(w).Encode(&APIResponse{Payload: b})
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	return http.StatusOK, nil
}
func (c *Controller) gcodesDownload(w http.ResponseWriter, r *http.Request) (int, error) {
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		return http.StatusBadRequest, terror.New(errors.New("no file_id"), "")
	}
	gc, err := db.FindGcodeG(fileID)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	blob, err := db.FindBlobG(gc.BlobID)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.html"`, gc.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(blob.Data)))
	w.Write(blob.Data)
	return http.StatusOK, nil
}
func (c *Controller) gcodesUpload(w http.ResponseWriter, r *http.Request) (int, error) {

	err := r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "failed to parse multipart message")
	}
	file, header, err := r.FormFile("file")
	defer file.Close()
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	blob := &db.Blob{Data: b, FileName: header.Filename, FileSizeBytes: header.Size}
	err = blob.InsertG(boil.Infer())
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	gcode := &db.Gcode{
		Name:   header.Filename,
		BlobID: blob.ID,
	}
	err = gcode.InsertG(boil.Infer())
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}

	return http.StatusOK, nil
}

func (c *Controller) websocketHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	wsconn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	defer wsconn.Close(websocket.StatusNormalClosure, "Unknown")
	sessionID := uuid.Must(uuid.NewV4()).String()
	agentChan := make(chan *messages.AsyncCommand)
	serverChan := make(chan *messages.AsyncCommand)

	fmt.Println("New connection request")
	c.Lock()
	c.Sessions[sessionID] = &Session{&messages.AgentInfo{Busy: false, Status: messages.StatusUnknown}, agentChan, serverChan}
	currentSession := c.Sessions[sessionID]
	c.Unlock()
	defer func() {
		c.Lock()
		delete(c.Sessions, sessionID)
		fmt.Println("Session removed")
		c.Unlock()
	}()
	fmt.Println("Session established")

	go func() {
		for {
			// Handle messages coming in from Agent to be processed
			time.Sleep(500 * time.Millisecond)
			result := &messages.AsyncCommand{}
			err := wsjson.Read(r.Context(), wsconn, result)
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				fmt.Println("websocket closed")
				continue
			}
			if err != nil {
				fmt.Println(err)
				continue
			}
			// fmt.Println(string(result.Payload))
			agentInfo := &messages.AgentInfo{}
			err = json.Unmarshal(result.Payload, agentInfo)
			if err != nil {
				fmt.Println(err)
				continue
			}
			currentSession.Info = agentInfo

		}
	}()
	for {
		select {
		case msg := <-agentChan:
			// Handle messages to be forwarded to Agent
			err = writeTimeout(r.Context(), 100*time.Second, wsconn, msg)
			if err != nil {
				return http.StatusBadRequest, terror.New(err, "")
			}
		}
	}
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, v interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wsjson.Write(ctx, c, v)
}

// LevelBedTest will send level bed command
func (c *Controller) commandLevelBedTest(w http.ResponseWriter, r *http.Request) (int, error) {
	// c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.LevelBedTest}
	return http.StatusOK, nil
}

// AutoHome will send level bed command
func (c *Controller) commandAutoHome(w http.ResponseWriter, r *http.Request) (int, error) {
	req := &SessionRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	if req.SessionID == "" {
		fmt.Printf("%+v", req)
		return http.StatusBadRequest, terror.New(errors.New("session id not provided"), "")
	}
	chs := c.Sessions[req.SessionID]
	chs.Agent <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), MessageType: messages.TypeCommand, RequestType: messages.CommandAutoHome}
	return http.StatusOK, nil
}

// commandUnlock will unlock its mutex
func (c *Controller) commandUnlock(w http.ResponseWriter, r *http.Request) (int, error) {
	// c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.UnlockPrinter}
	return http.StatusOK, nil
}
