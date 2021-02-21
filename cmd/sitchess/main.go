package main

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/chzchzchz/sitbot/ascii"
)

var rootCmd = &cobra.Command{
	Use:   "sitchess",
	Short: "Single shot chess",
}
var fromUser string

func init() {
	rootCmd.PersistentFlags().StringVarP(&fromUser, "from", "f", "nobody", "User issuing command")
	startCmd := &cobra.Command{
		Use:   "start opponent",
		Short: "start game",
		Run: func(cmd *cobra.Command, args []string) {
			s := strings.Map(func(r rune) rune {
				if unicode.IsSymbol(r) {
					return -1
				}
				return r
			}, args[0])
			if s != args[0] {
				fmt.Println("wut")
				return
			}
			start(fromUser, s)
		},
	}
	resignCmd := &cobra.Command{
		Use:   "resign",
		Short: "resign game",
		Run:   func(cmd *cobra.Command, args []string) { resign(fromUser) },
	}
	showCmd := &cobra.Command{
		Use:   "show user",
		Short: "show user's game",
		Run:   func(cmd *cobra.Command, args []string) { show(fromUser) },
	}
	moveCmd := &cobra.Command{
		Use:   "move pos-old pos-new",
		Short: "move piece",
		Run:   func(cmd *cobra.Command, args []string) { move(fromUser, args[0], args[1]) },
	}
	hintCmd := &cobra.Command{
		Use:   "hint",
		Short: "hint",
		Run:   func(cmd *cobra.Command, args []string) { hint(fromUser) },
	}
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(resignCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(hintCmd)
}

var cellWidth = 3
var xOffset = 3

func Board2Ascii(b *Board) *ascii.ASCII {
	a, _ := ascii.NewASCII("")
	cw, _ := ascii.MircColor(0)
	cb, _ := ascii.MircColor(1)

	cbrown, _ := ascii.MircColor(5)
	corange, _ := ascii.MircColor(7)
	coord2sqcolor := func(x, y int) color.Color {
		if ((x & 1) ^ (y & 1)) == 0 {
			return corange
		}
		return cbrown
	}
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			c := ascii.Cell{
				ColorPair: ascii.ColorPair{cb, coord2sqcolor(x, y)},
				Value:     ' ',
				CharAttr:  ascii.CharAttr{Bold: true},
			}
			for j := 0; j < cellWidth; j++ {
				a.Put(c, (cellWidth*x)+j+xOffset, y+1)
			}
		}
	}
	for x := 0; x < 8; x++ {
		c := ascii.Cell{
			ColorPair: ascii.ColorPair{nil, nil},
			Value:     rune('a' + x),
		}
		a.Put(c, (cellWidth*x)+(cellWidth/2)+xOffset, 0)
		a.Put(c, (cellWidth*x)+(cellWidth/2)+xOffset, 9)
	}
	for y := 0; y < 8; y++ {
		v := fmt.Sprintf("%d", 8-y)
		c := ascii.Cell{ColorPair: ascii.ColorPair{nil, nil}, Value: rune(v[0])}
		a.Put(c, xOffset-2, y+1)
		a.Put(c, cellWidth*8+xOffset+cellWidth/2, y+1)
	}
	// Apply cells.
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if p := b.Get(x, y); p != nil {
				pc := cw
				if p.player == Black {
					pc = cb
				}
				c := ascii.Cell{
					ColorPair: ascii.ColorPair{pc, coord2sqcolor(x, y)},
					Value:     p.Rune(),
					CharAttr:  ascii.CharAttr{Bold: true},
				}
				a.Put(c, (cellWidth*x)+xOffset+(cellWidth/2), y+1)
			}
		}
	}
	return a
}

func PrintBoardStatus(b *Board) bool {
	a := Board2Ascii(b)
	os.Stdout.Write(a.Bytes())

	wc, bc := b.Count(White), b.Count(Black)
	c := 0
	for _, v := range wc {
		c += v
	}
	for _, v := range bc {
		c += v
	}
	if c == 32 {
		// No pieces to show captured.
		return false
	}

	fmt.Printf("\n\x030,1")
	for i, v := range wc {
		for j := 0; j < pieceCounts[i]-v; j++ {
			fmt.Printf("%s", string(PlayerPiece{White, Piece(i)}.Rune()))
		}
		for j := 0; j < v; j++ {
			fmt.Printf(" ")
		}
	}
	fmt.Printf("\x031,0")
	for i, v := range bc {
		for j := 0; j < pieceCounts[i]-v; j++ {
			fmt.Printf("%s", string(PlayerPiece{Black, Piece(i)}.Rune()))
		}
		for j := 0; j < v; j++ {
			fmt.Printf(" ")
		}
	}
	fmt.Println("")
	if wc[King] == 0 {
		fmt.Println("black wins!")
		return true
	} else if bc[King] == 0 {
		fmt.Println("white wins!")
		return true
	}
	return false
}

func SaveBoard(b *Board, u string) error {
	ufname := path.Join("games", path.Base(u))
	return os.WriteFile(ufname, []byte(b.Fen()+"\n"), 0644)
}

func LoadBoard(u string) (*Board, error) {
	ufname := path.Join("games", path.Base(u))
	b, err := os.ReadFile(ufname)
	if err != nil {
		return nil, err
	}
	return NewBoard(string(b))
}

func gamePair(u string) (string, string, string) {
	ufname := path.Join("games", path.Base(u))
	s, err := os.Readlink(ufname)
	if err != nil {
		return "", "", ""
	}
	t := strings.Split(path.Base(s), ":")
	return t[0], t[1], s
}

func DeleteBoard(u string) error {
	a, b, s := gamePair(u)
	if s == "" {
		return io.EOF
	}
	os.Remove(path.Join("games", a))
	os.Remove(path.Join("games", b))
	os.Remove(path.Join("games", s))
	return nil
}

func start(a, b string) {
	afname := path.Join("games", path.Base(a))
	if _, err := os.Stat(afname); err == nil {
		fmt.Printf("game already running for %q, maybe resign?\n", a)
		return
	}
	bfname := path.Join("games", path.Base(b))
	if _, err := os.Stat(bfname); err == nil {
		fmt.Printf("game already running for %q, maybe resign?\n", b)
		return
	}
	fpart := path.Base(a) + ":" + path.Base(b)
	fname := path.Join("games", fpart)
	if _, err := os.Stat(fname); err == nil {
		fmt.Printf("game already running for %q, maybe resign?\n", a)
		return
	}

	board, _ := NewBoard(freshFen)

	f, err := os.OpenFile(fname, os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	f.Close()
	if err = os.Symlink(fpart, afname); err != nil {
		panic(err)
	}
	if afname != bfname {
		if err = os.Symlink(fpart, bfname); err != nil {
			panic(err)
		}
	}
	SaveBoard(board, a)

	show(a)
}

func resign(u string) {
	_, _, s := gamePair(u)
	if err := DeleteBoard(u); err != nil {
		fmt.Println("no game to resign")
		return
	}
	fmt.Println("resigned game", s)
}

func move(u, a, b string) {
	board, err := LoadBoard(u)
	if err != nil {
		fmt.Println("no game running")
		return
	}
	u1, u2, _ := gamePair(u)
	if u1 != u2 {
		if u == u1 && board.next != White {
			fmt.Println("wait for", u2)
			return
		} else if u == u2 && board.next != Black {
			fmt.Println("wait for", u1)
			return
		}
	}
	x0, y0 := Str2Coord(a)
	x1, y1 := Str2Coord(b)
	if !board.Move(x0, y0, x1, y1) {
		fmt.Println("bad move")
		return
	}
	SaveBoard(board, u)
	show(u)
}

func show(u string) {
	b, err := LoadBoard(u)
	if err != nil {
		fmt.Println("no game to show")
		return
	}
	if PrintBoardStatus(b) {
		DeleteBoard(u)
	}
}

func hint(u string) {
	fname := path.Join("games", path.Base(u))
	if _, err := os.Stat(fname); err != nil {
		fmt.Println("no game to hint")
	}
	fmt.Println("thinking...")
	s, err := GetMove(fname)
	if err != nil {
		fmt.Println("oops: ", err)
		return
	}
	fmt.Println("ok try", s)
}

func main() {
	rootCmd.Execute()
}
