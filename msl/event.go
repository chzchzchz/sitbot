package msl

import (
	"fmt"
	"strings"
)

type Event struct {
	Type     string
	Pattern  string
	Location string
	Level    string
	Line     int

	Command Command
}

func (ev *Event) Name() string {
	return fmt.Sprintf("ev_%s_%dl", ev.Type, ev.Line)
}

func (ev *Event) Flags() string {
	ret := "--level " + ev.Level + " --location " + ev.Location
	if ev.Type == "join" {
		ret += " --nick $user --chan $chan"
	}
	return ret
}

func (ev *Event) Match() string {
	if ev.Type == "join" {
		return "^(?P<user>[^\\s@!]+)[^\\s]*\\s+JOIN\\s+(?P<chan>#[^\\s]+)"
	}
	pat := ev.Pattern
	pat = strings.Replace(pat, "*", ".*", -1)
	pat = strings.Replace(pat, " &", "[\\s]+[^\\s]+", -1)
	if len(pat) > 0 && pat[0] == '/' {
		i := len(ev.Pattern)
		if strings.HasSuffix(pat, "/i") {
			return "(?i)(" + pat[1:i-2] + ")"
		}
		return pat[1 : i-1]
	}
	return "^(?i)(" + pat + ")$"
}
