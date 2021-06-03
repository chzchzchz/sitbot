package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/chzchzchz/sitbot/bot"
	"gopkg.in/sorcix/irc.v2"
)

var botState *bot.State
var client *http.Client = &http.Client{}

func MustBotState() *bot.State {
	if botState != nil {
		return botState
	}
	r, err := client.Get(botURL())
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()
	b := &bot.Bot{}
	if err := json.NewDecoder(r.Body).Decode(b); err != nil {
		panic(err)
	}
	botState = b.State
	return botState
}

func NopNicks(channame string) (ret []string) {
	bs := MustBotState()
	r := bs.Channels[channame]
	if r == nil {
		return ret
	}
	for uname, ru := range r.Users {
		if !strings.Contains(ru.Mode, "@") {
			ret = append(ret, uname)
		}
	}
	sort.Strings(ret)
	return ret
}

func mustPostCommand(msg *irc.Message) {
	if err := postCommand(msg); err != nil {
		panic(err)
	}
}

func botURL() string {
	return os.Getenv("SITBOT_URL") + "/bot/" + os.Getenv("SITBOT_ID")
}

func postCommand(msg *irc.Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	url := botURL()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}
