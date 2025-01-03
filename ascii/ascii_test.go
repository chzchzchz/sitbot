package ascii

import (
	"strings"
	"testing"
)

func cellCheck(a *ASCII, f func(c *Cell) bool) bool {
	for x := 0; x < a.Columns(); x++ {
		for y := 0; y < a.Rows(); y++ {
			if c := a.Get(x, y); c != nil && f(c) {
				return true
			}
		}
	}
	return false
}

func hasRune(a *ASCII, r rune) bool {
	return cellCheck(a, func(c *Cell) bool { return c.Value == r })
}

func hasColorPair(a *ASCII, c ColorPair) bool {
	return cellCheck(a, func(cc *Cell) bool { return cc.ColorPair == c })
}

func TestControlCodes(t *testing.T) {
	s := `11,11     1,1112,12         1,1112,12    0,0    1,1111,11  `
	a, _ := NewASCII(s)
	if hasRune(a, '\x03') {
		t.Errorf("found color code")
	}
}

func PrettyFormat(ircText string) string {
	out := strings.ReplaceAll(ircText, "\x03", "\x1b[033m[C]\x1b[0m")
	out = strings.ReplaceAll(out, "\x02", "\x1b[034m[B]\x1b[0m")
	out = strings.ReplaceAll(out, "\x1F", "\x1b[035m[U]\x1b[0m")
	out = strings.ReplaceAll(out, "\x1D", "\x1b[036m[I]\x1b[0m")
	return out
}

func PrintExpectedVsParsed(expected string, ascii *ASCII, t *testing.T) {
	t.Errorf(" Input   : %s", PrettyFormat(expected))
	t.Errorf(" Parsed  : %s", PrettyFormat(string(ascii.Bytes())))
}

func TestColorCodes(t *testing.T) {
	s := "hello \x0315,01hello"
	//    012345         6789
	a, _ := NewASCII(s)
	if c := a.Get(2, 0); c == nil || c.Value != 'l' {
		t.Errorf("Expected l but found %c", c.Value)
	}
	if c := a.Get(2, 0); c == nil || c.Foreground != nil {
		t.Errorf("Expected nil but found %v", c.Foreground)
	}
	if c := a.Get(2, 0); c == nil || c.Background != nil {
		t.Errorf("Expected nil but found %v", c.Background)
	}

	if c := a.Get(7, 0); c == nil || c.Value != 'e' {
		t.Errorf("Expected e but found %c", c.Value)
	}
	fg, _ := MircColor(15)
	if c := a.Get(7, 0); c == nil || c.Foreground != fg {
		t.Errorf("Expected 15 but found %v", c.Foreground)
	}
	bg, _ := MircColor(1)
	if c := a.Get(7, 0); c == nil || c.Background != bg {
		t.Errorf("Expected 1 but found %v", c.Background)
	}
}

func TestColorCodesBackToBack(t *testing.T) {
	s := "he\x037,01he\x03\x0315lo"
	//    01        23          45
	a, _ := NewASCII(s)

	c := a.Get(4, 0)
	if c == nil {
		t.Error("Expected l at pos 4 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'l' {
			t.Errorf("Expected l at pos 4 but found '%c'", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(15)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
	}

	c = a.Get(5, 0)
	if c == nil {
		t.Error("Expected o at pos 5 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Background != nil {
			t.Errorf("Expected nil but found %v", c.Background)
		}
	}
}

func TestColorCodesNoText(t *testing.T) {
	s := "ab\x037\x034cd"
	//    01          23
	a, _ := NewASCII(s)

	c := a.Get(2, 0)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(4)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}
func TestColorCodesNoText2d(t *testing.T) {
	s := "ab\x0315\x0314cd"
	//    01            23
	a, _ := NewASCII(s)

	c := a.Get(2, 0)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(14)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestBoldAfterColorCode(t *testing.T) {
	s := "ab\x0315\x02cd"
	//    01          23
	a, _ := NewASCII(s)

	c := a.Get(2, 0)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(15)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
		if !c.Bold {
			t.Errorf("Expected bold pos 2 but found %v", c.Bold)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestColorAfterBoldAfterColorCode(t *testing.T) {
	s := "ab\x0315\x02\x0314cd"
	//    01                23
	a, _ := NewASCII(s)

	c := a.Get(2, 0)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(14)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
		if !c.Bold {
			t.Errorf("Expected bold pos 2 but found %v", c.Bold)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestLeadingColorAfterItalicAfterColorCode(t *testing.T) {
	s := "\x0315\x1D\x0314cd"
	//                    01
	a, _ := NewASCII(s)

	c := a.Get(0, 0)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(14)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
		if !c.Italic {
			t.Errorf("Expected bold pos 2 but found %v", c.Bold)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestLeadingColorAfterItalicAfterColorCodeAfterNewlineAfterColor(t *testing.T) {
	s := "\x0315\n\x0313\x1D\x0314cd"
	//                    01
	a, _ := NewASCII(s)

	c := a.Get(0, 1)
	if c == nil {
		t.Error("Expected c at pos 2 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'c' {
			t.Errorf("Expected c at pos 2 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(14)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 2 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
		if !c.Italic {
			t.Errorf("Expected italic pos 2 but found %v", c.Bold)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestColoredComma(t *testing.T) {
	s := "\x0315, "
	//          01
	a, _ := NewASCII(s)

	c := a.Get(0, 0)
	if c == nil {
		t.Error("Expected , at pos 0 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != ',' {
			t.Errorf("Expected , at pos 0 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(15)
		if c.Foreground != fg {
			t.Errorf("Expected color 15 at pos 0 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestBackgroundPersistsAfterBackToBack(t *testing.T) {
	s := "\x0315,01\x0303\x0304moo"
	//                         012
	a, _ := NewASCII(s)

	c := a.Get(0, 0)
	if c == nil {
		t.Error("Expected m at pos 0 but found nil")
		PrintExpectedVsParsed(s, a, t)
	} else {
		if c.Value != 'm' {
			t.Errorf("Expected m at pos 0 but found %c", c.Value)
			PrintExpectedVsParsed(s, a, t)
		}
		fg, _ := MircColor(4)
		if c.Foreground != fg {
			t.Errorf("Expected color 4 at pos 0 but found %v", c.Foreground)
			PrintExpectedVsParsed(s, a, t)
		}
		bg, _ := MircColor(1)
		if c.Background != bg {
			t.Errorf("Expected color 1 at pos 0 but found %v", c.Background)
			PrintExpectedVsParsed(s, a, t)
		}
	}
}

func TestColorCancelAteChars(t *testing.T) {
	s := `11,11  test`
	//          01 2345
	a, _ := NewASCII(s)
	if c := a.Get(2, 0); c == nil || c.Value != 't' {
		t.Errorf("Expected t but found %c", c.Value)
	}
}

func TestFGColorLeadingZero(t *testing.T) {
	s := `09test`
	//       0123
	a, _ := NewASCII(s)
	c := a.Get(0, 0)
	if c == nil || c.Value != 't' {
		t.Errorf("Expected t but found %c", c.Value)
	}
	fg, _ := MircColor(9)
	if c.Foreground != fg {
		t.Errorf("Expected fg %v but found %v", fg, c.Foreground)
	}
	if c.Background != nil {
		t.Errorf("Expected bg %v but found %v", nil, c.Background)
	}
}

func TestBGColorLeadingZero(t *testing.T) {
	s := `9,09test`
	//         0123
	a, _ := NewASCII(s)
	c := a.Get(0, 0)
	if c == nil || c.Value != 't' {
		t.Errorf("Expected t but found %c", c.Value)
	}
	fg, _ := MircColor(9)
	if c.Foreground != fg {
		t.Errorf("Expected fg %v but found %v", fg, c.Foreground)
	}
	bg, _ := MircColor(9)
	if c.Background != bg {
		t.Errorf("Expected bg %v but found %v", bg, c.Background)
	}
}

func TestFGBGColorLeadingZero(t *testing.T) {
	s := `09,08test`
	//          0123
	a, _ := NewASCII(s)
	c := a.Get(0, 0)
	if c == nil || c.Value != 't' {
		t.Errorf("Expected t but found %c", c.Value)
	}
	fg, _ := MircColor(9)
	if c.Foreground != fg {
		t.Errorf("Expected fg %v but found %v", fg, c.Foreground)
	}
	bg, _ := MircColor(8)
	if c.Background != bg {
		t.Errorf("Expected bg %v but found %v", bg, c.Background)
	}
}

func TestColorReload(t *testing.T) {
	// expect black background
	s := `11,11     1,011111111111111111111111,11 1,11jjj11,11   `
	bgc, _ := MircColor(1)
	fgc, _ := MircColor(1)
	a, _ := NewASCII(s)
	if !hasColorPair(a, ColorPair{fgc, bgc}) {
		t.Errorf("could not find 1,1 pair")
	}
	b := a.Bytes()
	a2, _ := NewASCII(string(b))
	if !hasColorPair(a2, ColorPair{fgc, bgc}) {
		t.Errorf("could not find 1,1 pair")
	}
}
