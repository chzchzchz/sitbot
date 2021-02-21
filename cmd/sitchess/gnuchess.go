package main

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"
)

func GetMove(epdPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gnuchess")
	w, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	defer w.Close()
	go func() {
		io.WriteString(w, "epdload "+epdPath+"\n")
		time.Sleep(time.Second)
		io.WriteString(w, "easy\ngo\nexit\nexit\nexit\n")
	}()
	v, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, s := range strings.Split(string(v), "\n") {
		if !strings.Contains(s, "My move") {
			continue
		}
		toks := strings.Split(strings.TrimSpace(s), " ")
		return toks[len(toks)-1], nil
	}
	return "", io.EOF
}
