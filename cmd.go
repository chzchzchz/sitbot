package main

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

type Cmd struct {
	donec chan struct{}
	linec chan string
	err   error
}

func NewCmd(ctx context.Context, cmdtxt string) (*Cmd, error) {
	donec, linec := make(chan struct{}), make(chan string, 5)
	toks := strings.Split(cmdtxt, " ")
	cmdname, cmdargs := strings.Replace(toks[0], "/", "_", -1), ""
	if len(toks) > 1 {
		cmdargs = strings.Join(toks[1:], " ")
	}
	cmdpath := filepath.Join("scripts", cmdname)
	cmd := exec.CommandContext(ctx, cmdpath, cmdargs)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
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
