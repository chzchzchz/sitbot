package main

import (
	"fmt"
	"image"
	"math/rand"
	"os"
	"time"

	"github.com/chzchzchz/sitbot/ascii"
)

func putCenterString(a *ascii.ASCII, s string) {
	a.PutString(s, a.Columns()/2-len(s)/2, 0)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	w := 60
	balls := make([]int, 10)
	a, _ := ascii.NewASCII("")
	for i := 0; i < len(balls); i++ {
		balls[i] = 2 + i*(w/len(balls))
		a.PutString("(o)", balls[i]-1, 0)
		a.PutString(".", balls[i], 1)
	}
	h, y := 30, 2
	for y < h {
		pegs, pm := make([]int, rand.Intn(w)), make(map[int]struct{})
		for i := range pegs {
			pegs[i] = rand.Intn(w)
			for j := range pegs[:i] {
				if pegs[i] == pegs[j] {
					pegs[i] = rand.Intn(w)
				}
			}
		}
		for _, p := range pegs {
			pm[p] = struct{}{}
			a.PutString("x", p, y)
		}
		collides, rounds := true, 0
		for collides && rounds < 1000 {
			rounds++
			collides = false
			for bi, b := range balls {
				// Collision?
				if _, ok := pm[b]; ok {
					newv := b
					if rand.Intn(2) == 0 {
						newv += 1
					} else {
						newv -= 1
					}
					if newv < 0 {
						newv = 0
					} else if newv >= w {
						newv = w
					}
					bad := false
					for _, bb := range balls {
						if bad = (bb == newv); bad {
							break
						}
					}
					if !bad {
						balls[bi] = newv
					}
					collides = true
				}
			}
		}
		for _, b := range balls {
			a.PutString(".", b, y)
			a.PutString(".", b, y+1)
		}
		y += 2
	}

	points := 0
	for i := 0; i < len(balls); i++ {
		off, match := 1+i*(w/len(balls)), false
		for _, b := range balls {
			if b == off {
				match = true
				break
			}
		}
		if match {
			points++
			a.PutString("(.)", off-1, y)
		} else {
			a.PutString("(_)", off-1, y)
		}
	}

	hdr, _ := ascii.NewASCII("")
	hdr.PutString(" ", w, 0)
	putCenterString(hdr, "WELCOME TO PACHINKO")
	os.Stdout.Write(hdr.Bytes())
	fmt.Println()

	aa := ascii.RoundBox(image.Pt(a.Columns()+2, a.Rows()+2))
	aa.Paste(a, image.Pt(1, 1))

	os.Stdout.Write(aa.Bytes())
	fmt.Println()

	footer, _ := ascii.NewASCII("")
	footer.PutString(" ", w, 0)
	fs := ""
	if points >= 2 {
		fs = fmt.Sprintf("YOU SCORED: %d POINTS!!!", points)
	} else if points == 1 {
		fs = "NICE! ONE POINT!!!"
	} else {
		fs = fmt.Sprintf("YOU LOSE")
	}
	putCenterString(footer, fs)
	os.Stdout.Write(footer.Bytes())
	fmt.Println()
}
