package main

import (
	"image"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/chzchzchz/sitbot/ascii"
)

func mustNewASCII(path string) *ascii.ASCII {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	img, err := ascii.NewASCII(string(b))
	if err != nil {
		panic(err)
	}
	return img
}

func mustLines(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	ss := strings.Split(string(b), "\n")
	for ss[len(ss)-1] == "" {
		ss = ss[:len(ss)-1]
	}
	return ss
}

func makeCard(pal *ascii.ASCII, notice string, val string) *ascii.ASCII {
	out, _ := ascii.NewASCII("")
	y := 0
	out.Paste(pal, image.Pt(1, y))
	y += pal.Rows()
	out.PutString(" "+notice+" ", 0, y)
	y += 1
	boxw := max(len(val)+2, len(notice)+2)
	boxw = max(pal.Columns(), boxw)
	out.Paste(ascii.RoundBox(image.Pt(boxw, 3)), image.Pt(0, y))
	y += 1
	out.PutString(val, 1, y)
	return out
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

var moneys = []rune{'$', '¢', '£', '¤', '¥', '฿', '₠', '₡', '₢', '₣', '₤', '₥', '₦', '₧', '₨', '₩', '₪', '₫', '€', '₭', '₮', '₯', '₰', '₱'}
var moneys2 = []rune{'﹩', '＄', '￠', '￡', '￥', '￦'}

func choosePal() *ascii.ASCII {
	files, err := ioutil.ReadDir("ascii/people/")
	if err != nil {
		panic(err)
	}
	return mustNewASCII("ascii/people/" + files[rand.Intn(len(files))].Name())
}

func main() {
	rand.Seed(time.Now().UnixNano())

	name := os.Args[1]
	hdr := mustNewASCII("ascii/alert.txt")
	items := mustLines("ascii/items.txt")
	many := mustLines("ascii/many.txt")
	youPal := choosePal()
	corn := mustNewASCII("ascii/corn.ascii")
	corn.Scale(2, 2)

	you := " * " + items[rand.Intn(len(items))] + " "
	me := " * " + many[rand.Intn(len(many))] + " "
	out, _ := ascii.NewASCII("")
	out.Paste(hdr, image.Pt(0, 0))

	palCard := makeCard(youPal, name+", trade me your:", you)
	cornCard := makeCard(corn, "and you'll get:", me)

	for y := 0; y < max(palCard.Rows(), cornCard.Rows()); y++ {
		m := ""
		for i := 0; i < hdr.Columns(); i++ {
			//if rand.Intn(2) == 0 || i + 1 == hdr.Columns() {
			m += string(moneys[rand.Intn(len(moneys))])
			//} else {
			//	m += string(moneys2[rand.Intn(len(moneys2))])
			//	m += "\ufeff"
			//	i++
			//}
		}
		out.PutString(m, 0, hdr.Rows()+y)
	}

	cardh := hdr.Rows()
	offMe := hdr.Columns() - cornCard.Columns()

	diffh := (palCard.Rows() - cornCard.Rows())
	diffyou, diffme := 0, 0
	if diffh > 0 {
		diffme = diffh
	} else {
		diffyou = -diffh
	}
	out.PutTrimASCII(palCard, image.Pt(0, cardh+diffyou))
	out.PutTrimASCII(cornCard, image.Pt(offMe, cardh+diffme))

	os.Stdout.Write(out.Bytes())
}
