package server

import (
	"net/http"

	"github.com/ninja-software/terror"
)

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
