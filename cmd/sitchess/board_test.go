package main

import (
	"testing"
)

type fromTo struct {
	from string
	to   string
}

func MustBoard(t *testing.T, s string) *Board {
	b, err := NewBoard(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func strMove(board *Board, a, b string) bool {
	x0, y0 := Str2Coord(a)
	x1, y1 := Str2Coord(b)
	return board.Move(x0, y0, x1, y1)
}

func testMoves(t *testing.T, b *Board, tt []fromTo) {
	for i := range tt {
		if !strMove(b, tt[i].from, tt[i].to) {
			t.Fatalf("%d: expected move %s %s", i, tt[i].from, tt[i].to)
		}
	}
}

func TestBishop(t *testing.T) {
	b := MustBoard(t, "rnbqkbnr/pppp1ppp/4p3/8/3P4/2N5/PPP1PPPP/R1BQKBNR b Kqkq d4 0 2")
	if !strMove(b, "f8", "b4") {
		t.Fatal("expected move")
	}
}

func TestEnPassant(t *testing.T) {
	b := MustBoard(t, freshFen)
	tt := []fromTo{
		{"b1", "a3"},
		{"b7", "b5"},
		{"g1", "h3"},
		{"b5", "b4"},
		{"c2", "c4"},
		{"b4", "c3"},
	}
	testMoves(t, b, tt)
}

func TestCastle(t *testing.T) {
	b := MustBoard(t, freshFen)
	tt := []fromTo{
		{"e2", "e4"},
		{"b7", "b5"},
		{"f1", "d3"},
		{"b5", "b4"},
		{"g1", "h3"},
		{"f7", "f6"},
		// castle
		{"e1", "g1"},
	}
	testMoves(t, b, tt)
}
