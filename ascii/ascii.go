package ascii

import (
	"errors"
	"image"
	"strings"
	"unicode/utf8"
)

var ErrBadMircCode = errors.New("bad mirc code")
var ErrBadCellCoord = errors.New("out of range")

type Cell struct {
	ColorPair
	Value rune
	CharAttr
}

type ASCII struct {
	// Cells is indexed via [y][x].
	Cells [][]Cell
}

func NewASCII(dat string) (*ASCII, error) {
	var cells [][]Cell
	var row []Cell
	chompState, fg, bg, fgs, bgs := 0, -1, -1, 0, 0
	for _, v := range dat {
		switch v {
		case '\r':
			continue
		case '\n':
			cells = append(cells, row)
			chompState, fg, bg = 0, -1, -1
			row = nil
			continue
		}
		switch chompState {
		case 0:
			switch v {
			case '\x02':
				continue
			case '\x0f':
				fg, bg = -1, -1
				continue
			case '\x03':
				chompState, fg = 1, -1
				continue
			}
		case 1:
			if v >= '0' && v <= '9' && fg != 0 && fgs < 2 {
				if fg == -1 {
					fg = 0
				}
				fg, fgs = fg*10+int(v-'0'), fgs+1
				continue
			}
			chompState, fgs = 0, 0
			if fg == -1 {
				// Reset color codes.
				fg, bg = -1, -1
				continue
			}
			if v == ',' {
				chompState, bg = 2, -1
				continue
			}
		case 2:
			if v >= '0' && v <= '9' && bgs < 2 {
				if bg == -1 {
					bg = 0
				}
				bg, bgs = bg*10+int(v-'0'), bgs+1
				if bgs == 2 {
					bgs, chompState = 0, 0
				}
				continue
			}
			bgs, chompState = 0, 0
			if bg == -1 {
				return nil, ErrBadMircCode
			}
		}
		fgc, err := MircColor(fg)
		if err != nil {
			return nil, err
		}
		bgc, err := MircColor(bg)
		if err != nil {
			return nil, err
		}
		row = append(row, Cell{Value: v, ColorPair: ColorPair{fgc, bgc}})
	}
	if row != nil {
		cells = append(cells, row)
	}
	return &ASCII{cells}, nil
}

func (a *ASCII) Copy() *ASCII {
	newa := &ASCII{}
	for r, row := range a.Cells {
		newa.Cells = append(newa.Cells, make([]Cell, len(row)))
		for c, col := range row {
			newa.Cells[r][c] = col
		}
	}
	return newa
}

func (a *ASCII) Colors() (ret []ColorExtent) {
	off := 0
	for _, row := range a.Cells {
		if len(row) == 0 {
			off++
			continue
		}
		ce := ColorExtent{
			Start:     off,
			ColorPair: row[0].ColorPair,
			CharAttr:  DefaultCharAttr,
		}
		for _, col := range row {
			if ce.ColorPair != col.ColorPair || ce.CharAttr != col.CharAttr {
				if ce.Length != 0 {
					ret = append(ret, ce)
				}
				ce = ColorExtent{
					Start:     off,
					Length:    0,
					ColorPair: col.ColorPair,
					CharAttr:  col.CharAttr,
				}
			}
			l := utf8.RuneLen(col.Value)
			off += l
			ce.Length += l
		}
		if ce.Length != 0 {
			ret = append(ret, ce)
		}
		off++
	}
	return ret
}

func (a *ASCII) Get(x, y int) *Cell {
	if y < 0 || y >= len(a.Cells) {
		return nil
	}
	if x < 0 || x >= len(a.Cells[y]) {
		return nil
	}
	return &a.Cells[y][x]
}

func (a *ASCII) PutString(s string, x, y int) {
	for i, l := range strings.Split(s, "\n") {
		j := 0
		for _, v := range l {
			if c := a.Get(x+j, y+i); c != nil {
				c.Value = v
			} else {
				a.Put(Cell{Value: v}, x+j, y+i)
			}
			j++
		}
	}
}

func (a *ASCII) Put(c Cell, x, y int) *Cell {
	if x < 0 || y < 0 {
		return nil
	}
	if y >= len(a.Cells) {
		for y >= len(a.Cells) {
			a.Cells = append(a.Cells, nil)
		}
	}
	if x >= len(a.Cells[y]) {
		for x >= len(a.Cells[y]) {
			a.Cells[y] = append(a.Cells[y], Cell{Value: ' '})
		}
	}
	a.Cells[y][x] = c
	return &a.Cells[y][x]
}

func (a *ASCII) MergePut(c Cell, x, y int) {
	old := a.Get(x, y)
	if old == nil {
		a.Put(c, x, y)
		return
	}
	if c.Value == ' ' {
		c.Value, c.CharAttr = old.Value, old.CharAttr
	}
	if c.Foreground == nil {
		c.Foreground = old.Foreground
	}
	if c.Background == nil {
		c.Background = old.Background
	}
	a.Put(c, x, y)
}

func (a *ASCII) Columns() (w int) {
	for _, row := range a.Cells {
		if rw := len(row); rw > w {
			w = rw
		}
	}
	return w
}

func (a *ASCII) Rows() int { return len(a.Cells) }

func (a *ASCII) Text() string {
	s := ""
	for r, row := range a.Cells {
		for _, col := range row {
			s = s + string(col.Value)
		}
		if r != len(a.Cells)-1 {
			s = s + "\n"
		}
	}
	return s
}

func (a *ASCII) Bytes() []byte {
	txt, inserts := []byte(a.Text()), 0
	lastAttr := DefaultCharAttr
	for _, ce := range a.Colors() {
		var code []byte
		if ce.CharAttr != lastAttr && ce.ColorPair != DefaultColorPair {
			code = append(code, '\x0f')
		}
		if ce.Bold && lastAttr.Bold != ce.Bold {
			code = append(code, '\x02')
		}
		if ce.Italic && lastAttr.Italic != ce.Italic {
			code = append(code, '\x1d')
		}
		if ce.Underline && lastAttr.Underline != ce.Underline {
			code = append(code, '\x1f')
		}
		lastAttr = ce.CharAttr

		code = append(code, ce.MircCode(&mircPalette)...)
		codelen := len(code)
		newtxt := append(
			txt[:ce.Start+inserts],
			append(code, txt[ce.Start+inserts:]...)...)
		inserts += codelen
		txt = newtxt
	}
	return txt
}

func (a *ASCII) AnsiBytes() []byte {
	txt, inserts := []byte(a.Text()), 0
	for _, ce := range a.Colors() {
		var code []byte
		code = append(code, []byte("\u001b[0m")...)
		if ce.Bold {
			code = append(code, []byte("\u001b[1m")...)
		}
		if ce.Italic {
			code = append(code, []byte("\u001b[3m")...)
		}
		if ce.Underline {
			code = append(code, []byte("\u001b[4m")...)
		}
		code = append(code, ce.AnsiCode(&ansiPalette)...)
		codelen := len(code)
		newtxt := append(
			txt[:ce.Start+inserts],
			append(code, txt[ce.Start+inserts:]...)...)
		inserts += codelen
		txt = newtxt
	}
	return append(txt, []byte("\u001b[0m")...)
}

func (a *ASCII) Paste(aa *ASCII, pt image.Point) {
	for i := 0; i < aa.Columns(); i++ {
		for j := 0; j < aa.Rows(); j++ {
			if c := aa.Get(i, j); c != nil {
				a.MergePut(*c, pt.X+i, pt.Y+j)
			}
		}
	}
}

// PutTrimASCII puts with transparent spaces up to the first non-space on a row,
// then uses all overwriting spaces.
func (a *ASCII) PutTrimASCII(aa *ASCII, pt image.Point) {
	for j := 0; j < aa.Rows(); j++ {
		pastws := false
		for i := 0; i < aa.Columns(); i++ {
			c := aa.Get(i, j)
			if c == nil {
				break
			}
			if c.Value != ' ' {
				pastws = true
			}
			if pastws {
				a.Put(*c, pt.X+i, pt.Y+j)
			}
		}
	}
}

func (a *ASCII) Clip(r image.Rectangle) {
	aa, _ := NewASCII("")
	for i := r.Min.X; i < r.Max.X; i++ {
		for j := r.Min.Y; j < r.Max.Y; j++ {
			if c := a.Get(i, j); c != nil {
				aa.Put(*c, i, j)
			}
		}
	}
	a.Cells = aa.Cells
}

func (a *ASCII) Mirror() {
	aa, _ := NewASCII("")
	for i := 0; i < a.Columns(); i++ {
		for j := 0; j < a.Rows(); j++ {
			if c := a.Get(i, j); c != nil {
				aa.Put(*c, (a.Columns()-1)-i, j)
			}
		}
	}
	a.Cells = aa.Cells
}

func (a *ASCII) Flip() {
	aa, _ := NewASCII("")
	for i := 0; i < a.Columns(); i++ {
		for j := 0; j < a.Rows(); j++ {
			if c := a.Get(i, j); c != nil {
				aa.Put(*c, i, a.Rows()-1-j)
			}
		}
	}
	a.Cells = aa.Cells
}

func (a *ASCII) Scale(x, y int) {
	aa, _ := NewASCII("")
	for i := 0; i < a.Columns()/x; i++ {
		for j := 0; j < a.Rows()/y; j++ {
			if c := a.Get(x*i, y*j); c != nil {
				aa.Put(*c, i, j)
			}
		}
	}
	a.Cells = aa.Cells
}

func (a *ASCII) Rotate(degrees int) {
	panic("stub")
}

func (a *ASCII) Rectangle() image.Rectangle {
	return image.Rect(0, 0, a.Columns(), a.Rows())
}
