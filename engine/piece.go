/*
https://github.com/amanjpro/zahak/?tab=MIT-1-ov-file#readme
MIT License

Copyright (c) 2021 Amanj Sherwany

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package engine

type Piece int8

const (
	NoPiece Piece = iota
	WhitePawn
	WhiteKnight
	WhiteBishop
	WhiteRook
	WhiteQueen
	WhiteKing
	BlackPawn
	BlackKnight
	BlackBishop
	BlackRook
	BlackQueen
	BlackKing
)

var Pieces = []Piece{
	WhitePawn,
	WhiteKnight,
	WhiteBishop,
	WhiteRook,
	WhiteQueen,
	WhiteKing,
	BlackPawn,
	BlackKnight,
	BlackBishop,
	BlackRook,
	BlackQueen,
	BlackKing,
}

type PieceType int8

const (
	NoType PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

type Color int8

const (
	NoColor Color = iota - 1
	Black
	White
)

func (c Color) Other() Color {
	if c == White {
		return Black
	} else if c == Black {
		return White
	}
	return NoColor
}

func (t PieceType) Name() string {
	switch t {
	case Pawn:
		return "P"
	case Knight:
		return "N"
	case Bishop:
		return "B"
	case Rook:
		return "R"
	case Queen:
		return "Q"
	case King:
		return "K"
	}
	return " "
}

func (p Piece) Type() PieceType {
	switch p {
	case WhitePawn, BlackPawn:
		return Pawn
	case WhiteKnight, BlackKnight:
		return Knight
	case WhiteBishop, BlackBishop:
		return Bishop
	case WhiteRook, BlackRook:
		return Rook
	case WhiteQueen, BlackQueen:
		return Queen
	case WhiteKing, BlackKing:
		return King
	}
	return NoType
}

const MAX_INT = int16(32767)

func (p Piece) seeWeight() int16 {
	switch p {
	case WhitePawn, BlackPawn:
		return 100
	case WhiteKnight, BlackKnight:
		return 300
	case WhiteBishop, BlackBishop:
		return 300
	case WhiteRook, BlackRook:
		return 500
	case WhiteQueen, BlackQueen:
		return 900
	case WhiteKing, BlackKing:
		return MAX_INT
	}
	return 0
}

func (p Piece) Weight() int16 {
	switch p {
	case WhitePawn, BlackPawn:
		return 100
	case WhiteKnight, BlackKnight:
		return 320
	case WhiteBishop, BlackBishop:
		return 330
	case WhiteRook, BlackRook:
		return 525
	case WhiteQueen, BlackQueen:
		return 1000
	case WhiteKing, BlackKing:
		return MAX_INT
	}
	return 0
}

func (p Piece) Name() string {
	switch p {
	case WhitePawn:
		return "P"
	case WhiteKnight:
		return "N"
	case WhiteBishop:
		return "B"
	case WhiteRook:
		return "R"
	case WhiteQueen:
		return "Q"
	case WhiteKing:
		return "K"
	case BlackPawn:
		return "p"
	case BlackKnight:
		return "n"
	case BlackBishop:
		return "b"
	case BlackRook:
		return "r"
	case BlackQueen:
		return "q"
	case BlackKing:
		return "k"
	}
	return " "
}

func pieceFromName(name rune) Piece {
	switch name {
	case 'P':
		return WhitePawn
	case 'N':
		return WhiteKnight
	case 'B':
		return WhiteBishop
	case 'R':
		return WhiteRook
	case 'Q':
		return WhiteQueen
	case 'K':
		return WhiteKing
	case 'p':
		return BlackPawn
	case 'n':
		return BlackKnight
	case 'b':
		return BlackBishop
	case 'r':
		return BlackRook
	case 'q':
		return BlackQueen
	case 'k':
		return BlackKing
	}
	return NoPiece
}

func (p Piece) Color() Color {
	switch p {
	case WhitePawn, WhiteKnight, WhiteBishop, WhiteRook, WhiteQueen, WhiteKing:
		return White
	case NoPiece:
		return NoColor
	}
	return Black
}

func GetPiece(pieceType PieceType, color Color) Piece {
	if color == White {
		switch pieceType {
		case King:
			return WhiteKing
		case Queen:
			return WhiteQueen
		case Rook:
			return WhiteRook
		case Bishop:
			return WhiteBishop
		case Knight:
			return WhiteKnight
		case Pawn:
			return WhitePawn
		}
	}
	if color == Black {
		switch pieceType {
		case King:
			return BlackKing
		case Queen:
			return BlackQueen
		case Rook:
			return BlackRook
		case Bishop:
			return BlackBishop
		case Knight:
			return BlackKnight
		case Pawn:
			return BlackPawn
		}
	}
	return NoPiece
}
