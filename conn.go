package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/sorcix/irc"
	"golang.org/x/time/rate"
)

type MsgConn struct {
	*irc.Conn
	ctx    context.Context
	wg     sync.WaitGroup
	readc  chan irc.Message
	writec chan irc.Message
}

func (mc *MsgConn) ReadChan() <-chan irc.Message {
	return mc.readc
}

func (mc *MsgConn) WriteMsg(m irc.Message) error {
	select {
	case mc.writec <- m:
		return nil
	case <-mc.ctx.Done():
		return mc.ctx.Err()
	}
}

func (mc *MsgConn) WriteChan() chan<- irc.Message { return mc.writec }

func (mc *MsgConn) DoneChan() <-chan struct{} { return mc.ctx.Done() }

func NewMsgConn(ctx context.Context, serv string) (*MsgConn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", serv)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	mc := &MsgConn{
		Conn:   irc.NewConn(conn),
		ctx:    ctx,
		readc:  make(chan irc.Message, 1),
		writec: make(chan irc.Message, 16),
	}
	mc.wg.Add(2)
	stopf := func() {
		cancel()
		mc.wg.Done()
	}
	go func() {
		defer stopf()
		for {
			msg, err := mc.Decode()
			if err != nil {
				mc.Conn.Close()
				close(mc.readc)
				return
			}
			mc.readc <- *msg
		}
	}()
	go func() {
		defer func() {
			mc.Conn.Close()
			stopf()
		}()
		l := rate.NewLimiter(rate.Every(time.Second), 5)
		for {
			select {
			case msg := <-mc.writec:
				if l.Wait(mc.ctx) != nil {
					return
				}
				if err := mc.Encode(&msg); err != nil {
					log.Printf("failed to write.. %v\n", err)
					return
				}
			case <-mc.ctx.Done():
				return
			}
		}
	}()
	return mc, nil
}

func (mc *MsgConn) Close() error {
	err := mc.Conn.Close()
	mc.wg.Wait()
	return err
}
