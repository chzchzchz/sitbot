package main

import (
	"github.com/andlabs/ui"
	"github.com/chzchzchz/sitbot/ascii"
)

type Board struct {
	*ui.Area
	as         *ui.AttributedString
	TextWidth  float64
	TextHeight float64
	Brush      string
	a          *ascii.ASCII
	ctrlDown   bool
}

var uiFont = &ui.FontDescriptor{Family: "Monospace", Size: 6, Weight: 400}

func (b *Board) SetASCII(a *ascii.ASCII) {
	b.a = a
	b.RedrawASCII()
}

func (b *Board) ASCII() *ascii.ASCII { return b.a }

func (b *Board) RedrawASCII() {
	b.as = AttributedStringFromASCII(b.a)
	b.QueueRedrawAll()
}

func NewBoard() *Board {
	aa, _ := ascii.NewASCII(nil)
	b := &Board{a: aa, TextWidth: 1.0, TextHeight: 1.0}
	b.as = AttributedStringFromASCII(b.a)
	b.Area = ui.NewArea(&areaHandler{b})
	if b.Area == nil {
		return nil
	}
	return b
}

type areaHandler struct{ b *Board }

func (ah *areaHandler) Draw(a *ui.Area, dp *ui.AreaDrawParams) {
	tl := ui.DrawNewTextLayout(&ui.DrawTextLayoutParams{
		String:      ah.b.as,
		DefaultFont: uiFont,
		Width:       dp.AreaWidth,
		Align:       ui.DrawTextAlign(0),
	})
	defer tl.Free()

	p := ui.DrawNewPath(ui.DrawFillModeWinding)
	p.AddRectangle(0, 0, dp.AreaWidth, dp.AreaHeight)
	p.End()
	defer p.Free() // when done with the path
	db := &ui.DrawBrush{A: 1.0}
	dsp := &ui.DrawStrokeParams{Thickness: 5.0}
	dp.Context.Stroke(p, db, dsp)
	dp.Context.Text(tl, 2, 2)
	ah.b.TextWidth, ah.b.TextHeight = tl.Extents()
}

func (ah *areaHandler) MouseEvent(a *ui.Area, me *ui.AreaMouseEvent) {
	if me.Down == 0 {
		return
	}
	if len(me.Held) == 1 && me.Held[0] == 2 {
		if me.Down == 1 {
			// zoom out (-)
			if uiFont.Size > 2 {
				uiFont.Size -= 2
			}
		} else if me.Down == 3 {
			// zoom in (+)
			if uiFont.Size < 128 {
				uiFont.Size += 2
			}
		}
		ah.b.RedrawASCII()
		return
	}
	tw, th := ah.b.TextWidth, ah.b.TextHeight
	if tw <= 0 {
		tw = 1
	}
	if th <= 0 {
		th = 1
	}
	x, y := me.X/tw, me.Y/th
	x, y = x*float64(ah.b.a.Columns()), y*float64(ah.b.a.Rows())

	// Modify cell.
	if c := ah.b.a.Get(int(x), int(y)); c != nil {
		if ah.b.ctrlDown {
			SetSelectColor(c.ColorPair)
			return
		}
		if me.Down == 1 {
			if len(ah.b.Brush) > 0 {
				c.Value = byte(ah.b.Brush[0])
			}
			cp := GetSelectColor()
			if cp.Foreground != nil {
				c.Foreground = cp.Foreground
			}
			if cp.Background != nil {
				c.Background = cp.Background
			}
		} else if me.Down == 3 {
			c.Foreground, c.Background = nil, nil
			// TODO(optimize endlines if v == space)
		}
		ah.b.RedrawASCII()
		return
	}
	// Create new cells.
	if me.Down == 1 {
		br := byte(' ')
		if len(ah.b.Brush) > 0 {
			br = byte(ah.b.Brush[0])
		}
		c := GetSelectColor()
		ah.b.a.Put(ascii.Cell{ColorPair: c, Value: br}, int(x), int(y))
		ah.b.RedrawASCII()
		return
	}
}

func (ah *areaHandler) MouseCrossed(a *ui.Area, left bool) {}
func (ah *areaHandler) DragBroken(a *ui.Area)              {}
func (ah *areaHandler) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) bool {
	if ke.Key == 0 && ke.Modifier == 1 && ke.Modifiers == 0 {
		ah.b.ctrlDown = ke.Up == false
		return true
	}
	return false
}
