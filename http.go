package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"

	"gopkg.in/sorcix/irc.v2"
)

type httpHandler struct {
	g *Gang
}

func ServeHttp(g *Gang, serv string) error {
	h := &httpHandler{g: g}
	return http.ListenAndServe(serv, h)
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
		io.WriteString(w, string(b))
		return
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
		p, err := UnmarshalProfile(b)
		if err != nil {
			return
		}
		if err = h.g.Post(*p); err != nil {
			return
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
		io.WriteString(w, "OK")
	case "/bouncer/":
		// TODO
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("http: %+v", *r)
	switch r.Method {
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodDelete:
		h.handleDelete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
