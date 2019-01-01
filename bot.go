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

const taskContextKeyTask = taskContextKey("task")

type Time time.Time

func (t *Time) Elapsed() time.Duration { return time.Since(t.T()) }
func (t *Time) T() time.Time           { return time.Time(*t) }

type Task struct {
	Name  string
	Start Time
}

type Bot struct {
	Profile
	Start Time
	chans map[string]struct{}

	ctx    context.Context
	cancel context.CancelFunc

	mc *TeeMsgConn
	wg sync.WaitGroup

	tasks    map[context.Context]time.Time
	welcomec chan struct{}
	netpfx   *irc.Prefix

	pm    *PatternMatcher
	pmraw *PatternMatcher

	mu sync.RWMutex
}

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
	pmraw, err := NewPatternMatcher(p.PatternsRaw)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.pm, b.pmraw = pm, pmraw
	b.Patterns, b.PatternsRaw = p.Patterns, p.PatternsRaw
	b.mu.Unlock()
	return nil
}

func NewBot(ctx context.Context, p Profile) (b *Bot, err error) {
	b = &Bot{Profile: p,
		chans:    make(map[string]struct{}),
		welcomec: make(chan struct{}),
		tasks:    make(map[context.Context]time.Time),
		Start:    Time(time.Now()),
	}
	if err = b.Update(b.Profile); err != nil {
		return nil, err
	}
	b.ctx, b.cancel = context.WithCancel(ctx)
	defer func() {
		if err != nil {
			b.Close()
		}
	}()
	if b.mc, err = NewTeeMsgConnDial(b.ctx, p.Server.Host); err != nil {
		return nil, err
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
		{Command: irc.USER, Params: []string{p.Nick, p.Nick, "localhost", p.Nick}},
		{Command: irc.NICK, Params: []string{p.Nick}},
	} {
		if err := b.mc.WriteMsg(msg); err != nil {
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

func (b *Bot) Env() []string {
	return []string{"SITBOT_ID=" + b.Id, "SITBOT_NICK=" + b.Nick}
}

func (b *Bot) PipeCmd(ctx context.Context, cmdtxt, tgt string, env []string) error {
	cctx, cancel := context.WithCancel(ctx)
	cmd, err := NewCmd(cctx, cmdtxt, env)
	if err != nil {
		cancel()
		return err
	}
	defer func() {
		cancel()
		cmd.Close()
	}()
	for l := range cmd.Lines() {
		out := irc.Message{Command: irc.PRIVMSG, Params: []string{tgt, l}}
		if err := b.mc.WriteMsg(out); err != nil {
			return err
		}
	}
	return err
}

func (b *Bot) processPrivMsg(ctx context.Context, msg irc.Message) error {
	sender, tgt, txt := msg.Prefix.Name, msg.Params[0], msg.Params[1]
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
	env := append(b.Env(), "SITBOT_FROM="+sender, "SITBOT_CHAN="+tgt)
	return b.PipeCmd(ctx, cmdtxt, outtgt, env)
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
		fallthrough
	case irc.PONG:
		return nil
	case irc.PING:
		tctx := context.WithValue(b.ctx, taskContextKeyTask, "ping")
		b.runTask(tctx, func(context.Context) error {
			return b.mc.WriteMsg(
				irc.Message{Command: irc.PONG, Params: msg.Params})
		})
		return nil
	case irc.PRIVMSG:
		if msg.Prefix != nil && len(msg.Params) > 0 {
			tctx := context.WithValue(b.ctx, taskContextKeyTask, msg.Params[1])
			b.runTask(tctx, func(ctx context.Context) error {
				return b.processPrivMsg(ctx, msg)
			})
		}
	case irc.JOIN:
		if len(msg.Params) > 0 {
			b.mu.Lock()
			b.chans[msg.Params[0]] = struct{}{}
			b.mu.Unlock()
		}
	}
	tctx := context.WithValue(b.ctx, taskContextKeyTask, strings.Join(msg.Params, " "))
	b.runTask(tctx, func(ctx context.Context) error {
		b.mu.RLock()
		pm := b.pmraw
		b.mu.RUnlock()
		cmdtxt := pm.Apply(msg.Command + " " + strings.Join(msg.Params, " "))
		if cmdtxt == "" {
			return nil
		}
		return b.PipeCmd(ctx, cmdtxt, b.Nick, b.Env())
	})
	return nil
}

func (b *Bot) TxMsgs() uint64 { return b.mc.TxMsgs() }
func (b *Bot) RxMsgs() uint64 { return b.mc.RxMsgs() }
