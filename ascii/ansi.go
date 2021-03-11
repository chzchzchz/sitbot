package ascii

import "image/color"

var ansiCodes = color.Palette{
	// Black 1
	color.RGBA{0, 0, 0, 255},
	// Brown 5
	color.RGBA{127, 0, 0, 255},
	// Green 3
	color.RGBA{0, 147, 0, 255},
	// Orange 7
	color.RGBA{252, 127, 0, 255},
	// Blue 2
	color.RGBA{0, 0, 127, 255},
	// Purple 6
	color.RGBA{156, 0, 156, 255},
	// Cyan 10
	color.RGBA{0, 147, 147, 255},
	// Light Grey 15
	color.RGBA{210, 210, 210, 255},

	// Grey 14
	color.RGBA{127, 127, 127, 255},
	// Light red 4
	color.RGBA{255, 0, 0, 255},
	// Light Green 9
	color.RGBA{0, 252, 0, 255},
	// Yellow 8
	color.RGBA{255, 255, 0, 255},
	// Light Blue 12
	color.RGBA{0, 0, 252, 255},
	// Magenta 13
	color.RGBA{255, 0, 255, 255},
	// Light Cyan 11
	color.RGBA{0, 255, 255, 255},
	// White 0
	color.RGBA{255, 255, 255, 255},
}

var ansiPalette = Palette{ansiCodes, nil}

func lookupAnsiIndex(c color.Color) int {
	idx := ansiCodes.Index(c)
	if !colorsEqual(c, ansiCodes[idx]) {
		return -1
	}
	return idx
}
