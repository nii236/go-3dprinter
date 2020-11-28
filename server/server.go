package server

import (
	"encoding/json"
	"go-3dprint/db"
	"go-3dprint/messages"
	"io/ioutil"
	"net/http"

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
	AsyncCh chan *messages.AsyncCommand
}

// Routes for the master server
func Routes() chi.Router {
	c := &Controller{
		AsyncCh: make(chan *messages.AsyncCommand),
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

	r.Get("/", WithError(c.websocketHandler))

	r.Post("/command/levelbedtest", WithError(c.commandLevelBedTest))
	r.Post("/command/autohome", WithError(c.commandAutoHome))
	r.Post("/command/unlock", WithError(c.commandUnlock))

	r.Post("/command/load", WithError(c.commandLoad))
	r.Post("/command/print", WithError(c.commandPrint))
	r.Post("/command/pause", WithError(c.commandPause))
	r.Post("/command/cancel", WithError(c.commandCancel))

	r.Get("/gcodes/", WithError(c.gcodesList))
	r.Post("/gcodes/upload", WithError(c.gcodesUpload))

	return r
}
func (c *Controller) commandLoad(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandLoad}
	return http.StatusOK, nil
}
func (c *Controller) commandPrint(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandPrint}
	return http.StatusOK, nil
}
func (c *Controller) commandPause(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandPause}
	return http.StatusOK, nil
}
func (c *Controller) commandCancel(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.CommandCancel}
	return http.StatusOK, nil
}
func (c *Controller) gcodesList(w http.ResponseWriter, r *http.Request) (int, error) {
	result, err := db.Gcodes().AllG()
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
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

	gcode := &db.Gcode{}
	err = gcode.InsertG(boil.Infer())
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	err = gcode.SetBlobG(true, &db.Blob{Data: b, FileName: header.Filename, FileSizeBytes: header.Size})
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
	defer wsconn.Close(websocket.StatusNormalClosure, "")

	for {
		select {
		case msg := <-c.AsyncCh:
			err = wsjson.Write(r.Context(), wsconn, msg)
			if err != nil {
				return http.StatusBadGateway, terror.New(err, "")
			}
		}
	}
}

// LevelBedTest will send level bed command
func (c *Controller) commandLevelBedTest(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.LevelBedTest}
	return http.StatusOK, nil
}

// AutoHome will send level bed command
func (c *Controller) commandAutoHome(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.AutoHome}
	return http.StatusOK, nil
}

// commandUnlock will unlock its mutex
func (c *Controller) commandUnlock(w http.ResponseWriter, r *http.Request) (int, error) {
	c.AsyncCh <- &messages.AsyncCommand{RequestID: uuid.Must(uuid.NewV4()).String(), Type: messages.UnlockPrinter}
	return http.StatusOK, nil
}
