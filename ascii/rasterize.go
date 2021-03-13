package ascii

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"

	"golang.org/x/image/math/fixed"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	// "golang.org/x/image/font/basicfont"
	// "golang.org/x/image/font/inconsolata"
)

var defaultBg = &color.RGBA{0, 0, 0, 255}

//var defaultBg = &color.RGBA{255, 255, 255, 255}

var defaultFg = &color.RGBA{255, 0xbf, 0, 255}

//var defaultFg = &color.RGBA{0, 255, 0, 255}
// var defaultFg = &color.RGBA{0, 0, 0, 255}

type rasterizer struct {
	dst     *image.RGBA
	fw      int
	fh      int
	hPad    int
	wPad    int
	descent int
	face    font.Face
}

func Rasterize(a *ASCII) (*image.RGBA, error) {
	font, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(font, &opentype.FaceOptions{Size: 14, DPI: 72})
	if err != nil {
		return nil, err
	}
	r := newRasterizer(a, face)
	// r := newRasterizer(a, basicfont.Face7x13)
	// r := newRasterizer(a, inconsolata.Regular8x16)

	empty := &Cell{Value: ' '}
	// Writes a transparent line at end for full row count?
	for y := a.Rows(); y >= 0; y-- {
		for x := 0; x < a.Columns(); x++ {
			c := a.Get(x, y)
			if c == nil {
				c = empty
			}
			r.drawBg(c, x, y)
		}
	}
	// Backwards so tails aren't clobered.
	for y := a.Rows() - 1; y >= 0; y-- {
		for x := 0; x < a.Columns(); x++ {
			c := a.Get(x, y)
			if c == nil {
				c = empty
			}
			r.drawGlyph(c, x, y)
		}
	}

	return r.dst, nil
}

func newRasterizer(a *ASCII, f font.Face) *rasterizer {
	b, _, _ := f.GlyphBounds('M')
	w := int(((b.Max.X - b.Min.X) + 64/2) / 64)
	r := &rasterizer{
		fw:      w,
		fh:      int((f.Metrics().Height + 64/2) / 64),
		hPad:    0,
		wPad:    0,
		face:    f,
		descent: int((f.Metrics().Descent + 64/2) / 64),
	}
	r.dst = image.NewRGBA(
		image.Rect(0, 0, (r.fw+2*r.wPad)*a.Columns(), (r.fh+2*r.hPad)*a.Rows()))
	return r
}

func (r *rasterizer) toXY(x, y int) (int, int) {
	return x*(r.fw+2*r.wPad) + r.wPad, y * (r.fh + 2*r.hPad)
}

func (r *rasterizer) drawBg(c *Cell, logicalX, logicalY int) {
	x, y := r.toXY(logicalX, logicalY)
	bgc := c.Background
	if bgc == nil {
		bgc = defaultBg
	}
	box := image.Rect(x, y+r.descent, x+r.fw+2*r.wPad, y-(r.fh+2*r.hPad)+r.descent)
	draw.Draw(r.dst, box, &image.Uniform{bgc}, image.ZP, draw.Src)
}

func (r *rasterizer) drawGlyph(c *Cell, logicalX, logicalY int) {
	if c.Value == ' ' {
		return
	}
	x, y := r.toXY(logicalX, logicalY)
	fgc := c.Foreground
	if fgc == nil {
		fgc = defaultFg
	}
	point := fixed.Point26_6{
		fixed.Int26_6((x + r.wPad) * 64),
		fixed.Int26_6((y + r.hPad) * 64),
	}
	d := &font.Drawer{
		Dst:  r.dst,
		Src:  image.NewUniform(fgc),
		Face: r.face,
		Dot:  point,
	}
	d.DrawString(string(c.Value))
}

func (r *rasterizer) drawCell(c *Cell, logicalX, logicalY int) {
	r.drawBg(c, logicalX, logicalY)
	r.drawGlyph(c, logicalX, logicalY)
}
