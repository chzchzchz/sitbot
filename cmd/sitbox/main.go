package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/chzchzchz/sitbot/bot"
)

const BackoffBase = time.Second
const BackoffMul = 2
const LineTimeout = 5 * time.Second

func main() {
	if strings.Contains(os.Args[1], "/") {
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.TODO())
	cmdname := "scripts/" + os.Args[1]
	cmd, err := bot.NewCmd(ctx, cmdname, os.Args[2:], nil)
	if err != nil {
		cancel()
		os.Exit(1)
	}
	defer func() {
		cancel()
		cmd.Close()
		if rc := cmd.ProcessState.ExitCode(); rc != 0 {
			os.Exit(rc)
		}
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
			os.Exit(2)
		}
	}
}
