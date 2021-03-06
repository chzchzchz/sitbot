package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chzchzchz/sitbot/bot"
	"github.com/chzchzchz/sitbot/msl"
)

func addEvent(p *bot.Profile, ev *msl.Event) {
	pat := bot.Pattern{
		Match:    ev.Match(),
		Template: "mybot " + ev.Name() + " " + ev.Flags(),
	}
	if ev.Type == "text" {
		p.Patterns = append(p.Patterns, pat)
	} else {
		p.PatternsRaw = append(p.PatternsRaw, pat)
	}
}

func writeEvents(w io.Writer, g *msl.Grammar) {
	// Hook events.
	fmt.Fprintf(w, "func addEvents() {\n")
	for _, ev := range g.Events {
		fmt.Fprintf(w, "\truntime.AddEvent(%q, %s)\n", ev.Name(), ev.Name())
	}
	fmt.Fprintf(w, "}\n\n")
	// Emit event handlers.
	for _, ev := range g.Events {
		fmt.Fprintf(w, "// msl event %s:%s\n", ev.Type, ev.Pattern)
		fmt.Fprintf(w, "func %s() {\n", ev.Name())
		fmt.Fprintf(w, "%s", ev.Command.Emit())
		fmt.Fprintf(w, "}\n\n")
	}
}

func writeMain(w io.Writer) {
	fmt.Fprintf(w, `package main

import (
	"github.com/chzchzchz/sitbot/msl/runtime"
)

func main() {
	addEvents()
	runtime.Start()
}
`)
}

func main() {
	fnamep := flag.String("msl", "", "msl script file")
	chanp := flag.String("channel", "#mybot", "channel to autojoin")
	nickp := flag.String("nick", "mybot", "nick for bot")
	passp := flag.String("pass", "", "password for bouncer (e.g., user/network:pass)")
	serverp := flag.String("server", "irc://127.0.0.1:6667", "server for bot")
	verbosep := flag.Bool("verbose", true, "set debug verbosity in bot profile")
	flag.Parse()

	s, err := os.ReadFile(*fnamep)
	if err != nil {
		panic(err)
	}
	g := msl.Grammar{Buffer: string(s), Pretty: true}
	g.Init()
	if err := g.Parse(); err != nil {
		panic(err)
	}
	//g.PrintSyntaxTree()
	g.Execute()

	id := strings.Split(filepath.Base(*fnamep), ".")[0]
	dir := id
	if err := os.MkdirAll(id, 0755); err != nil {
		panic(err)
	}

	prof := bot.Profile{Id: id}
	for _, ev := range g.Events {
		addEvent(&prof, &ev)
	}
	if len(*chanp) != 0 {
		prof.Chans = []string{*chanp}
	}
	prof.Nick, prof.User, prof.Pass, prof.ServerURL = *nickp, *nickp, *passp, *serverp
	if *verbosep {
		prof.Verbosity = 9
	}

	b, err := json.Marshal(prof)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), b, 0644); err != nil {
		panic(err)
	}

	goFile := filepath.Join(dir, id+".go")
	w, err := os.OpenFile(goFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	writeMain(w)
	writeEvents(w, &g)
	w.Close()

	exec.Command("go", "fmt", goFile).Run()
}
