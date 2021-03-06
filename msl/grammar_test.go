package msl

import (
	"fmt"
	"testing"
)

func makeEvent(s string) *Event {
	ev := fmt.Sprintf("on *:TEXT:test_cmd:#: {\n%s\n}", s)
	g := &Grammar{Buffer: ev}
	g.Init()
	g.Parse()
	// g.PrintSyntaxTree()
	g.Execute()
	return &g.Events[0]
}

func makeCommand(s string) Command {
	return (makeEvent(s).Command.(*Block)).commands[0]
}

func TestVar(t *testing.T) {
	for _, tt := range []string{
		"var %r = $rand(1, 40)",
		"var %q = $right(%k,3)",
	} {
		if _, ok := makeCommand(tt).(*Statement); !ok {
			t.Fail()
		}
	}
}

func TestIf(t *testing.T) {
	tts := []struct {
		in string
	}{
		{"if (%spin$nick != 0) { msg $chan hello }"},
		{"if ($len(%pot) >= 5) { msg $chan ok }"},
	}
	for _, tt := range tts {
		cmd := makeCommand(tt.in)
		if _, ok := cmd.(*IfCmd); !ok {
			t.Errorf("no convert: %s -> %+v", tt.in, cmd)
		}
		t.Logf("%+v", cmd)
	}
}

func TestStrings(t *testing.T) {
	tts := []struct {
		in string
		n  int
	}{
		// "msg" "$chan" "2" ",10$ $+ %money [ $+ [ $nick ] ] $+ ."
		{"msg $chan 2,10$ $+ %money [ $+ [ $nick ] ] $+ .", 4},
		// "inc" "%total $+ $nick"
		//     "%card [ $+ [ [ %blackjack [ $+ [ $nick ] ] ] [ $+ [ $nick ] ] ] ]"
		{"/inc %total $+ $nick %card [ $+ [ [ %blackjack [ $+ [ $nick ] ] ] [ $+ [ $nick ] ] ] ]", 3},
		{"/set %card $+ [ %blackjack [ $+ [ $nick ] ] ] $+ $nick $rand(1,13)", 3},
		{"msg $chan ^C2,10escalator (abc).", 4},
		{"msg $chan ^C2,10escalator (a b c).", 6},
		{"msg $chan (in)", 3},
	}
	for _, tt := range tts {
		stmt, ok := makeCommand(tt.in).(*Statement)
		if !ok || tt.n != len(stmt.Values) {
			if ok {
				for i, v := range stmt.Values {
					t.Logf("%d: %+v\n", i, v)
				}
			}
			t.Fail()
		}
	}
}
