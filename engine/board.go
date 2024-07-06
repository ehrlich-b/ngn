package engine

type Bitboard uint64

type Board struct {
	whitePawns   Bitboard
	whiteKnights Bitboard
	whiteBishops Bitboard
	whiteRooks   Bitboard
	whiteQueens  Bitboard
	whiteKing    Bitboard

	blackPawns   Bitboard
	blackKnights Bitboard
	blackBishops Bitboard
	blackRooks   Bitboard
	blackQueens  Bitboard
	blackKing    Bitboard

	whiteOccupancy Bitboard
	blackOccupancy Bitboard

	whiteCastleKingside  bool
	whiteCastleQueenside bool
	blackCastleKingside  bool
	blackCastleQueenside bool

	enPassantSquare int
}

func NewBoard() *Board {
	b := Board{}
	b.whitePawns = Bitboard(A2 | B2 | C2 | D2 | E2 | F2 | G2 | H2)
	b.whiteKnights = Bitboard(B1 | G1)
	b.whiteBishops = Bitboard(C1 | F1)
	b.whiteRooks = Bitboard(A1 | H1)
	b.whiteQueens = Bitboard(D1)
	b.whiteKing = Bitboard(E1)

	b.blackPawns = Bitboard(A7 | B7 | C7 | D7 | E7 | F7 | G7 | H7)
	b.blackKnights = Bitboard(B8 | G8)
	b.blackBishops = Bitboard(C8 | F8)
	b.blackRooks = Bitboard(A8 | H8)
	b.blackQueens = Bitboard(D8)
	b.blackKing = Bitboard(E8)

	b.whiteOccupancy = b.WhiteOccupied()
	b.blackOccupancy = b.BlackOccupied()

	b.whiteCastleKingside = true
	b.whiteCastleQueenside = true
	b.blackCastleKingside = true
	b.blackCastleQueenside = true

	b.enPassantSquare = -1

	return &b
}

func (b *Board) WhiteOccupied() Bitboard {
	return b.whitePawns | b.whiteKnights | b.whiteBishops | b.whiteRooks | b.whiteQueens | b.whiteKing
}

func (b *Board) BlackOccupied() Bitboard {
	return b.blackPawns | b.blackKnights | b.blackBishops | b.blackRooks | b.blackQueens | b.blackKing
}

func (b *Board) Occupied() Bitboard {
	return b.whiteOccupancy | b.blackOccupancy
}

func (b *Board) MovePiece(from Position, to Position) {
	fromPos := Bitboard(from)
	toPos := Bitboard(to)
	fromWhiteOrBlack := b.whiteOccupancy & fromPos
	toOccupied := b.Occupied() & toPos

	if fromWhiteOrBlack != 0 {
		b.whiteOccupancy ^= fromPos
		b.whiteOccupancy |= toPos
		if toOccupied != 0 {
			b.blackPawns ^= toPos
			b.blackBishops ^= toPos
			b.blackKnights ^= toPos
			b.blackQueens ^= toPos
			b.blackRooks ^= toPos
			b.blackKing ^= toPos
			b.blackOccupancy ^= toPos
		}
	} else {
		b.blackOccupancy ^= fromPos
		b.blackOccupancy |= toPos
		if toOccupied != 0 {
			b.whitePawns ^= toPos
			b.whiteBishops ^= toPos
			b.whiteKnights ^= toPos
			b.whiteQueens ^= toPos
			b.whiteRooks ^= toPos
			b.whiteKing ^= toPos
			b.whiteOccupancy ^= toPos
		}
	}
}

type Position uint64

const (
	A1 Position = 0
	B1 Position = 1
	C1 Position = B1 << 1
	D1 Position = C1 << 1
	E1 Position = D1 << 1
	F1 Position = E1 << 1
	G1 Position = F1 << 1
	H1 Position = G1 << 1
	A2 Position = H1 << 1
	B2 Position = A2 << 1
	C2 Position = B2 << 1
	D2 Position = C2 << 1
	E2 Position = D2 << 1
	F2 Position = E2 << 1
	G2 Position = F2 << 1
	H2 Position = G2 << 1
	A3 Position = H2 << 1
	B3 Position = A3 << 1
	C3 Position = B3 << 1
	D3 Position = C3 << 1
	E3 Position = D3 << 1
	F3 Position = E3 << 1
	G3 Position = F3 << 1
	H3 Position = G3 << 1
	A4 Position = H3 << 1
	B4 Position = A4 << 1
	C4 Position = B4 << 1
	D4 Position = C4 << 1
	E4 Position = D4 << 1
	F4 Position = E4 << 1
	G4 Position = F4 << 1
	H4 Position = G4 << 1
	A5 Position = H4 << 1
	B5 Position = A5 << 1
	C5 Position = B5 << 1
	D5 Position = C5 << 1
	E5 Position = D5 << 1
	F5 Position = E5 << 1
	G5 Position = F5 << 1
	H5 Position = G5 << 1
	A6 Position = H5 << 1
	B6 Position = A6 << 1
	C6 Position = B6 << 1
	D6 Position = C6 << 1
	E6 Position = D6 << 1
	F6 Position = E6 << 1
	G6 Position = F6 << 1
	H6 Position = G6 << 1
	A7 Position = H6 << 1
	B7 Position = A7 << 1
	C7 Position = B7 << 1
	D7 Position = C7 << 1
	E7 Position = D7 << 1
	F7 Position = E7 << 1
	G7 Position = F7 << 1
	H7 Position = G7 << 1
	A8 Position = H7 << 1
	B8 Position = A8 << 1
	C8 Position = B8 << 1
	D8 Position = C8 << 1
	E8 Position = D8 << 1
	F8 Position = E8 << 1
	G8 Position = F8 << 1
	H8 Position = G8 << 1
)
