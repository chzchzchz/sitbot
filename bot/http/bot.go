package http

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"

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
	bot := h.g.Lookup(id)
	if v, ok := r.Header["Content-Type"]; ok && v[0] == "application/octet-stream" {
		q := r.URL.Query()
		tgt := q["target"][0]
		if tgt == "" {
			return io.EOF
		}
		for _, l := range strings.Split(string(b), "\n") {
			m := &BotPostMessage{
				Message: irc.Message{
					Command: irc.PRIVMSG, Params: []string{tgt, l}}}
			if err := h.postMessage(bot, m); err != nil {
				return err
			}
		}
	} else {
		m := &BotPostMessage{}
		if err := json.Unmarshal(b, m); err != nil {
			return err
		}
		return h.postMessage(bot, m)
	}
	return nil
}

func (h *botHandler) postMessage(b *bot.Bot, m *BotPostMessage) error {
	if len(m.Command) == 0 || b == nil {
		return io.EOF
	} else if m.Command == irc.KILL && len(m.Params) == 0 {
		return b.Tasks.Kill(m.TaskId)
	}
	return b.Write(m.TaskId, m.Message)
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
