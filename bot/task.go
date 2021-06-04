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

type TaskId uint64
type Task struct {
	Name    string
	Start   Time
	Command string
	tid     TaskId
	lines   uint32
	mc      *MsgConn
	ctx     context.Context
	cancel  context.CancelFunc
	donec   <-chan struct{}
}
type TaskFunc func(*Task) error

func (t *Task) Lines() uint32 { return atomic.LoadUint32(&t.lines) }

func (t *Task) Write(msg irc.Message) error {
	if err := t.mc.WriteMsg(msg); err != nil {
		return err
	}
	atomic.AddUint32(&t.lines, 1)
	return nil
}

func (t *Task) PipeCmd(cmdtxt, tgt string, env []string) (err error) {
	cctx, cancel := context.WithCancel(t.ctx)
	tidenv := fmt.Sprintf("SITBOT_TID=%d", t.tid)
	toks := strings.Split(cmdtxt, " ")
	cmdname, cmdargs := strings.Replace(toks[0], "/", "_", -1), toks[1:]
	env = append(env, tidenv)
	cmd, err := NewCmd(cctx, "scripts/sandbox", append([]string{cmdname}, cmdargs...), env)
	if err != nil {
		cancel()
		return err
	}
	defer func() {
		cancel()
		if err2 := cmd.Close(); err == nil {
			err = err2
		}
	}()
	for l := range cmd.Lines() {
		out := irc.Message{Command: irc.PRIVMSG, Params: []string{tgt, l}}
		if err := t.Write(out); err != nil {
			return err
		}
		if err := cctx.Err(); err != nil {
			return err
		}
	}
	return err
}

type Tasks struct {
	ctx     context.Context
	cancel  context.CancelFunc
	limiter *rate.Limiter
	Tasks   map[TaskId]*Task
	tid     TaskId
	mc      *MsgConn
	mu      sync.RWMutex
	wg      sync.WaitGroup
}

func NewTasks(ctx context.Context, l *rate.Limiter, mc *MsgConn) *Tasks {
	cctx, cancel := context.WithCancel(ctx)
	return &Tasks{
		ctx:     cctx,
		cancel:  cancel,
		limiter: l,
		mc:      mc,
		Tasks:   make(map[TaskId]*Task),
	}
}

func (t *Tasks) Close() {
	t.cancel()
	t.wg.Wait()
}

func (t *Tasks) Write(tid TaskId, msg irc.Message) error {
	t.mu.RLock()
	tt, ok := t.Tasks[tid]
	t.mu.RUnlock()
	if !ok {
		return io.EOF
	}
	return tt.Write(msg)
}

func (t *Tasks) Kill(tid TaskId) error {
	t.mu.RLock()
	tt, ok := t.Tasks[tid]
	t.mu.RUnlock()
	if !ok {
		return io.EOF
	}
	tt.cancel()
	<-tt.donec
	return nil
}

// Run puts a command in the task list and schedules it to run.
func (t *Tasks) Run(name, cmdtxt string, f TaskFunc) {
	cctx, cancel := context.WithCancel(t.ctx)
	donec := make(chan struct{})
	task := &Task{
		Name: name, Start: Time(time.Now()), Command: cmdtxt,
		mc: t.mc, ctx: cctx, cancel: cancel, donec: donec}
	t.mu.Lock()
	t.tid++
	tid := t.tid
	t.Tasks[tid], task.tid = task, tid
	t.mu.Unlock()
	t.wg.Add(1)
	go func() {
		defer func() {
			t.mu.Lock()
			delete(t.Tasks, task.tid)
			t.mu.Unlock()
			close(donec)
			t.wg.Done()
		}()
		if t.limiter.Wait(task.ctx) != nil {
			return
		}
		if err := f(task); err != nil {
			log.Printf("[task] failed on command %q (%v)", task.Command, err)
		}
	}()
}
