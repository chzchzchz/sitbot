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
	Italic    bool
	Bold      bool
	Underline bool
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
	ret = append(ret, []byte(fmt.Sprintf("%d", fgc))...)
	if c.Background == nil {
		return ret
	}
	bgc := lookupMircIndex(c.Background)
	if bgc == -1 {
		panic("invalid bg")
	}
	ret = append(ret, []byte(fmt.Sprintf(",%d", bgc))...)
	return ret
}

func ansiCode(inColor color.Color) []byte {
	c := lookupAnsiIndex(inColor)
	if c == -1 {
		panic("invalid ansi color")
	}
	ret := []byte(fmt.Sprintf("%d", c%8))
	if c >= 8 {
		ret = append(ret, []byte(";1")...)
	}
	return append(ret, 'm')
}

func (c ColorPair) AnsiCode(p *Palette) []byte {
	if c == DefaultColorPair {
		return []byte("\u001b[0m")
	}
	if c.Foreground == nil {
		panic("bg but fg")
	}
	ret := append([]byte("\u001b[3"), ansiCode(c.Foreground)...)
	if c.Background == nil {
		return ret
	}
	ret = append(ret, []byte("\u001b[4")...)
	ret = append(ret, ansiCode(c.Background)...)
	if string(ret[len(ret)-2:]) == ";1" {
		fmt.Println("hi")
		ret = ret[:len(ret)-2]
	}
	return ret
}
