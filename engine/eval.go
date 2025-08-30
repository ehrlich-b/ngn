package engine

func Evaluate(board *Bitboard) (eval int) {
	for _, piece := range Pieces {
		if piece == NoPiece || piece == WhiteKing || piece == BlackKing {
			// These pieces don't have distinct values
			continue
		}
		pieceBitboard := board.GetBitboardOf(piece)
		color := piece.Color()
		val := piece.Weight()
		colorMul := 1
		if color == Black {
			colorMul = -1
		}
		eval += colorMul * int(val) * PopCount(pieceBitboard)
	}
	// TODO: Small negative eval for home squares
	// TODO: Small positive eval for center squares
	// TODO: Ramping eval for pawns reaching the opposite rank
	// TODO: Small positive eval for castling
	return
}
