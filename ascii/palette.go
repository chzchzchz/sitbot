package ascii

import (
	"image/color"
)

type Palette struct {
	color.Palette
	Hues []color.Palette
}

func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

func index2byte(n int) byte {
	if n == -1 {
		return ' '
	} else if n >= 0 && n <= 9 {
		return byte(n) + '0'
	}
	return byte(n-10) + 'A'
}
