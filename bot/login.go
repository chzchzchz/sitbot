package bot

import (
	"gopkg.in/sorcix/irc.v2"
)

type Login struct {
	*ProfileLogin
	Netpfx   *irc.Prefix
	tasks    *Tasks
	welcomec chan struct{}
}

func NewLogin(p *ProfileLogin, t *Tasks) *Login {
	return &Login{ProfileLogin: p, tasks: t, welcomec: make(chan struct{})}
}

func (l *Login) Welcome() <-chan struct{} { return l.welcomec }

func (l *Login) Run() error {
	n := 0
	if l.Pass == "" {
		n++
	}
	if l.User == "" {
		l.User = l.Nick
	}
	msgs := []irc.Message{
		{Command: irc.PASS, Params: []string{l.Pass}},
		{Command: irc.NICK, Params: []string{l.Nick}},
		{Command: irc.USER, Params: []string{l.User, l.Nick, "localhost", l.Nick}},
	}
	for _, msg := range msgs[n:] {
		if err := l.tasks.mc.WriteMsg(msg); err != nil {
			return err
		}
	}
	return nil
}

func (l *Login) Process(msg irc.Message) error {
	switch msg.Command {
	case irc.RPL_WELCOME:
		oldPfx := l.Netpfx
		l.Netpfx = msg.Prefix
		if oldPfx == nil {
			close(l.welcomec)
		}
	case irc.PING:
		l.tasks.Run("ping", "PING", func(t *Task) error {
			return t.Write(irc.Message{Command: irc.PONG, Params: msg.Params})
		})
	}
	return nil
}
