package main

import (
	"fmt"
	"os"

	"github.com/chzchzchz/sitbot/ascii"
)

var corners = []string{
	`
│╰
╰─`,
	`
╭─
│╭`,
	`
─╮
╮│`,
	`
╯│
─╯`}

// directions to follow on corners
var dirs = [][2]int{
	{0, -2},
	{2, 0},
	{0, 2},
	{-2, 0},
}

var lines = []string{
	`
││
││`,
	`
──
──`,
}

func main() {
	a, _ := ascii.NewASCII("")
	x, y := 10, 10
	for i := 0; i < 20; i++ {
		a.PutString(corners[i%4][1:], x, y)
		dx, dy := dirs[i%4][0], dirs[i%4][1]
		x, y = x+dx, y+dy
		m := 1
		//		if i % 2 == 1 {
		//			m = 2
		//		}
		for reps := m * (i + 1) / 2; reps > 0; reps-- {
			a.PutString(lines[i%2][1:], x, y)
			x, y = x+dx, y+dy
		}
	}
	for i := 0; i < len(corners); i++ {
		fmt.Println(corners[i][1:])
	}
	fmt.Println("hi")
	os.Stdout.Write(a.Bytes())
	fmt.Println("")
	fmt.Println("hi")
}
