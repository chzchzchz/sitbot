package bot

import (
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/sorcix/irc.v2"
)

type MsgConnStats struct {
	txMsgs uint64
	rxMsgs uint64
}

func (m *MsgConnStats) TxMsgs() uint64 { return atomic.LoadUint64(&m.txMsgs) }
func (m *MsgConnStats) RxMsgs() uint64 { return atomic.LoadUint64(&m.rxMsgs) }

type MsgConn struct {
	*irc.Conn
	MsgConnStats
	ctx    context.Context
	wg     sync.WaitGroup
	readc  chan irc.Message
	writec chan irc.Message
}

func NewMsgConn(ctx context.Context, conn net.Conn, invl time.Duration) (*MsgConn, error) {
	cctx, cancel := context.WithCancel(ctx)
	mc := &MsgConn{
		Conn:   irc.NewConn(conn),
		ctx:    cctx,
		readc:  make(chan irc.Message, 16),
		writec: make(chan irc.Message),
	}
	mc.wg.Add(2)
	stopf := func() {
		cancel()
		mc.wg.Done()
	}
	go func() {
		defer func() {
			stopf()
			close(mc.readc)
		}()
		for {
			msg, err := mc.Decode()
			if err != nil {
				mc.Conn.Close()
				return
			}
			if msg == nil {
				log.Printf("got nil message on %s", conn.RemoteAddr().String())
				continue
			}
			select {
			case mc.readc <- *msg:
				atomic.AddUint64(&mc.rxMsgs, 1)
			case <-mc.ctx.Done():
			}
		}
	}()
	go func() {
		defer func() {
			mc.Conn.Close()
			stopf()
		}()
		l := rate.NewLimiter(rate.Every(invl), 1)
		for {
			if err := l.Wait(mc.ctx); err != nil {
				return
			}
			select {
			case msg := <-mc.writec:
				atomic.AddUint64(&mc.txMsgs, 1)
				if mc.Encode(&msg) != nil {
					return
				}
			case <-mc.ctx.Done():
				return
			}
		}
	}()
	return mc, nil
}

func (mc *MsgConn) WriteMsg(m irc.Message) error {
	select {
	case mc.writec <- m:
		return nil
	case <-mc.ctx.Done():
		return mc.ctx.Err()
	}
}

func (mc *MsgConn) ReadChan() <-chan irc.Message { return mc.readc }

func (mc *MsgConn) Close() error {
	err := mc.Conn.Close()
	mc.wg.Wait()
	return err
}

type TeeMsgConn struct {
	*MsgConn
	rchans []readChan
	mu     sync.Mutex
}

type readChan struct {
	readc chan irc.Message
	donec <-chan struct{}
}

func NewTeeMsgConn(ctx context.Context, conn net.Conn, ms int) (*TeeMsgConn, error) {
	mc, err := NewMsgConn(ctx, conn, time.Duration(ms)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	return &TeeMsgConn{MsgConn: mc}, nil
}

func (tmc *TeeMsgConn) start() {
	tmc.wg.Add(1)
	go func() {
		defer func() {
			for _, rc := range tmc.rchans {
				close(rc.readc)
			}
			tmc.wg.Done()
		}()
		for msg := range tmc.readc {
			tmc.mu.Lock()
			rchans := tmc.rchans
			tmc.mu.Unlock()
			for _, rc := range rchans {
				select {
				case rc.readc <- msg:
				case <-rc.donec:
				case <-tmc.ctx.Done():
					return
				}
			}
		}
	}()
}

func (tmc *TeeMsgConn) NewReadChan() (<-chan irc.Message, chan<- struct{}) {
	donec := make(chan struct{})
	rc := readChan{readc: make(chan irc.Message, 16), donec: donec}
	tmc.mu.Lock()
	oldrchans := tmc.rchans
	tmc.rchans = append(tmc.rchans, rc)
	tmc.mu.Unlock()
	if oldrchans == nil {
		tmc.start()
	}
	return rc.readc, donec
}

func (mc *TeeMsgConn) DropReadChan(rc <-chan irc.Message) {
	defer mc.mu.Unlock()
	mc.mu.Lock()
	if len(mc.rchans) == 0 {
		return
	}
	rchans := make([]readChan, 0, len(mc.rchans)-1)
	for _, mcrc := range mc.rchans {
		if rc != mcrc.readc {
			rchans = append(rchans, mcrc)
		}
	}
	mc.rchans = rchans
}
