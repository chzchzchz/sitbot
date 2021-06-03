package ascii

import (
	"image"
)

func (a *ASCII) Box(r image.Rectangle, c Cell) {
	for i := r.Min.X; i < r.Max.X; i++ {
		for j := r.Min.Y; j < r.Max.Y; j++ {
			a.Put(c, i, j)
		}
	}
}

var boxCorner = []rune{'╭', '╮', '╰', '╯'}
var boxSide = []rune{'─', '│'}

func RoundBox(wh image.Point) *ASCII {
	w, h := wh.X, wh.Y
	out, _ := NewASCII("")
	for x := 1; x < w-1; x++ {
		out.Put(Cell{Value: boxSide[0]}, x, 0)
		out.Put(Cell{Value: boxSide[0]}, x, h-1)
	}
	for y := 1; y < h-1; y++ {
		out.Put(Cell{Value: boxSide[1]}, 0, y)
		out.Put(Cell{Value: boxSide[1]}, w-1, y)
	}
	out.Put(Cell{Value: boxCorner[0]}, 0, 0)
	out.Put(Cell{Value: boxCorner[1]}, w-1, 0)
	out.Put(Cell{Value: boxCorner[2]}, 0, h-1)
	out.Put(Cell{Value: boxCorner[3]}, w-1, h-1)
	return out
}
