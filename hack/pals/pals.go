package main

import (
	"flag"
	"image"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
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

// Default pal if no paldir.
var pals = []string{
	`╭──╮
o_o│
╰╮ │
`,
	`╭──╮
│o_o
│ ╭╯
`,
}

var floor = "─"

func mustAscii(s string) *ascii.ASCII {
	a, err := ascii.NewASCII(s)
	if err != nil {
		panic(err)
	}
	return a
}

func loadCaps(dir string) (ret []*ascii.ASCII) {
	if dir == "" {
		return []*ascii.ASCII{mustAscii(pals[0]), mustAscii(pals[1])}
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	names := []string{}
	for _, f := range files {
		if !f.IsDir() {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			panic(err)
		}
		ret = append(ret, mustAscii(string(b)))
	}
	return ret
}

func main() {
	paldir := ""
	fill := ""
	flag.StringVar(&paldir, "paldir", "", "directory of pal caps")
	flag.StringVar(&fill, "fill", "", "fill string")
	flag.Parse()
	caps := loadCaps(paldir)

	rand.Seed(time.Now().UnixNano())
	a, _ := ascii.NewASCII("")
	x, y := 70, 14
	rate := 4
	for i := 0; i < x; i++ {
		if rand.Intn(rate) != 0 {
			a.PutString(floor, i, y)
			continue
		}
		n := rand.Intn(2)
		if rand.Intn(12) == 0 {
			n = rand.Intn(len(caps))
		}
		palcap := caps[n]
		ph := palcap.Rows()

		stddev := float64(y-2) / 3.0
		mu := float64(y / 4.0)
		neckHeight := int(rand.NormFloat64()*stddev + mu)
		if neckHeight >= y-(ph-1) {
			neckHeight = y - ph
		} else if neckHeight < 0 {
			neckHeight = 0
		}
		a.Paste(palcap, image.Point{i, y - ph - neckHeight})
		for j := 0; j <= neckHeight; j++ {
			a.PutString(necks[n%2][1:], i, (y - neckHeight + j))
		}
		a.PutString(shoulder[n%2], i, y)
		i += 3
	}

	if fill != "" {
		sidx := 0
		for j := 0; j < y; j++ {
			for i := 0; i < a.Columns(); i++ {
				v := fill[sidx%len(fill)]
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
