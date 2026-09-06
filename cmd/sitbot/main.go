package main

import (
	"flag"
	"log/slog"
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
	u, p, ok := r.BasicAuth()
	if !ok || u != h.user || p != h.pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		slog.Warn("http bad auth", "addr", r.RemoteAddr, "user", u, "pass", p)
		slog.Debug("http auth attempt", "addr", r.RemoteAddr, "provided_user", u, "ok", ok)
		return
	}
	slog.Debug("http auth ok", "addr", r.RemoteAddr, "user", u)
	h.h.ServeHTTP(w, r)
}

type corsHandler struct {
	h http.Handler
}

func (h *corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == http.MethodOptions {
		return
	}
	h.h.ServeHTTP(w, r)
}

func main() {
	laddrFlag := flag.String("l", "localhost:9991", "listen address")
	userFlag := flag.String("u", "", "username for basic http authentication")
	passFlag := flag.String("p", "", "password for basic http authentication")
	corsFlag := flag.Bool("cors", false, "enable CORS")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *debugFlag {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

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
		slog.Info("using basic authentication on user", "user", *userFlag)
		h = &authHttpHandler{h: h, user: *userFlag, pass: *passFlag}
	}
	if *corsFlag {
		slog.Info("enabling CORS")
		h = &corsHandler{h: h}
	}

	slog.Info("serving bot on", "laddr", laddr)
	http.ListenAndServe(laddr, h)
}
