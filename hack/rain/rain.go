package main

import (
	"fmt"
	"image"
	"math/rand"
	"os"
	"time"

	"github.com/chzchzchz/sitbot/ascii"
)

type scene struct {
	sprites []image.Rectangle
}

func (s *scene) overlaps(r image.Rectangle) bool {
	for _, ss := range s.sprites {
		if ss.Overlaps(r) {
			return true
		}
	}
	return false
}

func (s *scene) add(r image.Rectangle) bool {
	if s.overlaps(r) {
		return false
	}
	s.sprites = append(s.sprites, r)
	return true
}

func (s *scene) render(out *ascii.ASCII, img *ascii.ASCII, h int) {
	var newSprites []image.Rectangle
	for _, r := range s.sprites {
		out.Paste(img, r.Min)
		r = r.Add(image.Pt(0, -h))
		if r.Min.Y > -img.Rows() {
			newSprites = append(newSprites, r)
		}
	}
	s.sprites = newSprites
}

func main() {
	if len(os.Args) < 2 {
		panic("no ascii file")
	}
	rand.Seed(time.Now().UnixNano())

	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	img, err := ascii.NewASCII(string(b))
	if err != nil {
		panic(err)
	}

	n := 0xfffffff
	if len(os.Args) == 3 {
		if _, err := fmt.Sscanf(os.Args[2], "%d", &n); err != nil {
			panic(err)
		}
	}

	w, h := 80, (img.Rows()+1)/3
	rate := (3 * w) / img.Columns()
	s := &scene{}
	for j := 0; j < n; j++ {
		out, _ := ascii.NewASCII("")
		for i := 0; i < w-img.Columns(); i++ {
			if rand.Intn(rate) != 0 {
				continue
			}
			maxy := h - 1
			if s.overlaps(img.Rectangle().Add(image.Pt(i, maxy))) {
				// Impossible to place in this column.
				continue
			}
			r := rand.Intn(maxy)
			if s.add(img.Rectangle().Add(image.Pt(i, r))) {
				break
			}
		}
		s.render(out, img, h)
		out.Clip(image.Rect(0, 0, w, h))
		os.Stdout.Write(out.Bytes())
		fmt.Println("")
	}
	out, _ := ascii.NewASCII("")
	s.render(out, img, img.Rows())
	os.Stdout.Write(out.Bytes())
}
