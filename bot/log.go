package bot

import (
	"log"

	"gopkg.in/sorcix/irc.v2"
)

type Log struct {
	Stage
}

func (l *Log) Process(msg irc.Message) error {
	log.Printf("%+v", msg)
	return l.Stage.Process(msg)
}
