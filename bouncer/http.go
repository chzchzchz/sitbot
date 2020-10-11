package bouncer

import (
	"io"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/chzchzchz/sitbot/bot"
)

type httpHandler struct {
	g *bot.Gang
}

func NewHandler(g *bot.Gang) http.Handler {
	return &httpHandler{g: g}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var err error
		defer func() {
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
		}()
		err = func() error {
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
			_, f := path.Split(r.URL.Path)
			bot := h.g.Lookup(f)
			if bot == nil {
				return io.EOF
			}
			_, err = NewBouncer(bot, string(b))
			return err
		}()
	default:
		http.Error(w, "Not allowed", http.StatusMethodNotAllowed)
	}
}
