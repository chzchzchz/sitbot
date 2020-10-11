package ascii

import (
	"errors"
	"unicode/utf8"
)

var ErrBadMircCode = errors.New("bad mirc code")
var ErrBadCellCoord = errors.New("out of range")

type Cell struct {
	ColorPair
	Value rune
}

type ASCII struct {
	// Cells is indexed via [y][x].
	Cells [][]Cell
}

func NewASCII(dat string) (*ASCII, error) {
	var cells [][]Cell
	var row []Cell
	chompState, fg, bg := 0, -1, -1
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
			if v >= '0' && v <= '9' {
				if fg == -1 {
					fg = 0
				}
				fg = fg*10 + int(v-'0')
				continue
			} else if v == ',' {
				chompState, bg = 2, -1
				continue
			} else if fg == -1 {
				return nil, ErrBadMircCode
			}
			chompState = 0
		case 2:
			if v >= '0' && v <= '9' {
				if bg == -1 {
					bg = 0
				}
				bg = bg*10 + int(v-'0')
				continue
			} else if bg == -1 {
				return nil, ErrBadMircCode
			}
			chompState = 0
		}
		fgc, err := lookupColor(fg)
		if err != nil {
			return nil, err
		}
		bgc, err := lookupColor(bg)
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
		ce := ColorExtent{Start: off, ColorPair: row[0].ColorPair}
		for _, col := range row {
			if ce.ColorPair != col.ColorPair {
				if ce.Length != 0 {
					ret = append(ret, ce)
				}
				ce = ColorExtent{
					Start:     off,
					Length:    0,
					ColorPair: col.ColorPair}
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

func (a *ASCII) Put(c Cell, x, y int) {
	if x < 0 || y < 0 {
		return
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
	for _, ce := range a.Colors() {
		code := ce.Code(&mircPalette)
		codelen := len(code)
		newtxt := append(
			txt[:ce.Start+inserts],
			append(code, txt[ce.Start+inserts:]...)...)
		inserts += codelen
		txt = newtxt
	}
	return txt
}
