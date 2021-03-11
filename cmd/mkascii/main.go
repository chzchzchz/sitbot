package main

import (
	"io"
	"os"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil && err != io.EOF {
		panic(err)
	}
	g := Grammar{Buffer: string(in)}
	g.Init()
	if err := g.Parse(); err != nil {
		panic(err)
	}
	g.Execute()
	a, err := run(g.script.stmts)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(a.Bytes())
}
