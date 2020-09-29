package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/chzchzchz/sitbot/bot"
)

type authHttpHandler struct {
	h    http.Handler
	user string
	pass string
}

func (h *authHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u, p, ok := r.BasicAuth(); !ok || u != h.user || p != h.pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("http: bad auth %q %q %v", u, p, ok)
		return
	}
	h.h.ServeHTTP(w, r)
}

func main() {
	laddrFlag := flag.String("l", "localhost:9991", "listen address")
	userFlag := flag.String("u", "", "username for basic http authentication")
	passFlag := flag.String("p", "", "password for basic http authentication")
	flag.Parse()

	laddr := *laddrFlag
	os.Setenv("SITBOT_URL", "http://"+laddr)

	h := bot.GangHandler(bot.NewGang())
	if len(*userFlag) > 0 {
		log.Println("using basic authentication on user " + *userFlag)
		h = &authHttpHandler{h: h, user: *userFlag, pass: *passFlag}
	}

	log.Println("serving bot on", laddr)
	http.ListenAndServe(laddr, h)
}
