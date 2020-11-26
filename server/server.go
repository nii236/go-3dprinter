package main

import (
	"go-3dprint/messages"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/ninja-software/terror"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func Routes() chi.Router {
	c := &Controller{
		Ch: make(chan *messages.Command),
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", WithError(c.WSHandler))
	r.Get("/levelbedtest/", WithError(c.LevelBedTest))
	r.Get("/autohome/", WithError(c.AutoHome))
	r.Get("/printlink/", WithError(c.PrintLink))
	r.Get("/unlock/", WithError(c.UnlockPrinter))
	return r
}

type Controller struct {
	Ch chan *messages.Command
}

func WithError(next func(w http.ResponseWriter, r *http.Request) (int, error)) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		code, err := next(w, r)
		if err != nil {
			terror.Echo(err)
			http.Error(w, err.Error(), code)
		}
		w.WriteHeader(code)
	}
	return fn

}
func (c *Controller) WSHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	wsconn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return http.StatusBadRequest, terror.New(err, "")
	}
	defer wsconn.Close(websocket.StatusNormalClosure, "")

	for {
		select {
		case msg := <-c.Ch:
			err = wsjson.Write(r.Context(), wsconn, msg)
			if err != nil {
				return http.StatusBadGateway, terror.New(err, "")
			}
		}
	}
}

// LevelBedTest will send level bed command
func (c *Controller) LevelBedTest(w http.ResponseWriter, r *http.Request) (int, error) {
	c.Ch <- &messages.Command{Type: messages.TypeLevelBedTest}
	return http.StatusOK, nil
}

// AutoHome will send level bed command
func (c *Controller) AutoHome(w http.ResponseWriter, r *http.Request) (int, error) {
	c.Ch <- &messages.Command{Type: messages.TypeAutoHome}
	return http.StatusOK, nil
}

// PrintLink will send level bed command
func (c *Controller) PrintLink(w http.ResponseWriter, r *http.Request) (int, error) {
	c.Ch <- &messages.Command{Type: messages.TypePrintLink}
	return http.StatusOK, nil
}

// UnlockPrinter will unlock its mutex
func (c *Controller) UnlockPrinter(w http.ResponseWriter, r *http.Request) (int, error) {
	c.Ch <- &messages.Command{Type: messages.TypeUnlockPrinter}
	return http.StatusOK, nil
}
