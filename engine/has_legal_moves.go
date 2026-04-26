package engine

// HasLegalMove checks if the position has at least one legal move
// This is MUCH faster than GenerateLegalMoves() because it returns
// as soon as it finds the first legal move, rather than generating all moves
func (p *Position) HasLegalMove() bool {
	// Generate pseudo-legal moves into a buffer
	var moveBuffer [256]Move
	moveCount := GenerateMovesIntoBuffer(p, moveBuffer[:])

	// Check each move for legality - return true on first legal move found
	for i := 0; i < moveCount; i++ {
		move := moveBuffer[i]
		movingColor := move.MovingPiece().Color()

		// Try the move
		oldEP, oldTag, oldHalfClock, _ := p.MakeMove(move)
		legal := !isInCheck(p, movingColor)
		p.UnMakeMove(move, oldTag, oldEP, oldHalfClock)
		if legal {
			return true
		}
	}

	// No legal moves found
	return false
}

// HasLegalMoveQuick is an even faster version that only checks a subset of moves
// Used in performance-critical paths where approximate answers are acceptable
func (p *Position) HasLegalMoveQuick() bool {
	// Just check if we have any pieces that can move
	// This is a quick heuristic - not 100% accurate but very fast

	turn := p.Turn()

	// Check if we have pieces (besides the king)
	if turn == White {
		if p.Board.whitePawn != 0 || p.Board.whiteKnight != 0 ||
		   p.Board.whiteBishop != 0 || p.Board.whiteRook != 0 ||
		   p.Board.whiteQueen != 0 {
			// We have pieces, likely have moves
			return true
		}
	} else {
		if p.Board.blackPawn != 0 || p.Board.blackKnight != 0 ||
		   p.Board.blackBishop != 0 || p.Board.blackRook != 0 ||
		   p.Board.blackQueen != 0 {
			// We have pieces, likely have moves
			return true
		}
	}

	// Only king left - need to check if it has legal moves
	var moveBuffer [32]Move
	moveCount := 0

	if turn == White {
		kingSquare := bitScanForward(p.Board.whiteKing)
		if kingSquare != 64 {
			// Generate only king moves
			moveCount = generateKingMovesQuick(&p.Board, Square(kingSquare), turn, moveBuffer[:])
		}
	} else {
		kingSquare := bitScanForward(p.Board.blackKing)
		if kingSquare != 64 {
			// Generate only king moves
			moveCount = generateKingMovesQuick(&p.Board, Square(kingSquare), turn, moveBuffer[:])
		}
	}

	// Check if any king move is legal
	for i := 0; i < moveCount; i++ {
		move := moveBuffer[i]
		movingColor := move.MovingPiece().Color()
		oldEP, oldTag, oldHalfClock, _ := p.MakeMove(move)
		legal := !isInCheck(p, movingColor)
		p.UnMakeMove(move, oldTag, oldEP, oldHalfClock)
		if legal {
			return true
		}
	}

	return false
}

// Helper function to generate king moves quickly
func generateKingMovesQuick(board *Bitboard, from Square, color Color, moves []Move) int {
	moveCount := 0

	var piece Piece
	if color == White {
		piece = WhiteKing
	} else {
		piece = BlackKing
	}

	// King can move one square in any direction
	kingMoves := [8][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1},           {0, 1},
		{1, -1},  {1, 0},  {1, 1},
	}

	fromRank := int(from / 8)
	fromFile := int(from % 8)

	for _, delta := range kingMoves {
		toRank := fromRank + delta[0]
		toFile := fromFile + delta[1]

		// Check bounds
		if toRank < 0 || toRank > 7 || toFile < 0 || toFile > 7 {
			continue
		}

		to := Square(toRank*8 + toFile)

		// Check if destination is occupied by own piece
		if board.PieceAt(to).Color() == color {
			continue
		}

		// Add the move
		captured := board.PieceAt(to)
		moves[moveCount] = NewMove(from, to, piece, captured, NoType, 0)
		moveCount++
	}

	return moveCount
}