package bot

import (
	"log"
	"strings"
	"sync"

	"gopkg.in/sorcix/irc.v2"
)

type Dispatcher struct {
	*Tasks
	*Profile
	pm    *PatternMatcher
	pmraw *PatternMatcher
	mu    sync.RWMutex
}

func NewDispatcher(p *Profile, t *Tasks) *Dispatcher {
	return &Dispatcher{Tasks: t, Profile: p}
}

func (d *Dispatcher) Env() []string {
	return []string{"SITBOT_ID=" + d.Id, "SITBOT_NICK=" + d.Nick}
}

func (d *Dispatcher) Update(pats, rawpats []Pattern) error {
	pm, err := NewPatternMatcher(pats)
	if err != nil {
		return err
	}
	pmraw, err := NewPatternMatcher(rawpats)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.pm, d.pmraw = pm, pmraw
	d.mu.Unlock()
	return nil
}

func (d *Dispatcher) processPrivMsg(t *Task, msg irc.Message) error {
	sender, tgt := msg.Prefix.Name, msg.Params[0]
	outtgt := tgt
	if tgt[0] != '#' {
		outtgt = sender
	}
	cmdtxt := strings.Replace(t.Command, "%s", sender, -1)
	env := append(d.Env(),
		"SITBOT_FROM="+sender,
		"SITBOT_CHAN="+tgt,
		"SITBOT_MSG="+msg.Params[1])
	return t.PipeCmd(cmdtxt, outtgt, env)
}

func (d *Dispatcher) run(name, cmdtxt string, pm **PatternMatcher, f TaskFunc) {
	d.mu.RLock()
	p := *pm
	d.mu.RUnlock()
	if p == nil {
		return
	}
	taskCmd := p.Apply(cmdtxt)
	if taskCmd == "" {
		return
	}
	log.Printf("[task] %q matched to %q", cmdtxt, taskCmd)
	d.Tasks.Run(name, taskCmd, f)
}

func (d *Dispatcher) Process(msg irc.Message) error {
	if msg.Command == irc.PRIVMSG {
		if msg.Prefix != nil && len(msg.Params) > 0 {
			tf := func(t *Task) error { return d.processPrivMsg(t, msg) }
			d.run(msg.Params[1], msg.Params[1], &d.pm, tf)
		}
	}
	msgcmd := msg.Command + " " + strings.Join(msg.Params, " ")
	if msg.Prefix != nil {
		msgcmd = msg.Prefix.String() + " " + msgcmd
	}
	d.run(msgcmd, msgcmd, &d.pmraw, func(t *Task) error {
		return t.PipeCmd(t.Command, d.Nick, d.Env())
	})
	return nil
}
