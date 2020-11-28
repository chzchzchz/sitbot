package http

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func errWrap(w http.ResponseWriter, r *http.Request, f func() error) (err error) {
	defer func() {
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}()
	return f()
}

func postWrap(w http.ResponseWriter, r *http.Request, f func(b []byte) error) (err error) {
	defer func() {
		r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return f(b)
}

func ok(w http.ResponseWriter) error {
	_, err := io.WriteString(w, `{ "error" : 0 }`)
	return err
}

type logHandler struct {
	h   http.Handler
	pfx string
}

func newLogHandler(pfx string, h http.Handler) http.Handler {
	return &logHandler{h, pfx}
}

func (h *logHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s: %+v", h.pfx, *r)
	h.ServeHTTP(w, r)
}
