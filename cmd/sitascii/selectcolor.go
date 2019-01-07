package main

import (
	"image/color"

	"github.com/andlabs/ui"
	"github.com/chzchzchz/sitbot/ascii"
)

const SelectBarHeight = 10
const SelectBarY = 10
const PaletteY = SelectBarY + SelectBarHeight
const HueDim = 20

type SelectColor struct {
	*ui.Area
	*ascii.Palette
	ascii.ColorPair
}

func NewSelectColor(p *ascii.Palette) *SelectColor {
	sc := &SelectColor{Palette: p}
	sc.Area = ui.NewArea(sc)
	if sc.Area == nil {
		return nil
	}
	return sc
}

func drawRect(dp *ui.AreaDrawParams, c color.Color, x, y, w, h float64) {
	if c == nil {
		return
	}
	p := ui.DrawNewPath(ui.DrawFillModeWinding)
	p.AddRectangle(x, y, w, h)
	p.End()
	r, g, b, a := color2float64(c)
	db := &ui.DrawBrush{R: r, G: g, B: b, A: a}
	dp.Context.Fill(p, db)
	p.Free()
}

func (ah *SelectColor) Draw(a *ui.Area, dp *ui.AreaDrawParams) {
	drawRect(dp, ah.Foreground, 0, SelectBarY, dp.AreaWidth/2, SelectBarHeight)
	drawRect(dp, ah.Background, dp.AreaWidth/2, SelectBarY, dp.AreaWidth/2, SelectBarHeight)
	for hidx, hue := range ah.Hues {
		for i, c := range hue {
			drawRect(dp, c,
				float64(i*HueDim), float64(PaletteY+hidx*HueDim),
				HueDim, HueDim)
		}
	}
}

func (ah *SelectColor) MouseEvent(a *ui.Area, me *ui.AreaMouseEvent) {
	if me.Down == 0 {
		return
	}
	// Drop color.
	if len(me.Held) == 1 && me.Held[0] == 2 {
		if me.Down == 1 {
			ah.Foreground = nil
		} else if me.Down == 3 {
			ah.Background = nil
		}
		ah.QueueRedrawAll()
		return
	}
	// Resolve color.
	x, y := int(me.X/HueDim), int((me.Y-PaletteY)/HueDim)
	if y >= len(ah.Hues) {
		return
	}
	if x >= len(ah.Hues[y]) {
		return
	}
	c := ah.Hues[y][x]
	// Set color.
	if me.Down == 1 {
		// fg
		ah.Foreground = c
	} else if me.Down == 3 {
		// bg
		ah.Background = c
	}
	ah.QueueRedrawAll()
}

func (ah *SelectColor) MouseCrossed(a *ui.Area, left bool) {}
func (ah *SelectColor) DragBroken(a *ui.Area)              {}
func (ah *SelectColor) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) bool {
	return false
}
