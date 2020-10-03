package bot

import (
	"context"
	"fmt"
	"io"
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

type TaskId uint64
type Task struct {
	Name    string
	Start   Time
	Command string
	tid     TaskId
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
	tidenv := fmt.Sprintf("SITBOT_TID=%d", t.tid)
	toks := strings.Split(cmdtxt, " ")
	cmdname, cmdargs := strings.Replace(toks[0], "/", "_", -1), ""
	if len(toks) > 1 {
		cmdargs = strings.Join(toks[1:], " ")
	}
	env = append(env, tidenv)
	cmd, err := NewCmd(cctx, "scripts/sandbox", []string{cmdname, cmdargs}, env)
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

	Tasks map[TaskId]*Task
	tid   TaskId

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

func NewBot(ctx context.Context, p Profile) (_ *Bot, err error) {
	invl := time.Duration(p.RateMs) * time.Millisecond
	b := &Bot{Profile: p,
		Start:    Time(time.Now()),
		Channels: make(map[string]struct{}),
		Tasks:    make(map[TaskId]*Task),
		welcomec: make(chan struct{}),
		limiter:  rate.NewLimiter(rate.Every(invl), 1),
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
	conn, cerr := p.Dial(b.ctx)
	if cerr != nil {
		return nil, cerr
	}
	if b.mc, err = NewTeeMsgConn(b.ctx, conn, p.RateMs); err != nil {
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

	n := 0
	if p.Pass == "" {
		n++
	}
	if p.User == "" {
		p.User = p.Nick
	}
	msgs := []irc.Message{
		{Command: irc.PASS, Params: []string{p.Pass}},
		{Command: irc.NICK, Params: []string{p.Nick}},
		{Command: irc.USER, Params: []string{p.User, p.Nick, "localhost", p.Nick}},
	}
	for _, msg := range msgs[n:] {
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

func (b *Bot) Write(tid TaskId, msg irc.Message) error {
	if tid == 0 {
		return b.mc.WriteMsg(msg)
	}
	b.mu.RLock()
	t, ok := b.Tasks[tid]
	b.mu.RUnlock()
	if !ok {
		return io.EOF
	}
	return t.Write(msg)
}

// runTask puts a command in the task list and schedules it to run.
func (b *Bot) runTask(name, cmdtxt string, pm **PatternMatcher, f func(*Task) error) {
	cctx, cancel := context.WithCancel(b.ctx)
	task := &Task{
		Name: name, Start: Time(time.Now()), Command: cmdtxt,
		b: b, ctx: cctx, cancel: cancel}
	b.mu.Lock()
	b.tid++
	tid := b.tid
	b.Tasks[tid] = task
	task.tid = tid
	b.mu.Unlock()
	b.wg.Add(1)
	go func() {
		defer func() {
			b.mu.Lock()
			delete(b.Tasks, task.tid)
			b.mu.Unlock()
			b.wg.Done()
		}()
		if pm != nil {
			cmdtxt := task.Command
			task.Command = b.tryPatternMatch(task.Command, pm)
			if task.Command != "" {
				log.Printf("[task] %q matched to %q", cmdtxt, task.Command)
			}
		}
		if task.Command != "" && b.limiter.Wait(task.ctx) == nil {
			if err := f(task); err != nil {
				log.Printf("[task] failed on command %q (%v)", task.Command, err)
			}
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
	if msg.Prefix != nil {
		msgcmd = msg.Prefix.String() + " " + msgcmd
	}
	b.runTask(msgcmd, msgcmd, &b.pmraw, func(t *Task) error {
		return t.PipeCmd(t.Command, b.Nick, b.Env())
	})
	return nil
}

func (b *Bot) TxMsgs() uint64 { return b.mc.TxMsgs() }
func (b *Bot) RxMsgs() uint64 { return b.mc.RxMsgs() }
