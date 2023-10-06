package bot

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/sorcix/irc.v2"
)

type Time time.Time

func (t Time) Elapsed() time.Duration { return time.Since(t.T()).Round(time.Second) }
func (t Time) T() time.Time           { return time.Time(t) }

type Stage interface {
	Process(msg irc.Message) error
}

type Bot struct {
	Profile
	Start Time

	dispatcher *Dispatcher
	State      *State
	Login      *Login

	ctx    context.Context
	cancel context.CancelFunc

	Tasks *Tasks

	mc *TeeMsgConn
	wg sync.WaitGroup

	mu sync.RWMutex
}

func (b *Bot) Ctx() context.Context { return b.ctx }

func (b *Bot) Update(p Profile) error {
	if err := b.dispatcher.Update(p.Patterns, p.PatternsRaw); err != nil {
		return err
	}
	b.mu.Lock()
	b.Profile = p
	b.mu.Unlock()
	return nil
}

func NewBot(ctx context.Context, p Profile) (_ *Bot, err error) {
	cctx, cancel := context.WithCancel(ctx)
	b := &Bot{Profile: p,
		Start:  Time(time.Now()),
		State:  NewState(),
		ctx:    cctx,
		cancel: cancel,
	}
	defer func() {
		if err != nil {
			cancel()
			b.Close()
		}
	}()
	conn, cerr := p.Dial(cctx)
	if cerr != nil {
		return nil, cerr
	}
	if b.mc, err = NewTeeMsgConn(cctx, conn, p.RateMs); err != nil {
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Every(time.Duration(p.RateMs)*time.Millisecond), 1)
	b.Tasks = NewTasks(cctx, limiter, b.mc.MsgConn)

	// Build pipeline.
	b.dispatcher = NewDispatcher(&b.Profile, b.Tasks)
	if err = b.Update(b.Profile); err != nil {
		return nil, err
	}
	b.Login = NewLogin(&b.Profile.ProfileLogin, b.Tasks)
	b.AddStage(b.Login)
	if b.Verbosity > 0 {
		b.AddStage(&Log{b.dispatcher})
	} else {
		b.AddStage(b.dispatcher)
	}
	b.AddStage(b.State)
	// Login.
	if err := b.Login.Run(); err != nil {
		return nil, err
	}
	select {
	case <-b.Login.Welcome():
	case <-b.mc.ctx.Done():
		return nil, b.mc.ctx.Err()
	}
	// Join channels.
	for _, ch := range p.Chans {
		b.Tasks.Run("JOIN", "JOIN", func(t *Task) error {
			return t.Write(irc.Message{Command: irc.JOIN, Params: []string{ch}})
		})
	}
	return b, nil
}

func (b *Bot) AddStage(s Stage) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		rc, dc := b.mc.NewReadChan()
		defer close(dc)
		for msg := range rc {
			if err := s.Process(msg); err != nil {
				panic(err)
			}
		}
	}()
}

func (b *Bot) Close() {
	if b.Tasks != nil {
		b.Tasks.Close()
	}
	b.cancel()
	b.wg.Wait()
}

func (b *Bot) Write(tid TaskId, msg irc.Message) error {
	if tid == 0 {
		return b.mc.WriteMsg(msg)
	}
	return b.Tasks.Write(tid, msg)
}

func (b *Bot) TxMsgs() uint64 { return b.mc.TxMsgs() }
func (b *Bot) RxMsgs() uint64 { return b.mc.RxMsgs() }

func (b *Bot) RLock()   { b.mu.RLock() }
func (b *Bot) RUnlock() { b.mu.RUnlock() }

func (b *Bot) TeeMsg() *TeeMsgConn { return b.mc }
