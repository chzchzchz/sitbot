package bouncer

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/chzchzchz/sitbot/bot"
	"gopkg.in/sorcix/irc.v2"
)

type Bouncer struct {
	ln     net.Listener
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	b      *bot.Bot
}

func NewBouncer(b *bot.Bot, serv string) (*Bouncer, error) {
	ln, err := net.Listen("tcp", serv)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(b.Ctx())
	bounce := &Bouncer{ln: ln, ctx: ctx, cancel: cancel, b: b}
	bounce.wg.Add(1)
	go func() {
		defer func() {
			ln.Close()
			cancel()
			bounce.wg.Done()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			mc, err := bot.NewMsgConn(ctx, conn, time.Millisecond)
			if err != nil {
				log.Println(err)
				return
			}
			bounce.wg.Add(1)
			go func() {
				defer bounce.wg.Done()
				err := bounce.handleConn(mc)
				log.Printf("bouncer closing %v (%v)", conn.RemoteAddr(), err)
			}()
		}
	}()
	return bounce, nil
}

func (bounce *Bouncer) handleConn(mc *bot.MsgConn) error {
	var wg sync.WaitGroup
	defer func() {
		mc.Close()
		wg.Wait()
	}()
	handshaking := true
	for handshaking {
		select {
		case msg, ok := <-mc.ReadChan():
			log.Printf("handshake %+v", msg)
			handshaking = ok
		case <-time.After(time.Second):
			handshaking = false
		}
	}
	select {
	case <-bounce.b.Welcomec:
	case <-time.After(time.Second):
		return io.EOF
	}
	date := time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	cn := bounce.b.Nick
	nnick, nnpfx := cn+"!bot@masked", bounce.b.Netpfx
	nn := bounce.b.Netpfx.String()
	for _, msg := range []irc.Message{
		{
			Prefix:  nnpfx,
			Command: irc.RPL_WELCOME,
			Params:  []string{cn, "Welcome to the bouncer " + nnick},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_YOURHOST,
			Params: []string{
				cn, "Your host is " + nn + ", running version sitbot-1.0"},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_CREATED,
			Params:  []string{cn, "This server was created " + date},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_MYINFO,
			Params:  []string{cn, nn + " v1.0 iosw biklmnopstv bklov"},
		},
		{
			Command: irc.PING,
			Params:  []string{cn, "hello"},
		},
	} {
		if err := mc.WriteMsg(msg); err != nil {
			return err
		}
	}
	nnpfx2 := &irc.Prefix{Name: nnick}
	bounce.b.RLock()
	// TODO: get channel data from state watcher, not bot
	for c := range bounce.b.Channels {
		// Have chat server return names list for channel as if joined.
		wg.Add(1)
		go func(chn string) {
			defer wg.Done()
			msg := irc.Message{Prefix: nnpfx2, Command: irc.JOIN, Params: []string{chn}}
			mc.WriteMsg(msg)
			msg = irc.Message{Command: irc.NAMES, Params: []string{chn}}
			bounce.b.TeeMsg().WriteMsg(msg)
		}(c)
	}
	bounce.b.RUnlock()

	brc, bdc := bounce.b.TeeMsg().NewReadChan()
	defer close(bdc)
	for {
		var tgtmc *bot.MsgConn
		var msg irc.Message
		var ok bool
		select {
		case msg, ok = <-mc.ReadChan():
			log.Printf("bouncer got from client %+v", msg)
			if msg.Command == irc.QUIT {
				return io.EOF
			}
			if msg.Command == irc.PING {
				msg.Command, tgtmc = irc.PONG, mc
				break
			}
			tgtmc = bounce.b.TeeMsg().MsgConn
			switch msg.Command {
			case irc.WHO, irc.NAMES, irc.MODE, irc.PRIVMSG, irc.KICK:
			default:
				msg.Command = irc.PING
			}
		case msg, ok = <-brc:
			tgtmc = mc
		}
		if !ok {
			return io.EOF
		}
		if msg.Command == irc.PING {
			continue
		}
		log.Printf("bouncer relaying %+v", msg)
		if err := tgtmc.WriteMsg(msg); err != nil {
			return err
		}
	}
}

func (b *Bouncer) Close() {
	b.ln.Close()
	b.cancel()
	b.wg.Wait()
}
