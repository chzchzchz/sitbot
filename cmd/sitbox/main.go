package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"time"
)

const BackoffBase = time.Second
const BackoffMul = 2
const LineTimeout = 5 * time.Second

type Cmd struct {
	donec chan struct{}
	linec chan string
	err   error
}

func NewCmd(ctx context.Context, cmdname, args string) (*Cmd, error) {
	donec, linec := make(chan struct{}), make(chan string, 5)
	cmd := exec.CommandContext(ctx, "scripts/"+cmdname, args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &Cmd{donec: donec, linec: linec}
	lr := bufio.NewReader(stdout)
	go func() {
		defer func() {
			close(linec)
			stdout.Close()
			if err := cmd.Wait(); err != nil {
				c.err = err
			}
			close(donec)
		}()
		for {
			l, err := lr.ReadString('\n')
			if err != nil {
				c.err = err
				if l == "" {
					break
				}
			}
			select {
			case linec <- l:
			case <-ctx.Done():
				return
			}
		}
	}()
	return c, nil
}

func (c *Cmd) Lines() <-chan string { return c.linec }

func (c *Cmd) Close() error {
	<-c.donec
	return c.err
}

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	cmd, err := NewCmd(ctx, os.Args[1], os.Args[2])
	if err != nil {
		panic(err)
	}
	defer func() {
		cancel()
		cmd.Close()
	}()
	lastline := ""
	backoff := BackoffBase
	for {
		select {
		case l, ok := <-cmd.Lines():
			if !ok {
				return
			}
			if l == lastline {
				backoff = BackoffMul * backoff
				time.Sleep(backoff)
			} else {
				backoff = BackoffBase
				lastline = l
			}
			if _, err := os.Stdout.WriteString(l); err != nil {
				return
			}
		case <-time.After(LineTimeout):
			os.Stdout.WriteString("TIMEOUT\n")
			return
		}
	}
}
