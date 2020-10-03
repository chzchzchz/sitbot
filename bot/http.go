package bot

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sync"

	"gopkg.in/sorcix/irc.v2"
)

type httpHandler struct {
	g    *Gang
	tmpl *template.Template
	mu   sync.RWMutex
}

func GangHandler(g *Gang) http.Handler {
	return &httpHandler{
		g:    g,
		tmpl: template.Must(template.New("bot").Parse("")),
	}
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	tmpl := h.tmpl
	h.mu.RUnlock()
	h.g.LockBots()
	defer h.g.UnlockBots()
	if err := tmpl.Execute(w, h.g); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type BotPostMessage struct {
	TaskId TaskId
	irc.Message
}

func (h *httpHandler) handlePost(w http.ResponseWriter, r *http.Request) (err error) {
	defer func() {
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	d, f := path.Split(r.URL.Path)
	switch d {
	case "/bot/":
		m := &BotPostMessage{}
		if err = json.Unmarshal(b, m); err != nil {
			return err
		}
		bot := h.g.Lookup(f)
		if len(m.Command) == 0 || bot == nil {
			return io.EOF
		}
		if err = bot.Write(m.TaskId, m.Message); err != nil {
			return err
		}
	case "/":
		if f == "tmpl" {
			tmpl, err := template.New("bot").Parse(string(b))
			if err != nil {
				return err
			}
			h.mu.Lock()
			h.tmpl = tmpl
			h.mu.Unlock()
		} else {
			p, err := UnmarshalProfile(b)
			if err != nil {
				return err
			}
			if err = h.g.Post(*p); err != nil {
				return err
			}
		}
	}
	io.WriteString(w, "OK")
	return nil
}

func (h *httpHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	d, f := path.Split(r.URL.Path)
	switch d {
	case "/bot/":
		if err := h.g.Delete(f); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	io.WriteString(w, "OK")
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("http: %+v", *r)
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodDelete:
		h.handleDelete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
