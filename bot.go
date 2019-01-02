package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/sorcix/irc.v2"
)

type Time time.Time

func (t Time) Elapsed() time.Duration { return time.Since(t.T()).Round(time.Second) }
func (t Time) T() time.Time           { return time.Time(t) }

type Task struct {
	Name    string
	Start   Time
	Command string
	lines   uint32
	b       *Bot
	ctx     context.Context
	cancel  context.CancelFunc
}

func (t *Task) Lines() uint32 { return atomic.LoadUint32(&t.lines) }

func (t *Task) Write(msg irc.Message) error {
	if err := t.b.mc.WriteMsg(msg); err != nil {
		return err
	}
	atomic.AddUint32(&t.lines, 1)
	return nil
}

func (t *Task) PipeCmd(cmdtxt, tgt string, env []string) error {
	cctx, cancel := context.WithCancel(t.ctx)
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
		if err := t.Write(out); err != nil {
			return err
		}
	}
	return err
}

type Bot struct {
	Profile
	Start    Time
	Channels map[string]struct{}

	ctx     context.Context
	cancel  context.CancelFunc
	limiter *rate.Limiter

	mc *TeeMsgConn
	wg sync.WaitGroup

	Tasks map[uint64]*Task
	tid   uint64

	welcomec chan struct{}
	netpfx   *irc.Prefix

	pm    *PatternMatcher
	pmraw *PatternMatcher

	mu sync.RWMutex
}

func (b *Bot) Ctx() context.Context { return b.ctx }

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
		Start:    Time(time.Now()),
		Channels: make(map[string]struct{}),
		Tasks:    make(map[uint64]*Task),
		welcomec: make(chan struct{}),
		limiter:  rate.NewLimiter(rate.Every(time.Second), 1),
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
		b.runTask("JOIN", "JOIN", nil, func(t *Task) error {
			return t.Write(irc.Message{Command: irc.JOIN, Params: []string{ch}})
		})
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

func (b *Bot) processPrivMsg(t *Task, msg irc.Message) error {
	sender, tgt := msg.Prefix.Name, msg.Params[0]
	outtgt := tgt
	if tgt[0] != '#' {
		outtgt = sender
	}
	cmdtxt := strings.Replace(t.Command, "%s", sender, -1)
	env := append(b.Env(), "SITBOT_FROM="+sender, "SITBOT_CHAN="+tgt)
	return t.PipeCmd(cmdtxt, outtgt, env)
}

func (b *Bot) runTask(name, cmdtxt string, pm **PatternMatcher, f func(*Task) error) {
	cctx, cancel := context.WithCancel(b.ctx)
	task := &Task{
		Name: name, Start: Time(time.Now()), Command: cmdtxt,
		b: b, ctx: cctx, cancel: cancel}
	b.mu.Lock()
	b.tid++
	tid := b.tid
	b.Tasks[tid] = task
	b.mu.Unlock()
	b.wg.Add(1)
	go func() {
		defer func() {
			b.mu.Lock()
			delete(b.Tasks, tid)
			b.mu.Unlock()
			b.wg.Done()
		}()
		if pm != nil {
			task.Command = b.tryPatternMatch(task.Command, pm)
		}
		if task.Command != "" && b.limiter.Wait(cctx) == nil {
			f(task)
		}
	}()
}

func (b *Bot) tryPatternMatch(txt string, pm **PatternMatcher) string {
	b.mu.RLock()
	p := *pm
	b.mu.RUnlock()
	return p.Apply(txt)
}

func (b *Bot) processMsg(msg irc.Message) error {
	log.Printf("processMsg: %+v\n", msg)
	switch msg.Command {
	case irc.RPL_WELCOME:
		b.netpfx = msg.Prefix
		close(b.welcomec)
	case irc.PONG:
		return nil
	case irc.PING:
		b.runTask("ping", "PING", nil, func(t *Task) error {
			return t.Write(irc.Message{Command: irc.PONG, Params: msg.Params})
		})
		return nil
	case irc.PRIVMSG:
		if msg.Prefix != nil && len(msg.Params) > 0 {
			tf := func(t *Task) error { return b.processPrivMsg(t, msg) }
			b.runTask(msg.Params[1], msg.Params[1], &b.pm, tf)
		}
	case irc.JOIN:
		if len(msg.Params) > 0 {
			b.mu.Lock()
			b.Channels[msg.Params[0]] = struct{}{}
			b.mu.Unlock()
		}
	}
	msgcmd := msg.Command + " " + strings.Join(msg.Params, " ")
	b.runTask(msgcmd, msgcmd, &b.pmraw, func(t *Task) error {
		return t.PipeCmd(t.Command, b.Nick, b.Env())
	})
	return nil
}

func (b *Bot) TxMsgs() uint64 { return b.mc.TxMsgs() }
func (b *Bot) RxMsgs() uint64 { return b.mc.RxMsgs() }
