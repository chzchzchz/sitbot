package main

import (
	"flag"
	"io"
	"os"
)

func main() {
	showAnsi := false
	flag.BoolVar(&showAnsi, "ansi", false, "display ansi colors")
	flag.Parse()

	in, err := io.ReadAll(os.Stdin)
	if err != nil && err != io.EOF {
		panic(err)
	}
	g := Grammar{Buffer: string(in), script: Script{groups: make(map[string]*Group)}}
	g.Init()
	if err := g.Parse(); err != nil {
		panic(err)
	}
	g.Execute()
	a, err := run(g.script.stmts)
	if err != nil {
		panic(err)
	}
	if showAnsi {
		os.Stdout.Write(a.AnsiBytes())
	} else {
		os.Stdout.Write(a.Bytes())
	}
}
