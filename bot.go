package main

import (
	"context"
	"log"
	"strings"
	"sync"

	"gopkg.in/sorcix/irc.v2"
)

type Bot struct {
	Profile
	ctx    context.Context
	mc     *TeeMsgConn
	cancel context.CancelFunc
	wg     sync.WaitGroup

	welcomec chan struct{}
	netpfx   *irc.Prefix

	pm *PatternMatcher

	chans map[string]*chanInfo

	mu sync.RWMutex
}

type chanInfo struct {
	nicks  map[string]struct{}
	filled bool
}

type Channel struct {
	Name  string
	Nicks []string
}

func (b *Bot) Channels() (ret []Channel) {
	for n, ci := range b.chans {
		nicks := make([]string, 0, len(ci.nicks))
		for nick := range ci.nicks {
			nicks = append(nicks, nick)
		}
		ret = append(ret, Channel{Name: n, Nicks: nicks})
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
		chans:    make(map[string]*chanInfo),
		welcomec: make(chan struct{}),
		pm:       pm,
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
		{Command: irc.CAP, Params: []string{"LS"}},
	} {
		if err := mc.WriteMsg(msg); err != nil {
			return nil, err
		}
	}
	select {
	case <-b.welcomec:
	case <-b.mc.DoneChan():
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

func (b *Bot) Done() <-chan struct{} { return b.mc.DoneChan() }

func (b *Bot) Close() {
	b.cancel()
	b.wg.Wait()
}

func (b *Bot) lookupChanInfo(chname string) *chanInfo {
	if ci := b.chans[chname]; ci != nil {
		return ci
	}
	ci := &chanInfo{nicks: make(map[string]struct{})}
	b.chans[chname] = ci
	return ci
}

func (b *Bot) processPrivMsg(sender string, tgt string, txt string) error {
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

	cctx, cancel := context.WithCancel(b.ctx)
	cmd, err := NewCmd(cctx, cmdtxt)
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

func (b *Bot) processMsg(msg irc.Message) error {
	log.Printf("processMsg: %+v\n", msg)
	switch msg.Command {
	case irc.RPL_WELCOME:
		b.netpfx = msg.Prefix
		close(b.welcomec)
	case irc.PING:
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.mc.WriteMsg(irc.Message{Command: irc.PONG, Params: msg.Params})
		}()
	case irc.PRIVMSG:
		if msg.Prefix == nil || len(msg.Params) < 1 {
			return nil
		}
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.processPrivMsg(msg.Prefix.Name, msg.Params[0], msg.Params[1])
		}()
	case irc.RPL_NAMREPLY:
		if len(msg.Params) < 4 {
			return nil
		}
		ci := b.lookupChanInfo(msg.Params[2])
		for _, n := range strings.Split(msg.Params[3], " ") {
			ci.nicks[n] = struct{}{}
		}
	case irc.RPL_ENDOFNAMES:
		if len(msg.Params) < 2 {
			return nil
		}
		chname := msg.Params[1]
		b.lookupChanInfo(chname).filled = true
	case irc.PART:
		if msg.Prefix == nil {
			return nil
		}
		if msg.Prefix.Name == b.Nick {
			delete(b.chans, msg.Params[0])
		} else {
			delete(b.lookupChanInfo(msg.Params[0]).nicks, msg.Prefix.Name)
		}
	case irc.KICK:
		if len(msg.Params) >= 2 && msg.Params[1] == b.Nick {
			delete(b.chans, msg.Params[0])
		}
	case irc.INVITE:
		if len(msg.Params) > 1 {
			if cn := msg.Params[1]; len(cn) > 0 && cn[0] == '#' {
				out := irc.Message{Command: irc.JOIN, Params: []string{cn}}
				return b.mc.WriteMsg(out)
			}
		}
	case irc.JOIN:
		if msg.Prefix == nil || msg.Prefix.Name == b.Nick || len(msg.Params) == 0 {
			return nil
		}
		chname := msg.Params[0]
		b.lookupChanInfo(chname).nicks[msg.Prefix.Name] = struct{}{}
	}
	return nil
}
