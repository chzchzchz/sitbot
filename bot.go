package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"gopkg.in/sorcix/irc.v2"
)

type taskContextKey string
type Time time.Time

const taskContextKeyTask = taskContextKey("task")

type Bot struct {
	Profile
	StartTime Time
	chans     map[string]struct{}

	ctx    context.Context
	mc     *TeeMsgConn
	cancel context.CancelFunc
	wg     sync.WaitGroup

	tasks    map[context.Context]time.Time
	welcomec chan struct{}
	netpfx   *irc.Prefix

	pm *PatternMatcher

	mu sync.RWMutex
}

type Task struct {
	Name  string
	Start Time
}

func (t *Time) Elapsed() time.Duration { return time.Since(t.T()) }
func (t *Time) T() time.Time           { return time.Time(*t) }

func (b *Bot) Tasks() (ret []Task) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for c, t := range b.tasks {
		ret = append(ret, Task{c.Value(taskContextKeyTask).(string), Time(t)})
	}
	return ret
}

func (b *Bot) Channels() (ret []string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for c := range b.chans {
		ret = append(ret, c)
	}
	return ret
}

func (b *Bot) Update(p Profile) error {
	pm, err := NewPatternMatcher(p.Patterns)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.pm = pm
	b.Patterns = p.Patterns
	b.mu.Unlock()
	return nil
}

func NewBot(ctx context.Context, p Profile) (*Bot, error) {
	pm, err := NewPatternMatcher(p.Patterns)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithCancel(ctx)
	mc, err := NewTeeMsgConnDial(cctx, p.Server.Host)
	if err != nil {
		cancel()
		return nil, err
	}
	b := &Bot{Profile: p, ctx: ctx, mc: mc, cancel: cancel,
		chans:     make(map[string]struct{}),
		welcomec:  make(chan struct{}),
		tasks:     make(map[context.Context]time.Time),
		pm:        pm,
		StartTime: Time(time.Now()),
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		rc, dc := b.mc.NewReadChan()
		defer close(dc)
		for msg := range rc {
			if err := b.processMsg(msg); err != nil {
				panic(err)
			}
		}
	}()
	for _, msg := range []irc.Message{
		{Command: irc.USER,
			Params: []string{p.Nick, p.Nick, "localhost", "realname"},
		},
		{Command: irc.NICK, Params: []string{p.Nick}},
	} {
		if err := mc.WriteMsg(msg); err != nil {
			return nil, err
		}
	}
	select {
	case <-b.welcomec:
	case <-b.mc.ctx.Done():
		return nil, ctx.Err()
	}
	for _, ch := range p.Chans {
		b.wg.Add(1)
		go func(chn string) {
			defer b.wg.Done()
			b.mc.WriteMsg(irc.Message{Command: irc.JOIN, Params: []string{chn}})
		}(ch)
	}
	return b, nil
}

func (b *Bot) Close() {
	b.cancel()
	b.wg.Wait()
}

func (b *Bot) processPrivMsg(ctx context.Context, sender string, tgt string, txt string) error {
	outtgt := tgt
	if tgt[0] != '#' {
		outtgt = sender
	}
	b.mu.RLock()
	pm := b.pm
	b.mu.RUnlock()
	cmdtxt := pm.Apply(txt)
	if cmdtxt == "" {
		return nil
	}
	cmdtxt = strings.Replace(cmdtxt, "%s", sender, -1)

	cctx, cancel := context.WithCancel(ctx)
	env := []string{
		"SITBOT_ID=" + b.Id,
		"SITBOT_NICK=" + b.Nick,
		"SITBOT_FROM=" + sender,
		"SITBOT_CHAN=" + tgt,
	}
	cmd, err := NewCmd(cctx, cmdtxt, env)
	if err != nil {
		cancel()
		log.Println("cmd failed", err)
		return err
	}
	defer func() {
		cancel()
		cmd.Close()
	}()
	for l := range cmd.Lines() {
		out := irc.Message{Command: irc.PRIVMSG, Params: []string{outtgt, l}}
		if err := b.mc.WriteMsg(out); err != nil {
			return err
		}
	}
	return err
}

func (b *Bot) runTask(ctx context.Context, f func(context.Context) error) {
	t := time.Now()
	b.mu.Lock()
	b.tasks[ctx] = t
	b.mu.Unlock()
	b.wg.Add(1)
	go func() {
		defer func() {
			b.mu.Lock()
			delete(b.tasks, ctx)
			b.mu.Unlock()
			b.wg.Done()
		}()
		f(ctx)
	}()
}

func (b *Bot) processMsg(msg irc.Message) error {
	log.Printf("processMsg: %+v\n", msg)
	switch msg.Command {
	case irc.RPL_WELCOME:
		b.netpfx = msg.Prefix
		close(b.welcomec)
	case irc.PING:
		tctx := context.WithValue(b.ctx, taskContextKeyTask, "ping")
		b.runTask(tctx, func(context.Context) error {
			return b.mc.WriteMsg(
				irc.Message{Command: irc.PONG, Params: msg.Params})
		})
	case irc.PRIVMSG:
		if msg.Prefix == nil || len(msg.Params) < 1 {
			return nil
		}
		tctx := context.WithValue(b.ctx, taskContextKeyTask, msg.Params[1])
		b.runTask(tctx, func(ctx context.Context) error {
			return b.processPrivMsg(
				ctx,
				msg.Prefix.Name,
				msg.Params[0],
				msg.Params[1])
		})
	case irc.JOIN:
		if len(msg.Params) == 0 {
			return nil
		}
		b.mu.Lock()
		b.chans[msg.Params[0]] = struct{}{}
		b.mu.Unlock()
	case irc.INVITE:
		if len(msg.Params) > 1 {
			if cn := msg.Params[1]; len(cn) > 0 && cn[0] == '#' {
				tctx := context.WithValue(b.ctx, taskContextKeyTask, "invite")
				b.runTask(tctx, func(context.Context) error {
					out := irc.Message{Command: irc.JOIN, Params: []string{cn}}
					return b.mc.WriteMsg(out)
				})
			}
		}
	}
	return nil
}

func (b *Bot) TxMsgs() uint64 { return b.mc.TxMsgs() }
func (b *Bot) RxMsgs() uint64 { return b.mc.RxMsgs() }
