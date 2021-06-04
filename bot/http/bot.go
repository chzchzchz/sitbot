package http

import (
	"encoding/json"
	"io"
	"net/http"
	"path"

	"github.com/chzchzchz/sitbot/bot"
	"gopkg.in/sorcix/irc.v2"
)

type botHandler struct {
	g *bot.Gang
}

func (h *botHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, id := path.Split(r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		errWrap(w, r, func() error { return h.get(id, w, r) })
	case http.MethodDelete:
		errWrap(w, r, func() error { return h.g.Delete(id) })
	case http.MethodPost:
		postWrap(w, r, func(b []byte) error { return h.post(id, w, r, b) })
	default:
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
	}
}

type BotPostMessage struct {
	TaskId bot.TaskId
	irc.Message
}

func (h *botHandler) post(id string, w http.ResponseWriter, r *http.Request, b []byte) error {
	m := &BotPostMessage{}
	if err := json.Unmarshal(b, m); err != nil {
		return err
	}
	bot := h.g.Lookup(id)
	if len(m.Command) == 0 || bot == nil {
		return io.EOF
	} else if m.Command == irc.KILL && len(m.Params) == 0 {
		return bot.Tasks.Kill(m.TaskId)
	}
	return bot.Write(m.TaskId, m.Message)
}

func (h *botHandler) get(id string, w http.ResponseWriter, r *http.Request) error {
	bot := h.g.Lookup(id)
	if bot == nil {
		return io.EOF
	}
	b, err := json.Marshal(bot)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	return err
}
