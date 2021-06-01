package bot

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
)

type Cmd struct {
	donec  chan struct{}
	linec  chan string
	err    error
	closer io.Closer
}

func NewCmd(ctx context.Context, cmdname string, args []string, env []string) (*Cmd, error) {
	donec, linec := make(chan struct{}), make(chan string, 5)
	cmd := exec.CommandContext(ctx, cmdname, args...)
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &Cmd{donec: donec, linec: linec, closer: stdout}
	lr := bufio.NewReader(stdout)
	go func() {
		defer func() {
			close(linec)
			stdout.Close()
			if err := cmd.Wait(); c.err == nil {
				c.err = err
			}
			close(donec)
		}()
		for {
			l, err := lr.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					c.err = err
				}
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
	c.closer.Close()
	<-c.donec
	return c.err
}
