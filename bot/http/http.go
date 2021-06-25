package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/chzchzchz/sitbot/bot"
)

func errWrap(w http.ResponseWriter, r *http.Request, f func() error) (err error) {
	defer func() {
		r.Body.Close()
		if err != nil {
			slog.Error("http request failed", "method", r.Method, "path", r.URL.Path, "err", err)
			http.Error(w, err.Error(), statusForError(err))
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
			http.Error(w, err.Error(), statusForError(err))
		} else {
			slog.Debug("http request ok", "method", r.Method, "path", r.URL.Path)
		}
	}()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return f(b)
}

func statusForError(err error) int {
	if errors.Is(err, bot.ErrNotFound) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
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
