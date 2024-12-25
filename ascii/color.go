package ascii

import (
	"fmt"
	"image/color"
)

type ColorExtent struct {
	Start  int
	Length int
	ColorPair
	CharAttr
}

type CharAttr struct {
	Italic        bool
	Bold          bool
	Underline     bool
	Strikethrough bool
}

var DefaultCharAttr = CharAttr{}
var DefaultColorPair = ColorPair{nil, nil}

type ColorPair struct {
	Foreground color.Color
	Background color.Color
}

func (c ColorPair) MircCode(p *Palette) []byte {
	if c == DefaultColorPair {
		return []byte{'\x0f'}
	}
	if c.Foreground == nil {
		panic("bg but no fg")
	}
	fgc := lookupMircIndex(c.Foreground)
	if fgc == -1 {
		panic("invalid fg")
	}
	ret := []byte{'\x03'}
	if c.Background == nil {
		return append(ret, []byte(fmt.Sprintf("%02d", fgc))...)
	}
	ret = append(ret, []byte(fmt.Sprintf("%d", fgc))...)
	bgc := lookupMircIndex(c.Background)
	if bgc == -1 {
		panic("invalid bg")
	}
	// TODO: peephole optimization when <10 bg byte is not adjacent to numeric.
	ret = append(ret, []byte(fmt.Sprintf(",%02d", bgc))...)
	return ret
}

func ansiCode(inColor color.Color) ([]byte, bool) {
	high := false
	c := lookupAnsiIndex(inColor)
	if c == -1 {
		panic("invalid ansi color")
	}
	ret := []byte(fmt.Sprintf("%d", c%8))
	if c >= 8 {
		high = true
	}
	return append(ret, 'm'), high
}

func (c ColorPair) AnsiCode(p *Palette) []byte {
	if c == DefaultColorPair {
		return []byte("\u001b[0m")
	}
	var ret []byte
	if c.Foreground == nil {
		panic("bg but fg")
	}
	if code, high := ansiCode(c.Foreground); !high {
		ret = append([]byte("\u001b[3"), code...)
	} else {
		ret = append([]byte("\u001b[9"), code...)
	}
	if c.Background == nil {
		return ret
	}
	if code, high := ansiCode(c.Background); !high {
		ret = append(ret, append([]byte("\u001b[4"), code...)...)
	} else {
		ret = append(ret, append([]byte("\u001b[10"), code...)...)
	}
	if string(ret[len(ret)-2:]) == ";1" {
		fmt.Println("hi")
		ret = ret[:len(ret)-2]
	}
	return ret
}
