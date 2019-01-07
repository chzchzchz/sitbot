package main

import (
	"image/color"

	"github.com/andlabs/ui"
	"github.com/chzchzchz/sitbot/ascii"
)

func color2float64(c color.Color) (r, g, b, a float64) {
	rc, gc, bc, ac := c.RGBA()
	return float64(rc) / 0xffff,
		float64(gc) / 0xffff,
		float64(bc) / 0xffff,
		float64(ac) / 0xffff
}

func color2fg(c color.Color) ui.TextColor {
	r, g, b, a := color2float64(c)
	return ui.TextColor{R: r, G: g, B: b, A: a}
}

func color2bg(c color.Color) ui.TextBackground {
	r, g, b, a := color2float64(c)
	return ui.TextBackground{R: r, G: g, B: b, A: a}
}

func AttributedStringFromASCII(a *ascii.ASCII) *ui.AttributedString {
	as := ui.NewAttributedString(a.Text())
	for _, ce := range a.Colors() {
		if ce.Foreground != nil {
			as.SetAttribute(color2fg(ce.Foreground), ce.Start, ce.Start+ce.Length)
		}
		if ce.Background != nil {
			as.SetAttribute(color2bg(ce.Background), ce.Start, ce.Start+ce.Length)
		}
	}
	return as
}
