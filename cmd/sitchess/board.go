package main

import (
	"fmt"
	"strings"
)

type Board struct {
	squares [][]*PlayerPiece
	next    Player

	castle [2][2]bool

	epx, epy int

	halfMoves int
	fullMoves int
}

type Player int
type Piece int

const (
	White Player = iota
	Black
)

const (
	King Piece = iota
	Queen
	Rook
	Bishop
	Knight
	Pawn
)

const freshFen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

var pieceCounts = []int{1, 1, 2, 2, 2, 8}

var blkFen = []byte{'k', 'q', 'r', 'b', 'n', 'p'}
var whtFen = []byte{'K', 'Q', 'R', 'B', 'N', 'P'}

//var blkUtf8 = []rune{'♔', '♕', '♖', '♗', '♘', '♙'}
//var whtUtf8 = []rune{'♚', '♛', '♜', '♝', '♞', '♟'}
//var blkFenR = []rune{'k', 'q', 'r', 'b', 'n', 'p'}
var whtFenR = []rune{'K', 'Q', 'R', 'B', 'N', 'P'}

var blkFenR = whtFenR
var whtUtf8 = whtFenR
var blkUtf8 = blkFenR

var fen2piece = map[rune]PlayerPiece{
	'K': PlayerPiece{White, King},
	'Q': PlayerPiece{White, Queen},
	'R': PlayerPiece{White, Rook},
	'B': PlayerPiece{White, Bishop},
	'N': PlayerPiece{White, Knight},
	'P': PlayerPiece{White, Pawn},

	'k': PlayerPiece{Black, King},
	'q': PlayerPiece{Black, Queen},
	'r': PlayerPiece{Black, Rook},
	'b': PlayerPiece{Black, Bishop},
	'n': PlayerPiece{Black, Knight},
	'p': PlayerPiece{Black, Pawn},
}

type PlayerPiece struct {
	player Player
	piece  Piece
}

func (p PlayerPiece) Rune() rune {
	if p.player == Black {
		return blkUtf8[p.piece]
	}
	return whtUtf8[p.piece]
}

func (p PlayerPiece) Fen() byte {
	if p.player == Black {
		return blkFen[p.piece]
	}
	return whtFen[p.piece]
}

func Str2Coord(s string) (int, int) {
	if len(s) != 2 || s[0] < 'a' || s[1] > 'h' || s[1] < '1' || s[1] > '8' {
		return -1, -1
	}
	return int(s[0] - 'a'), int(7 - (s[1] - '1'))
}

func Coord2Str(x, y int) string {
	if x < 0 || x >= 8 || y < 0 || y >= 8 {
		return "-"
	}
	return fmt.Sprintf("%c%d", 'a'+x, 8-y)
}

func NewBoard(fen string) (b *Board, err error) {
	tokens := strings.Split(fen, " ")
	pieces := strings.Split(tokens[0], "/")
	b = &Board{
		squares:   make([][]*PlayerPiece, len(pieces)),
		next:      White,
		halfMoves: 0,
		fullMoves: 0,
	}
	for i, row := range pieces {
		for _, v := range row {
			if p, ok := fen2piece[v]; ok {
				b.squares[i] = append(b.squares[i], &p)
			} else if v >= '1' && v <= '9' {
				for j := 0; j < int(v-'0'); j++ {
					b.squares[i] = append(b.squares[i], nil)
				}
			} else {
				return nil, fmt.Errorf("bad fen row %q", row)
			}
		}
	}
	if tokens[1] == "b" {
		b.next = Black
	}
	if tokens[2] != "-" {
		for _, c := range tokens[2] {
			p, ok := fen2piece[c]
			if !ok {
				return nil, fmt.Errorf("unknown castling %q", tokens[2])
			}
			b.castle[p.player][p.piece] = true
		}
	}

	b.epx, b.epy = Str2Coord(tokens[3])
	fmt.Sscanf(tokens[4], "%d", &b.halfMoves)
	fmt.Sscanf(tokens[5], "%d", &b.fullMoves)
	return b, nil
}

func (b *Board) Fen() string {
	s := ""
	for _, row := range b.squares {
		spc := 0
		for _, p := range row {
			if p == nil {
				spc++
				continue
			}
			if spc != 0 {
				s += fmt.Sprintf("%d", spc)
				spc = 0
			}
			s += string(p.Fen())
		}
		if spc != 0 {
			s += fmt.Sprintf("%d", spc)
		}
		s += "/"
	}
	// Drop trailing /.
	s = s[:len(s)-1]

	// Next move.
	if b.next == Black {
		s += " b"
	} else {
		s += " w"
	}

	// Castling state.
	s += " "
	for _, pl := range []Player{White, Black} {
		for _, pi := range []Piece{King, Queen} {
			if b.castle[pl][pi] {
				s += string(PlayerPiece{pl, pi}.Fen())
			}
		}
	}
	if s[len(s)-1] == ' ' {
		s += "-"
	}

	s += fmt.Sprintf(" %s %d %d", Coord2Str(b.epx, b.epy), b.halfMoves, b.fullMoves)
	return s
}

func (b *Board) inBounds(x, y int) bool {
	return y >= 0 && y < len(b.squares) && x >= 0 && x < len(b.squares[y])
}

func (b *Board) Get(x, y int) *PlayerPiece {
	if y >= len(b.squares) || x >= len(b.squares[y]) {
		return nil
	}
	return b.squares[y][x]
}

func (b *Board) isClear(x0, y0, x1, y1 int) bool {
	dx, dy := x1-x0, y1-y0
	sx, sy := 1, 1
	if dx < 0 {
		sx = -1
	} else if dx == 0 {
		sx = 0
	}
	if dy < 0 {
		sy = -1
	} else if dy == 0 {
		sy = 0
	}
	for x, y := x0+sx, y0+sy; x != x1 && y != y1; x, y = x+sx, y+sy {
		if b.Get(x, y) != nil {
			return false
		}
	}
	return true
}

func (b *Board) Count(p Player) []int {
	ret := make([]int, 6)
	for _, row := range b.squares {
		for _, sq := range row {
			if sq != nil && sq.player == p {
				ret[sq.piece]++
			}
		}
	}
	return ret
}

// Move using algebraic notation.
func (b *Board) MoveAlgebraic(p Piece, x1, y1 int) bool {
	want := PlayerPiece{b.next, p}
	for y, row := range b.squares {
		for x, sq := range row {
			if sq != nil && *sq == want {
				if b.Move(x, y, x1, y1) {
					return true
				}
			}
		}
	}
	return false
}

func (b *Board) Move(x0, y0, x1, y1 int) bool {
	if !b.inBounds(x0, y0) || !b.inBounds(x1, y1) {
		return false
	}
	p := b.squares[y0][x0]
	if p == nil || p.player != b.next {
		return false
	}
	if pp := b.squares[y1][x1]; pp != nil && pp.player == b.next {
		return false
	}
	tgtEmpty := b.squares[y1][x1] == nil
	dx, dy, ok := x1-x0, y1-y0, false
	isEp, isCastle := false, false
	switch p.piece {
	case King:
		ok = dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1
		if dy == 0 {
			ckc := b.isClear(x0, y0, 7, y0)
			ck := b.castle[p.player][King] && dx == 2 && ckc
			cqc := b.isClear(x0, y0, 0, y0)
			cq := b.castle[p.player][Queen] && dx == -2 && cqc
			isCastle = ck || cq
		}
		ok = ok || isCastle
	case Queen:
		if dx == 0 || dy == 0 || dx == dy || dx == -dy {
			ok = b.isClear(x0, y0, x1, y1)
		}
	case Rook:
		ok = (dx == 0 || dy == 0) && b.isClear(x0, y0, x1, y1)
	case Bishop:
		ok = (dx == dy || dx == -dy) && b.isClear(x0, y0, x1, y1)
	case Knight:
		ok = ok || ((dx == 1 || dx == -1) && (dy == 2 || dy == -2))
		ok = ok || ((dx == 2 || dx == -2) && (dy == 1 || dy == -1))
	case Pawn:
		ok = ok || (p.player == White && dx == 0 && dy == -1 && tgtEmpty)
		ok = ok || (p.player == White && dx == 0 && dy == -2 && y0 == 6 && tgtEmpty)
		ok = ok || (p.player == White && (dx == 1 || dx == -1) && dy == -1 && !tgtEmpty)
		isEp = isEp || (p.player == White && (dx == 1 || dx == -1) && dy == -1 && tgtEmpty && x1 == b.epx && y0 == b.epy)

		ok = ok || (p.player == Black && dx == 0 && dy == 1 && tgtEmpty)
		ok = ok || (p.player == Black && dx == 0 && dy == 2 && y0 == 1 && tgtEmpty)
		ok = ok || (p.player == Black && (dx == 1 || dx == -1) && dy == 1 && !tgtEmpty)
		isEp = isEp || (p.player == Black && (dx == 1 || dx == -1) && dy == 1 && tgtEmpty && x1 == b.epx && y0 == b.epy)

		ok = (ok || isEp) && b.isClear(x0, y0, x1, y1)
	}
	if !ok {
		return false
	}
	captured := b.squares[y1][x1]
	b.squares[y0][x0], b.squares[y1][x1] = nil, p
	if isCastle {
		// Move rook for castling.
		if dx < 0 {
			b.squares[y1][4] = b.squares[y1][0]
			b.squares[y1][0] = nil
		} else {
			b.squares[y1][5] = b.squares[y1][7]
			b.squares[y1][7] = nil
		}
	}

	if isEp {
		b.squares[b.epx][b.epy] = nil
	}
	b.epx, b.epy = -1, -1
	if p.piece == Pawn && (dy == -2 || dy == 2) {
		b.epx, b.epy = x1, y1
	}

	if p.piece == King {
		b.castle[p.player][King] = false
		b.castle[p.player][Queen] = false
	}
	updateRookCastling := func(pp PlayerPiece, x, y int) {
		if pp.piece != Rook || y > 0 && y < 7 {
			return
		} else if x == 0 {
			b.castle[p.player][Queen] = false
		} else if x == 7 {
			b.castle[p.player][King] = false
		}
	}
	updateRookCastling(*p, x0, y0)
	if captured != nil {
		updateRookCastling(*captured, x1, y1)
	}

	if tgtEmpty && p.piece != Pawn {
		b.halfMoves++
	} else {
		b.halfMoves = 0
	}
	if b.next == White {
		b.next = Black
	} else {
		b.next = White
		b.fullMoves++
	}
	return true
}
