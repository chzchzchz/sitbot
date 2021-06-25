package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/chzchzchz/sitbot/bot"
)

type botletHandler struct {
	g *bot.Gang
}

func (h *botletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, id := path.Split(r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		errWrap(w, r, func() error { return h.get(id, w) })
	case http.MethodPut:
		errWrap(w, r, func() error { return h.put(id, w, r) })
	case http.MethodDelete:
		errWrap(w, r, func() error { return h.g.DeleteBotlet(id) })
	case http.MethodPost:
		postWrap(w, r, func(b []byte) error { return h.post(id, w, b) })
	default:
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
	}
}

func (h *botletHandler) post(id string, w http.ResponseWriter, b []byte) error {
	bl := &bot.Botlet{}
	if err := json.Unmarshal(b, bl); err != nil {
		return err
	}
	if err := setBotletName(id, bl); err != nil {
		return err
	}
	return h.save(w, bl)
}

func (h *botletHandler) put(id string, w http.ResponseWriter, r *http.Request) error {
	bl := &bot.Botlet{}
	if err := json.NewDecoder(r.Body).Decode(bl); err != nil {
		return err
	}
	if err := setBotletName(id, bl); err != nil {
		return err
	}
	return h.save(w, bl)
}

func setBotletName(id string, bl *bot.Botlet) error {
	if id != "" && bl.Name != "" && bl.Name != id {
		return fmt.Errorf("botlet name %q does not match URL name %q", bl.Name, id)
	}
	if bl.Name == "" {
		bl.Name = id
	}
	return nil
}

func (h *botletHandler) save(w http.ResponseWriter, bl *bot.Botlet) error {
	if err := h.g.PostBotlet(bl); err != nil {
		return err
	}
	return ok(w)
}

func (h *botletHandler) get(id string, w http.ResponseWriter) error {
	if bl := h.g.LookupBotlet(id); bl != nil {
		return writeJSON(bl, w)
	}
	http.Error(w, "botlet not found", http.StatusNotFound)
	return nil
}
