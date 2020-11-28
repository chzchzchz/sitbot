package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/chzchzchz/sitbot/bot"
	bothttp "github.com/chzchzchz/sitbot/bot/http"
	"github.com/chzchzchz/sitbot/bouncer"
)

type authHttpHandler struct {
	h    http.Handler
	user string
	pass string
}

func (h *authHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(ip).IsLoopback() {
		// Pass-through for any local accesses so script callbacks work.
		h.h.ServeHTTP(w, r)
		return
	}
	if u, p, ok := r.BasicAuth(); !ok || u != h.user || p != h.pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("http: [%s] bad auth %q %q", r.RemoteAddr, u, p)
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
	if os.Getenv("SITBOT_URL") == "" {
		// May set an alternative SITBOT_URL if listen address is not the
		// the same as access address (e.g., 0.0.0.0 vs 127.0.0.1).
		os.Setenv("SITBOT_URL", "http://"+laddr)
	}

	mux := http.NewServeMux()

	g := bot.NewGang()
	mux.Handle("/", bothttp.NewGangHandler(g))
	mux.Handle("/bouncer/", http.StripPrefix("/bouncer", bouncer.NewHandler(g)))
	var h http.Handler
	h = mux
	if len(*userFlag) > 0 {
		log.Println("using basic authentication on user " + *userFlag)
		h = &authHttpHandler{h: h, user: *userFlag, pass: *passFlag}
	}

	log.Println("serving bot on", laddr)
	http.ListenAndServe(laddr, h)
}
