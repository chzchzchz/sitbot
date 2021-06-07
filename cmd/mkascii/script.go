//go:generate peg script.peg
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/chzchzchz/sitbot/ascii"
)

type Script struct {
	groups map[string]*Group
	stmts  []StmtFunc
}

type StmtFunc func(*ascii.ASCII) error

type Group struct {
	id    string
	stmts []StmtFunc
}

func run(stmts []StmtFunc) (*ascii.ASCII, error) {
	a, _ := ascii.NewASCII("")
	for _, stmt := range stmts {
		if err := stmt(a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (g *Grammar) num(text string) {
	var v int
	fmt.Sscanf(text, "%d", &v)
	g.nums = append(g.nums, v)
}

func (g *Grammar) popNum() (v int) {
	v, g.nums = g.nums[len(g.nums)-1], g.nums[:len(g.nums)-1]
	return v
}

func (g *Grammar) popCoord() (v image.Point) {
	v, g.coords = g.coords[len(g.coords)-1], g.coords[:len(g.coords)-1]
	return v
}

func (g *Grammar) color() ascii.ColorPair {
	fg, _ := ascii.MircColor(g.fgc)
	bg, _ := ascii.MircColor(g.bgc)
	return ascii.ColorPair{fg, bg}
}

func (s *Script) loadASCII(id string) (*ascii.ASCII, error) {
	if grp, ok := s.groups[id]; ok {
		return run(grp.stmts)
	}
	bytes, err := os.ReadFile(id)
	if err != nil {
		return nil, err
	}
	return ascii.NewASCII(string(bytes))
}

func (g *Grammar) put() StmtFunc {
	id, coord, s := g.id, g.popCoord(), g.script
	return func(a *ascii.ASCII) (err error) {
		aa, err := s.loadASCII(id)
		if err != nil {
			return err
		}
		a.Paste(aa, coord)
		return nil
	}
}

func (g *Grammar) cput() StmtFunc {
	id, coord, s := g.id, g.popCoord(), g.script
	return func(a *ascii.ASCII) (err error) {
		aa, err := s.loadASCII(id)
		if err != nil {
			return err
		}
		c := image.Pt(coord.X-aa.Columns()/2, coord.Y-aa.Rows()/2)
		a.Paste(aa, c)
		return nil
	}
}

func (g *Grammar) box() StmtFunc {
	r, c := g.rectangle, ascii.Cell{ColorPair: g.color(), Value: ' '}
	return func(a *ascii.ASCII) error {
		a.Box(r, c)
		return nil
	}
}

func eqColors(a, b color.Color) bool {
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

func apply(a *ascii.ASCII, r image.Rectangle, f func(cell *ascii.Cell)) {
	if r == image.Rect(-1, -1, -1, -1) {
		r = image.Rect(0, 0, a.Columns(), a.Rows())
	}
	for i := r.Min.X; i < a.Columns() && i < r.Max.X; i++ {
		for j := r.Min.Y; j < a.Rows() && j < r.Max.Y; j++ {
			if c := a.Get(i, j); c != nil {
				f(c)
			}
		}
	}
}

func (g *Grammar) bbox() StmtFunc {
	r, c, hard := g.rectangle, g.color(), g.hard
	setBg := func(cell *ascii.Cell) {
		if !hard && cell.Foreground == nil && cell.Background == nil {
			return
		} else if cell.Foreground == nil {
			cell.Background = c.Foreground
		} else if eqColors(cell.Foreground, c.Foreground) {
			cell.Background = c.Foreground
		} else {
			cell.Background = c.Foreground
		}
	}
	return func(a *ascii.ASCII) error {
		apply(a, r, setBg)
		return nil
	}
}

func (g *Grammar) fbox() StmtFunc {
	r, c, hard := g.rectangle, g.color(), g.hard
	setFg := func(cell *ascii.Cell) {
		if !hard && cell.Foreground == nil && cell.Background == nil {
			return
		} else if cell.Background == nil {
			cell.Foreground = c.Foreground
		} else if eqColors(cell.Background, c.Foreground) {
			cell.Foreground = c.Background
		} else {
			cell.Foreground = c.Foreground
		}
	}
	return func(a *ascii.ASCII) error {
		apply(a, r, setFg)
		return nil
	}
}

func (g *Grammar) scale() StmtFunc {
	c := g.popCoord()
	return func(a *ascii.ASCII) error {
		a.Scale(c.X, c.Y)
		return nil
	}
}

func (g *Grammar) clip() StmtFunc {
	r := g.rectangle
	return func(a *ascii.ASCII) error {
		if r.Max.X == 0 {
			r.Max.X = a.Columns()
		}
		if r.Max.Y == 0 {
			r.Max.Y = a.Rows()
		}
		a.Clip(r)
		return nil
	}
}

func (g *Grammar) flip() StmtFunc {
	return func(a *ascii.ASCII) error {
		a.Flip()
		return nil
	}
}

func (g *Grammar) mirror() StmtFunc {
	return func(a *ascii.ASCII) error {
		a.Mirror()
		return nil
	}
}

func (g *Grammar) rotate() StmtFunc {
	n := g.popNum()
	return func(a *ascii.ASCII) error {
		a.Rotate(n)
		return nil
	}
}

func (g *Grammar) clearText() StmtFunc {
	return func(a *ascii.ASCII) error {
		a.ClearText()
		return nil
	}
}

func (g *Grammar) rect() {
	br, tl := g.popCoord(), g.popCoord()
	g.rectangle = image.Rect(br.X, br.Y, tl.X, tl.Y)
}

func (g *Grammar) xy() {
	y, x := g.popNum(), g.popNum()
	g.coords = append(g.coords, image.Pt(x, y))
}

func (g *Grammar) addStmt(s StmtFunc) {
	g.script.stmts = append(g.script.stmts, s)
}

func (g *Grammar) openGroup() {
	g.group, g.script.stmts = &Group{id: g.id, stmts: g.script.stmts}, nil
}

func (g *Grammar) closeGroup() {
	g.group.stmts, g.script.stmts = g.script.stmts, g.group.stmts
	g.script.groups[g.group.id] = g.group
}
