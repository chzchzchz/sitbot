package http

import (
	"net/http"

	"github.com/chzchzchz/sitbot/bot"
)

func NewGangHandler(g *bot.Gang) http.Handler {
	mux := http.NewServeMux()
	th := newTemplateHandler(g)
	mux.Handle("/", http.StripPrefix("/", th))

	bh := &botHandler{g: g}
	mux.Handle("/bot/", http.StripPrefix("/bot", bh))

	return mux
}
