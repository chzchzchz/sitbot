package main

import (
	"encoding/json"
	"fmt"
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

func ServeHttp(g *Gang, serv string) error {
	h := &httpHandler{
		g:    g,
		tmpl: template.Must(template.New("bot").Parse("")),
	}
	return http.ListenAndServe(serv, h)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	tmpl := h.tmpl
	h.mu.RUnlock()
	io.WriteString(w, "<html><title>bot report</title><body><h1>Active Bot Report</h1>")
	for _, b := range h.g.bots {
		if err := tmpl.Execute(w, b); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, fmt.Sprintf("Total bots: %d</body></html>", len(h.g.bots)))
}

func (h *httpHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, err.Error())
		}
	}()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	d, f := path.Split(r.URL.Path)
	switch d {
	case "/bouncer/":
		bot := h.g.Lookup(f)
		if bot == nil {
			err = io.EOF
			return
		}
		if _, err = NewBouncer(bot, string(b)); err != nil {
			return
		}
	case "/bot/":
		m := &irc.Message{}
		if err = json.Unmarshal(b, m); err != nil {
			return
		}
		bot := h.g.Lookup(f)
		if len(m.Command) == 0 || bot == nil {
			err = io.EOF
			return
		}
		if err = bot.mc.WriteMsg(*m); err != nil {
			return
		}
	case "/":
		if f == "tmpl" {
			tmpl, err := template.New("bot").Parse(string(b))
			if err != nil {
				return
			}
			h.mu.Lock()
			h.tmpl = tmpl
			h.mu.Unlock()
		} else {
			p, err := UnmarshalProfile(b)
			if err != nil {
				return
			}
			if err = h.g.Post(*p); err != nil {
				return
			}
		}
	}
	io.WriteString(w, "OK")
}

func (h *httpHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	d, f := path.Split(r.URL.Path)
	switch d {
	case "/bot/":
		if err := h.g.Delete(f); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case "/bouncer/":
		// TODO
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusBadRequest)
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
