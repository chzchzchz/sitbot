package ascii

import (
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

func TestColorCancelAteChars(t *testing.T) {
	s := `11,11  test`
	//          01 2345
	a, _ := NewASCII(s)
	if c := a.Get(2, 0); c == nil || c.Value != 't' {
		t.Errorf("Expected t but found %c", c.Value)
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
