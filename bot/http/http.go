package http

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
)

func errWrap(w http.ResponseWriter, r *http.Request, f func() error) (err error) {
	defer func() {
		r.Body.Close()
		if err != nil {
			slog.Error("http request failed", "method", r.Method, "path", r.URL.Path, "err", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			slog.Debug("http request ok", "method", r.Method, "path", r.URL.Path)
		}
	}()
	return f()
}

func postWrap(w http.ResponseWriter, r *http.Request, f func(b []byte) error) (err error) {
	defer func() {
		r.Body.Close()
		if err != nil {
			slog.Error("http request failed", "method", r.Method, "path", r.URL.Path, "err", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			slog.Debug("http request ok", "method", r.Method, "path", r.URL.Path)
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

func writeJSON(v interface{}, w http.ResponseWriter) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
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
	slog.Info("http request", "prefix", h.pfx, "method", r.Method, "path", r.URL.Path)
	h.ServeHTTP(w, r)
}
