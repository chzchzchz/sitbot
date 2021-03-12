//go:generate peg script.peg
package main

import (
	"fmt"
	"image"
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
