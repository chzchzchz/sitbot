package http

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/chzchzchz/sitbot/bot"
)

type templateHandler struct {
	g    *bot.Gang
	tmpl *template.Template
	mu   sync.RWMutex
}

func newTemplateHandler(g *bot.Gang) *templateHandler {
	return &templateHandler{
		g:    g,
		tmpl: template.Must(template.New("bot").Parse("")),
	}
}

func (h *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.mu.RLock()
		tmpl := h.tmpl
		h.mu.RUnlock()
		h.g.LockBots()
		defer h.g.UnlockBots()
		if err := tmpl.Execute(w, h.g); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		postWrap(w, r, func(b []byte) error { return h.post(w, r, b) })
	default:
		http.Error(w, "bad request", http.StatusMethodNotAllowed)
	}
}

func (h *templateHandler) post(w http.ResponseWriter, r *http.Request, b []byte) error {
	f := r.URL.Path
	if f == "tmpl" {
		tmpl, err := template.New("bot").Parse(string(b))
		if err != nil {
			return err
		}
		h.mu.Lock()
		h.tmpl = tmpl
		h.mu.Unlock()
	} else {
		p, err := bot.UnmarshalProfile(b)
		if err != nil {
			return err
		}
		if err = h.g.Post(*p); err != nil {
			return err
		}
	}
	return ok(w)
}
