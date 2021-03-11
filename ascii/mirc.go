package ascii

import "image/color"

var mircCodes = color.Palette{
	// White 0
	color.RGBA{255, 255, 255, 255},
	// Black 1
	color.RGBA{0, 0, 0, 255},
	// Blue 2
	color.RGBA{0, 0, 127, 255},
	// Green 3
	color.RGBA{0, 147, 0, 255},
	// Light red 4
	color.RGBA{255, 0, 0, 255},
	// Brown 5
	color.RGBA{127, 0, 0, 255},
	// Purple 6
	color.RGBA{156, 0, 156, 255},
	// Orange 7
	color.RGBA{252, 127, 0, 255},
	// Yellow 8
	color.RGBA{255, 255, 0, 255},
	// Light Green 9
	color.RGBA{0, 252, 0, 255},
	// Cyan 10
	color.RGBA{0, 147, 147, 255},
	// Light Cyan 11
	color.RGBA{0, 255, 255, 255},
	// Light Blue 12
	color.RGBA{0, 0, 252, 255},
	// Magenta 13
	color.RGBA{255, 0, 255, 255},
	// Grey 14
	color.RGBA{127, 127, 127, 255},
	// Light Grey 15
	color.RGBA{210, 210, 210, 255},
}

var mircHues = []color.Palette{
	{
		color.RGBA{255, 255, 255, 255},
		color.RGBA{210, 210, 210, 255},
		color.RGBA{127, 127, 127, 255},
		color.RGBA{0, 0, 0, 255},
	},
	{
		color.RGBA{0, 0, 252, 255},
		color.RGBA{0, 0, 127, 255},
	},
	{
		color.RGBA{0, 252, 0, 255},
		color.RGBA{0, 147, 0, 255},
	},
	{
		color.RGBA{255, 0, 0, 255},
		color.RGBA{127, 0, 0, 255},
	},
	{
		color.RGBA{255, 0, 255, 255},
		color.RGBA{156, 0, 156, 255},
	},
	{
		color.RGBA{0, 255, 255, 255},
		color.RGBA{0, 147, 147, 255},
	},
	{
		color.RGBA{255, 255, 0, 255},
		color.RGBA{252, 127, 0, 255},
	},
}

var mircPalette = Palette{mircCodes, mircHues}

func NewPaletteMIRC() *Palette {
	return &Palette{
		Palette: mircCodes,
		Hues:    mircHues,
	}
}

func MircColor(n int) (color.Color, error) {
	if n < 0 {
		return nil, nil
	} else if n >= len(mircCodes) {
		return nil, ErrBadMircCode
	}
	return mircCodes[n], nil
}

func lookupMircIndex(c color.Color) int {
	idx := mircCodes.Index(c)
	if !colorsEqual(c, mircCodes[idx]) {
		return -1
	}
	return idx
}
