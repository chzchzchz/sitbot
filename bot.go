package main

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/sorcix/irc"
)

type Bot struct {
	Profile
	ctx      context.Context
	mc       *MsgConn
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	welcomec chan struct{}

	re   []*regexp.Regexp
	tmpl [][]byte

	chans map[string]*chanInfo
}

type chanInfo struct {
	nicks  map[string]struct{}
	filled bool
}

func NewBot(ctx context.Context, p Profile) (*Bot, error) {
	re := make([]*regexp.Regexp, len(p.Patterns))
	tmpl := make([][]byte, len(p.Patterns))
	for i, pat := range p.Patterns {
		mat := strings.Replace(pat.Match, "%n", p.Nick, -1)
		r, err := regexp.Compile(mat)
		if err != nil {
			return nil, err
		}
		re[i] = r
		tmpl[i] = []byte(p.Patterns[i].Template)
	}
	cctx, cancel := context.WithCancel(ctx)
	mc, err := NewMsgConn(cctx, p.Server.Host)
	if err != nil {
		cancel()
		return nil, err
	}
	b := &Bot{Profile: p, ctx: ctx, mc: mc, cancel: cancel,
		chans:    make(map[string]*chanInfo),
		welcomec: make(chan struct{}),
		re:       re,
		tmpl:     tmpl,
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for msg := range b.mc.ReadChan() {
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

func (b *Bot) txt2cmd(txt string) (cmdtxt string) {
	if len(txt) == 0 {
		return ""
	}
	txtb := []byte(txt)
	for i, re := range b.re {
		if si := re.FindAllSubmatchIndex(txtb, 1); len(si) != 0 {
			res := []byte{}
			for _, submatches := range si {
				res = re.Expand(res, b.tmpl[i], txtb, submatches)
			}
			return string(res)
		}
	}
	return ""
}

func (b *Bot) processPrivMsg(sender string, tgt string, txt string) error {
	outtgt := tgt
	if tgt[0] != '#' {
		outtgt = sender
	}
	cmdtxt := b.txt2cmd(txt)
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
			b.processPrivMsg(msg.Prefix.Name, msg.Params[0], msg.Trailing)
		}()
	case irc.RPL_NAMREPLY:
		if len(msg.Params) < 3 {
			return nil
		}
		ci := b.lookupChanInfo(msg.Params[2])
		for _, n := range strings.Split(msg.Trailing, " ") {
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
			delete(b.chans, msg.Trailing)
		} else {
			delete(b.lookupChanInfo(msg.Trailing).nicks, msg.Prefix.Name)
		}
	case irc.KICK:
		if len(msg.Params) >= 2 && msg.Params[1] == b.Nick {
			delete(b.chans, msg.Params[0])
		}
	case irc.INVITE:
		if cn := msg.Trailing; len(cn) > 0 && cn[0] == '#' {
			out := irc.Message{Command: irc.JOIN, Params: []string{cn}}
			return b.mc.WriteMsg(out)
		}
	case irc.JOIN:
		if msg.Prefix == nil || msg.Prefix.Name == b.Nick {
			return nil
		}
		chname := msg.Trailing
		b.lookupChanInfo(chname).nicks[msg.Prefix.Name] = struct{}{}
	}
	return nil
}
