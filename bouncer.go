package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"gopkg.in/sorcix/irc.v2"
)

type Bouncer struct {
	ln     net.Listener
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	bot    *Bot
}

func NewBouncer(bot *Bot, serv string) (*Bouncer, error) {
	ln, err := net.Listen("tcp", serv)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &Bouncer{
		ln:     ln,
		ctx:    ctx,
		cancel: cancel,
		bot:    bot}
	b.wg.Add(1)
	go func() {
		defer func() {
			cancel()
			b.wg.Done()
		}()
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				return
			}
			mc, err := NewMsgConn(ctx, conn, time.Millisecond)
			if err != nil {
				log.Println(err)
				return
			}
			b.wg.Add(1)
			go func() {
				defer b.wg.Done()
				if err := b.handleConn(mc); err != nil {
					log.Printf("bouncer closing %v (%v)",
						conn.RemoteAddr(),
						err)
				}
			}()
		}
	}()
	return b, nil
}

func (b *Bouncer) handleConn(mc *MsgConn) error {
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
	case <-b.bot.welcomec:
	case <-time.After(time.Second):
		return io.EOF
	}
	date := time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	nnick := b.bot.Nick + "!bot@masked"
	nnpfx := b.bot.netpfx
	nn := b.bot.netpfx.String()
	for _, msg := range []irc.Message{
		{
			Prefix:  nnpfx,
			Command: irc.RPL_WELCOME,
			Params: []string{
				b.bot.Nick,
				"Welcome to the bouncer " + nnick,
			},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_YOURHOST,
			Params: []string{b.bot.Nick,
				"Your host is " + nn + ", running version sitbot-1.0"},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_CREATED,
			Params:  []string{b.bot.Nick, "This server was created " + date},
		},
		{
			Prefix:  nnpfx,
			Command: irc.RPL_MYINFO,
			Params:  []string{b.bot.Nick, nn + " v1.0 iosw biklmnopstv bklov"},
		},
		{
			Command: irc.PING,
			Params:  []string{b.bot.Nick, "hello"},
		},
	} {
		if err := mc.WriteMsg(msg); err != nil {
			return err
		}
	}
	nnpfx2 := &irc.Prefix{Name: nnick}
	for _, c := range b.bot.Channels() {
		msg := irc.Message{Prefix: nnpfx2, Command: irc.JOIN, Params: []string{c}}
		if err := mc.WriteMsg(msg); err != nil {
			return err
		}
		// Have chat server return names list for channel as if joined.
		wg.Add(1)
		go func(chn string) {
			defer wg.Done()
			msg := irc.Message{Command: irc.NAMES, Params: []string{chn}}
			b.bot.mc.WriteMsg(msg)
		}(c)
	}
	brc, bdc := b.bot.mc.NewReadChan()
	defer close(bdc)
	for {
		var tgtmc *MsgConn
		var msg irc.Message
		var ok bool
		select {
		case msg, ok = <-mc.ReadChan():
			log.Printf("bouncer got from client %+v", msg)
			if msg.Command == irc.QUIT {
				return io.EOF
			}
			if msg.Command == irc.PING {
				tgtmc = mc
				msg.Command = irc.PONG
				break
			}
			tgtmc = b.bot.mc.MsgConn
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
