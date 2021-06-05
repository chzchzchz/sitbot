package main

import (
	"flag"
	"fmt"
	"image"
	"math/rand"
	"os"
	"time"
	"unicode/utf8"

	"github.com/chzchzchz/sitbot/ascii"
)

func putCenterString(a *ascii.ASCII, s string) {
	a.PutString(s, a.Columns()/2-len(s)/2, 0)
}

var pegWideFace = []rune{'^', '▔', '▀'}
var ballFace = []rune{'.', ':', '·'}
var edge = [][]rune{{'◢', '◣'},
	{'▀', '▀'},
	{'▔', '▔'},
	{'▟', '▙'},
	{'▞', '▚'},
	{'▝', '▘'},
	{'▕', '▌'},
	{'▕', '▎'},
}

func randRune(s []rune) rune {
	return s[rand.Intn(len(s))]
}

func placePegs(a *ascii.ASCII, y int) map[int]struct{} {
	w := a.Columns()
	pegs, pm, pegline := make([]int, rand.Intn(w)), make(map[int]struct{}), make([]bool, w)
	for i := range pegs {
		pegs[i] = rand.Intn(w)
		for j := range pegs[:i] {
			if pegs[i] == pegs[j] {
				pegs[i] = rand.Intn(w)
			}
		}
		pm[pegs[i]] = struct{}{}
		pegline[pegs[i]] = true
	}

	if pegline[0] {
		a.PutString(string(randRune(pegWideFace)), 0, y)
	}
	if pegline[len(pegline)-1] {
		a.PutString(string(randRune(pegWideFace)), len(pegline)-1, y)
	}
	for i := 1; i < len(pegline)-1; i++ {
		if !pegline[i] {
			continue
		}
		if pegline[i-1] == pegline[i+1] {
			a.PutString(string(randRune(pegWideFace)), i, y)
		} else {
			ee, s := edge[rand.Intn(len(edge))], ""
			if !pegline[i-1] {
				s = string(ee[0])
			} else {
				s = string(ee[1])
			}
			a.PutString(s, i, y)
		}
	}

	return pm
}

func main() {
	xPegs := flag.String("x", "", "use only x's for pegs")
	flag.Parse()
	if *xPegs != "" {
		r, _ := utf8.DecodeRuneInString(*xPegs)
		pegWideFace, edge = []rune{r}, [][]rune{{r, r}}
	}

	rand.Seed(time.Now().UnixNano())
	w := 5 * 11
	balls := make([]int, w/5)
	a, _ := ascii.NewASCII("")
	for i := 0; i < len(balls); i++ {
		balls[i] = 2 + i*(w/len(balls))
		a.PutString("(o)", balls[i]-1, 0)
		a.PutString(":", balls[i], 1)
	}
	a.PutString(" ", w-1, 0)
	h, y := w/2, 2
	for y < h {
		pm := placePegs(a, y)
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
						newv = w - 1
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
			a.PutString(":", b, y)
			a.PutString(string(randRune(ballFace)), b, y+1)

		}
		y += 2
	}

	points := 0
	for i := 0; i < len(balls); i++ {
		a.PutString(string(randRune(ballFace)), balls[i], y)
	}
	cups := make([]int, len(balls))
	for i := 0; i < len(balls); i++ {
		off := 2 + i*(w/len(balls))
		for _, b := range balls {
			if b >= off-1 && b <= off+1 {
				cups[i]++
				points++
			}
		}
	}
	for i := 0; i < len(balls); i++ {
		off := 2 + i*(w/len(balls))
		switch cups[i] {
		case 1:
			a.PutString(`\./`, off-1, y)
		case 2:
			a.PutString(`\:/`, off-1, y)
		case 3:
			a.PutString(`\∵/`, off-1, y)
		default:
			a.PutString(`\_/`, off-1, y)
		}
	}

	hdr, _ := ascii.NewASCII("")
	hdr.PutString(" ", w, 0)
	putCenterString(hdr, "✿ WELCOME TO PACHINKO2 ✿")
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
