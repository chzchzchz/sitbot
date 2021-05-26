package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/chzchzchz/sitbot/ascii"
)

var shoulder = []string{
	`─╯ ╰`,
	`╯ ╰─`,
}

var necks = []string{
	`
 │ │`,
	`
│ │ `}

var pals = []string{
	`
╭──╮
o_o│
╰╮ │

`,
	`
╭──╮
│o_o
│ ╭╯`,
}

var pals2 = []string{
	`
╭──╮
O O│
╰o │
`,
	`
╭──╮
│O O
│ o╯`,
}

var pals3 = []string{
	`
╭──╮
o o│
│- │
╰╮ │`,
	`
╭──╮
│o o
│ -│
│ ╭╯`,
}

var floor = "─"

func main() {
	rand.Seed(time.Now().UnixNano())
	a, _ := ascii.NewASCII("")
	x, y := 70, 14
	rate := 4
	for i := 0; i < x; i++ {
		if rand.Intn(rate) != 0 {
			a.PutString(floor, i, y)
			continue
		}
		//		neckHeight := rand.Intn(y-2) - 1
		stddev := float64(y-2) / 3.0
		mu := float64(y / 4.0)
		neckHeight := int(rand.NormFloat64()*stddev + mu)
		if neckHeight >= y-3 {
			neckHeight = y - 4
		} else if neckHeight < 0 {
			neckHeight = 0
		}
		p := rand.Intn(2)
		if rand.Intn(12) != 0 {
			a.PutString(pals[p][1:], i, y-4-neckHeight)
		} else if neckHeight <= y-5 && rand.Intn(12) <= 6 {
			a.PutString(pals3[p][1:], i, y-5-neckHeight)
		} else {
			a.PutString(pals2[p][1:], i, y-4-neckHeight)
		}
		for j := 0; j <= neckHeight; j++ {
			a.PutString(necks[p][1:], i, (y - neckHeight + j - 1))
		}
		a.PutString(shoulder[p], i, y)
		i += 3
	}

	if len(os.Args) > 1 {
		sidx := 0
		for j := 0; j < y; j++ {
			for i := 0; i < x; i++ {
				v := os.Args[1][sidx%len(os.Args[1])]
				if cc := a.Get(i, j); cc != nil {
					if cc.Value == ' ' {
						cc.Value = rune(v)
						sidx++
					}
				} else {
					a.Put(ascii.Cell{Value: rune(v)}, i, j)
					sidx++
				}
			}
		}
	}
	os.Stdout.Write(a.Bytes())
}
