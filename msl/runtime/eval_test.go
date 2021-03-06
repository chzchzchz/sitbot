package runtime

import (
	"math/rand"
	"testing"
)

func TestVarSet(t *testing.T) {
	mslEv.Nick, mslEv.Msg = "testnick", "abc defg xyzzz"
	mslVar.Globals["moneytestnick"] = "1123.1"
	mslVar.Globals["dealerbjtestnick"] = "1"
	mslVar.Globals["dealer1testnick"] = "2"

	rand.Seed(1)
	tts := []struct {
		in  string
		out string
	}{
		{"var", "var"},
		{"var ", "var"},
		{"%r", "$null"},
		{"%money $+ $nick", "1123.1"},
		{"\x038,9$nick", "\x038,9testnick"},
		{"\x034,41$9000", "\x034,41$9000"},
		{"4,14 $+ $nick $+ ,", "4,14testnick,"},
		{"4,14", "4,14"},
		{"4,1$000", "4,1$000"},
		{"$replace(%money [ $+ [ $nick ] ],.1,.10)", "1123.10"},
		{"$rand(1,10)", "1"},
		{"2,10$%money$nick", "2,10$1123.1"},
		{"$bytes(%money$nick,b)", "1,123"},
		{"$len(%money$nick)", "6"},
		{"%dealer $+ [ %dealerbj [ $+ [ $nick ] ] ] $+ $nick", "2"},
		{"0$ $+ 1", "0$1"},
		{"%money [ $+ [ $nick ] ] $+ .", "1123.1."},
		{"%money [ $+  [ $nick ] ]", "1123.1"},
		{"%moneytestnick", "1123.1"},
		{"\x0f\x038,4$5000!!! $+ \x0311,6", "\x0f\x038,4$5000!!!\x0311,6"},
		{"$bytes(%moneytestnick,b)", "1,123"},
		{"8,7$1,000,000", "8,7$1,000,000"},
	}
	for _, tt := range tts {
		if v := eval(tt.in); v != tt.out {
			t.Errorf("%q: wanted %q got %q", tt.in, tt.out, v)
		}
	}
}

func TestCondition(t *testing.T) {
	mslEv.Nick = "testnick"
	mslEv.Msg = "abc 1234.5 xyzw"
	mslVar.Globals["spintestnick"] = "1"
	mslVar.Globals["moneytestnick"] = "123.1"

	tts := []struct {
		in  string
		out string
	}{
		{"(%spin [ $+ [ $nick ] ] != 0)", "true"},
		{"(%spin [ $+ [ $nick ] ] == 0)", "false"},
		{"(%money [ $+ [ $nick ] ] == 0) || (%money [ $+ [ $nick ] ] == $null)", "false"},
		{"($len($1) == 3)", "true"},
		{"($len($2) == 6)", "true"},
		{"($len($2-) == 11)", "true"},
		{"((1 == 1) && ((1234 < 5000) || (1234 > 20000)))", "true"},
		{"(($2 < 5000) || ($2 > 20000))", "true"},
		{"(%spin [ $+ [ $nick ] ] == 1)", "true"},
		{"(1 == 1 && 6 >= 5)", "true"},
		{"(. isin %money [ $+ [ $nick ] ] && 6 >= 5)", "true"},
		{"($len(%spintestnick) >= 5)", "false"},
	}
	for _, tt := range tts {
		if v := eval(tt.in); v != tt.out {
			t.Errorf("%q: wanted %q got %q", tt.in, tt.out, v)
		}
	}
}
