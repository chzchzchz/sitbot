package bot

import (
	"log/slog"

	"gopkg.in/sorcix/irc.v2"
)

type Log struct {
	Stage
}

func (l *Log) Process(msg irc.Message) error {
	slog.Info("irc message", "msg", msg)
	return l.Stage.Process(msg)
}
