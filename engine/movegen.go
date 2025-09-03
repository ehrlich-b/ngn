package engine

import "math/bits"

// GenerateMoves generates all pseudo-legal moves for the current position
func GenerateMoves(pos *Position) []Move {
	return WrapMoveGeneration("GenerateMoves", func() []Move {
		return generateMovesUnsafe(pos)
	}, pos)
}

func generateMovesUnsafe(pos *Position) []Move {
	moves := make([]Move, 0, 64)
	turn := pos.Turn()
	
	if turn == White {
		moves = append(moves, generateWhitePawnMoves(pos)...)
		moves = append(moves, generatePieceMoves(pos, WhiteKnight, generateKnightMoves)...)
		moves = append(moves, generatePieceMoves(pos, WhiteBishop, generateBishopMoves)...)
		moves = append(moves, generatePieceMoves(pos, WhiteRook, generateRookMoves)...)
		moves = append(moves, generatePieceMoves(pos, WhiteQueen, generateQueenMoves)...)
		moves = append(moves, generateKingMoves(pos, WhiteKing)...)
	} else {
		moves = append(moves, generateBlackPawnMoves(pos)...)
		moves = append(moves, generatePieceMoves(pos, BlackKnight, generateKnightMoves)...)
		moves = append(moves, generatePieceMoves(pos, BlackBishop, generateBishopMoves)...)
		moves = append(moves, generatePieceMoves(pos, BlackRook, generateRookMoves)...)
		moves = append(moves, generatePieceMoves(pos, BlackQueen, generateQueenMoves)...)
		moves = append(moves, generateKingMoves(pos, BlackKing)...)
	}
	
	return moves
}

// GenerateLegalMoves generates all legal moves for the current position
func GenerateLegalMoves(pos *Position) []Move {
	return WrapMoveGeneration("GenerateLegalMoves", func() []Move {
		return generateLegalMovesUnsafe(pos)
	}, pos)
}

func generateLegalMovesUnsafe(pos *Position) []Move {
	pseudoLegalMoves := generateMovesUnsafe(pos)
	legalMoves := make([]Move, 0, len(pseudoLegalMoves))
	
	for _, move := range pseudoLegalMoves {
		if IsLegalMove(pos, move) {
			legalMoves = append(legalMoves, move)
		}
	}
	
	return legalMoves
}


// GenerateCaptures generates only capture moves (for quiescence search)
func GenerateCaptures(pos *Position) []Move {
	return WrapMoveGeneration("GenerateCaptures", func() []Move {
		return generateCapturesUnsafe(pos)
	}, pos)
}

func generateCapturesUnsafe(pos *Position) []Move {
	moves := generateMovesUnsafe(pos)
	captures := make([]Move, 0, len(moves))
	
	for _, move := range moves {
		if move.IsCapture() {
			captures = append(captures, move)
		}
	}
	
	return captures
}


// generatePieceMoves generates moves for a specific piece type using the provided generator function
func generatePieceMoves(pos *Position, piece Piece, generator func(*Position, Square) []Move) []Move {
	moves := make([]Move, 0, 32)
	pieceBitboard := pos.Board.GetBitboardOf(piece)
	
	for pieceBitboard != 0 {
		square := Square(bits.TrailingZeros64(pieceBitboard))
		moves = append(moves, generator(pos, square)...)
		pieceBitboard &^= SquareMask[int(square)]
	}
	
	return moves
}

// generateWhitePawnMoves generates all white pawn moves
func generateWhitePawnMoves(pos *Position) []Move {
	moves := make([]Move, 0, 16)
	whitePawns := pos.Board.GetBitboardOf(WhitePawn)
	allPieces := pos.Board.GetWhitePieces() | pos.Board.GetBlackPieces()
	enemyPieces := pos.Board.GetBlackPieces()
	
	// Single pawn pushes
	singlePushes := (whitePawns << 8) & ^allPieces
	singlePushesCopy := singlePushes // Save copy for double push calculation
	
	for singlePushes != 0 {
		to := Square(bits.TrailingZeros64(singlePushes))
		from := Square(int(to) - 8)
		
		// Check for promotion
		if to.Rank() == Rank8 {
			moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, Queen, 0))
			moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, Rook, 0))
			moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, Bishop, 0))
			moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, Knight, 0))
		} else {
			moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, NoType, 0))
		}
		
		singlePushes &^= SquareMask[int(to)]
	}
	
	// Double pawn pushes - only from starting rank, with both intermediate and destination empty
	doublePushes := ((singlePushesCopy & RankMasks[Rank3]) << 8) & ^allPieces
	for doublePushes != 0 {
		to := Square(bits.TrailingZeros64(doublePushes))
		from := Square(int(to) - 16)
		moves = append(moves, NewMove(from, to, WhitePawn, NoPiece, NoType, 0))
		doublePushes &^= SquareMask[int(to)]
	}
	
	// Pawn captures (diagonal left)
	leftCaptures := ((whitePawns & ^FileMasks[FileA]) << 7) & enemyPieces
	for leftCaptures != 0 {
		to := Square(bits.TrailingZeros64(leftCaptures))
		from := Square(int(to) - 7)
		capturedPiece := pos.Board.PieceAt(to)
		
		// Check for promotion
		if to.Rank() == Rank8 {
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Queen, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Rook, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Bishop, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Knight, Capture))
		} else {
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, NoType, Capture))
		}
		
		leftCaptures &^= SquareMask[int(to)]
	}
	
	// Pawn captures (diagonal right)
	rightCaptures := ((whitePawns & ^FileMasks[FileH]) << 9) & enemyPieces
	for rightCaptures != 0 {
		to := Square(bits.TrailingZeros64(rightCaptures))
		from := Square(int(to) - 9)
		
		// Bounds check
		if int(to) < 0 || int(to) > 63 || int(from) < 0 || int(from) > 63 {
			rightCaptures &^= SquareMask[int(to)]
			continue
		}
		
		capturedPiece := pos.Board.PieceAt(to)
		
		// Check for promotion
		if to.Rank() == Rank8 {
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Queen, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Rook, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Bishop, Capture))
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, Knight, Capture))
		} else {
			moves = append(moves, NewMove(from, to, WhitePawn, capturedPiece, NoType, Capture))
		}
		
		rightCaptures &^= SquareMask[int(to)]
	}
	
	// En passant captures
	if pos.EnPassant != NoSquare {
		enPassantTarget := pos.EnPassant
		// Check left diagonal for en passant
		if enPassantTarget.File() > FileA {
			leftSquare := SquareOf(enPassantTarget.File()-1, enPassantTarget.Rank()-1)
			if pos.Board.PieceAt(leftSquare) == WhitePawn {
				moves = append(moves, NewMove(leftSquare, enPassantTarget, WhitePawn, BlackPawn, NoType, EnPassant|Capture))
			}
		}
		// Check right diagonal for en passant
		if enPassantTarget.File() < FileH {
			rightSquare := SquareOf(enPassantTarget.File()+1, enPassantTarget.Rank()-1)
			if pos.Board.PieceAt(rightSquare) == WhitePawn {
				moves = append(moves, NewMove(rightSquare, enPassantTarget, WhitePawn, BlackPawn, NoType, EnPassant|Capture))
			}
		}
	}
	
	return moves
}

// generateBlackPawnMoves generates all black pawn moves
func generateBlackPawnMoves(pos *Position) []Move {
	moves := make([]Move, 0, 16)
	blackPawns := pos.Board.GetBitboardOf(BlackPawn)
	allPieces := pos.Board.GetWhitePieces() | pos.Board.GetBlackPieces()
	enemyPieces := pos.Board.GetWhitePieces()
	
	// Single pawn pushes (black pawns move "down" the board)
	singlePushes := (blackPawns >> 8) & ^allPieces
	singlePushesCopy := singlePushes // Save copy for double push calculation
	
	for singlePushes != 0 {
		to := Square(bits.TrailingZeros64(singlePushes))
		from := Square(int(to) + 8)
		
		// Check for promotion
		if to.Rank() == Rank1 {
			moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, Queen, 0))
			moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, Rook, 0))
			moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, Bishop, 0))
			moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, Knight, 0))
		} else {
			moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, NoType, 0))
		}
		
		singlePushes &^= SquareMask[int(to)]
	}
	
	// Double pawn pushes - only from starting rank, with both intermediate and destination empty
	doublePushes := ((singlePushesCopy & RankMasks[Rank6]) >> 8) & ^allPieces
	for doublePushes != 0 {
		to := Square(bits.TrailingZeros64(doublePushes))
		from := Square(int(to) + 16)
		moves = append(moves, NewMove(from, to, BlackPawn, NoPiece, NoType, 0))
		doublePushes &^= SquareMask[int(to)]
	}
	
	// Pawn captures (diagonal left for black)
	leftCaptures := ((blackPawns & ^FileMasks[FileH]) >> 7) & enemyPieces
	for leftCaptures != 0 {
		to := Square(bits.TrailingZeros64(leftCaptures))
		from := Square(int(to) + 7)
		capturedPiece := pos.Board.PieceAt(to)
		
		// Check for promotion
		if to.Rank() == Rank1 {
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Queen, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Rook, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Bishop, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Knight, Capture))
		} else {
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, NoType, Capture))
		}
		
		leftCaptures &^= SquareMask[int(to)]
	}
	
	// Pawn captures (diagonal right for black)
	rightCaptures := ((blackPawns & ^FileMasks[FileA]) >> 9) & enemyPieces
	for rightCaptures != 0 {
		to := Square(bits.TrailingZeros64(rightCaptures))
		from := Square(int(to) + 9)
		capturedPiece := pos.Board.PieceAt(to)
		
		// Check for promotion
		if to.Rank() == Rank1 {
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Queen, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Rook, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Bishop, Capture))
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, Knight, Capture))
		} else {
			moves = append(moves, NewMove(from, to, BlackPawn, capturedPiece, NoType, Capture))
		}
		
		rightCaptures &^= SquareMask[int(to)]
	}
	
	// En passant captures
	if pos.EnPassant != NoSquare {
		enPassantTarget := pos.EnPassant
		// Check left diagonal for en passant (from black's perspective)
		if enPassantTarget.File() > FileA {
			leftSquare := SquareOf(enPassantTarget.File()-1, enPassantTarget.Rank()+1)
			if pos.Board.PieceAt(leftSquare) == BlackPawn {
				moves = append(moves, NewMove(leftSquare, enPassantTarget, BlackPawn, WhitePawn, NoType, EnPassant|Capture))
			}
		}
		// Check right diagonal for en passant (from black's perspective)
		if enPassantTarget.File() < FileH {
			rightSquare := SquareOf(enPassantTarget.File()+1, enPassantTarget.Rank()+1)
			if pos.Board.PieceAt(rightSquare) == BlackPawn {
				moves = append(moves, NewMove(rightSquare, enPassantTarget, BlackPawn, WhitePawn, NoType, EnPassant|Capture))
			}
		}
	}
	
	return moves
}

// generateKnightMoves generates moves for a knight at the given square
func generateKnightMoves(pos *Position, from Square) []Move {
	moves := make([]Move, 0, 8)
	attacks := KnightAttacks[from]
	piece := pos.Board.PieceAt(from)
	friendlyPieces := pos.Board.GetWhitePieces()
	if piece.Color() == Black {
		friendlyPieces = pos.Board.GetBlackPieces()
	}
	
	// Remove squares occupied by friendly pieces
	attacks &^= friendlyPieces
	
	for attacks != 0 {
		to := Square(bits.TrailingZeros64(attacks))
		capturedPiece := pos.Board.PieceAt(to)
		tag := MoveTag(0)
		if capturedPiece != NoPiece {
			tag = Capture
		}
		moves = append(moves, NewMove(from, to, piece, capturedPiece, NoType, tag))
		attacks &^= SquareMask[int(to)]
	}
	
	return moves
}

// generateBishopMoves generates moves for a bishop at the given square
func generateBishopMoves(pos *Position, from Square) []Move {
	return generateSlidingMoves(pos, from, bishopDirections)
}

// generateRookMoves generates moves for a rook at the given square
func generateRookMoves(pos *Position, from Square) []Move {
	return generateSlidingMoves(pos, from, rookDirections)
}

// generateQueenMoves generates moves for a queen at the given square
func generateQueenMoves(pos *Position, from Square) []Move {
	return generateSlidingMoves(pos, from, queenDirections)
}

// generateSlidingMoves generates moves for sliding pieces (bishop, rook, queen)
// PERFORMANCE: Changed to return fixed-size array to eliminate heap allocations
// Was allocating 253MB in profiling! Now stack-allocated.
func generateSlidingMoves(pos *Position, from Square, directions []int) []Move {
	// Stack-allocated buffer - max 27 moves (queen on center square)
	var moveBuffer [28]Move
	numMoves := 0
	piece := pos.Board.PieceAt(from)
	friendlyPieces := pos.Board.GetWhitePieces()
	if piece.Color() == Black {
		friendlyPieces = pos.Board.GetBlackPieces()
	}
	allPieces := pos.Board.GetWhitePieces() | pos.Board.GetBlackPieces()
	
	for _, direction := range directions {
		for i := 1; i < 8; i++ {
			to := int(from) + direction*i
			
			// Check bounds
			if to < 0 || to > 63 {
				break
			}
			
			// Check for file wrapping on horizontal and diagonal moves
			fromFile := int(from) % 8
			toFile := to % 8
			
			// For horizontal moves, check direct file wrapping
			if direction == 1 && toFile < fromFile {
				break
			}
			if direction == -1 && toFile > fromFile {
				break
			}
			
			// For diagonal moves, check file distance matches step count and prevent board wrapping
			if abs(direction) == 7 || abs(direction) == 9 {
				// Check if we've wrapped around the board horizontally
				expectedFileDiff := i
				if direction == 7 || direction == -9 {
					expectedFileDiff = -i
				}
				actualFileDiff := toFile - fromFile
				
				// If the actual file difference doesn't match expected, or if we wrapped, break
				if actualFileDiff != expectedFileDiff {
					break
				}
			}
			
			toSquare := Square(to)
			
			// If square is occupied by friendly piece, stop
			if SquareMask[to]&friendlyPieces != 0 {
				break
			}
			
			capturedPiece := pos.Board.PieceAt(toSquare)
			tag := MoveTag(0)
			if capturedPiece != NoPiece {
				tag = Capture
			}
			
			moveBuffer[numMoves] = NewMove(from, toSquare, piece, capturedPiece, NoType, tag)
			numMoves++
			
			// If we captured an enemy piece, stop sliding in this direction
			if SquareMask[to]&allPieces != 0 {
				break
			}
		}
	}
	
	return moveBuffer[:numMoves]
}

// generateKingMoves generates moves for a king at the given square (including castling)
func generateKingMoves(pos *Position, piece Piece) []Move {
	moves := make([]Move, 0, 8)
	kingBitboard := pos.Board.GetBitboardOf(piece)
	
	// Check if king exists on the board
	if kingBitboard == 0 {
		return moves // No king found, return empty moves
	}
	
	kingSquare := Square(bits.TrailingZeros64(kingBitboard))
	
	// Additional safety check (should never happen with valid positions)
	if kingSquare > 63 {
		return moves
	}
	
	attacks := KingAttacks[kingSquare]
	friendlyPieces := pos.Board.GetWhitePieces()
	if piece.Color() == Black {
		friendlyPieces = pos.Board.GetBlackPieces()
	}
	
	// Remove squares occupied by friendly pieces
	attacks &^= friendlyPieces
	
	for attacks != 0 {
		to := Square(bits.TrailingZeros64(attacks))
		capturedPiece := pos.Board.PieceAt(to)
		tag := MoveTag(0)
		if capturedPiece != NoPiece {
			tag = Capture
		}
		moves = append(moves, NewMove(kingSquare, to, piece, capturedPiece, NoType, tag))
		attacks &^= SquareMask[int(to)]
	}
	
	// Generate castling moves
	if piece == WhiteKing && kingSquare == E1 {
		// White king-side castling
		if pos.HasTag(WhiteCanCastleKingSide) && 
			pos.Board.PieceAt(F1) == NoPiece && pos.Board.PieceAt(G1) == NoPiece &&
			pos.Board.PieceAt(H1) == WhiteRook {
			moves = append(moves, NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle))
		}
		// White queen-side castling
		if pos.HasTag(WhiteCanCastleQueenSide) &&
			pos.Board.PieceAt(D1) == NoPiece && pos.Board.PieceAt(C1) == NoPiece && pos.Board.PieceAt(B1) == NoPiece &&
			pos.Board.PieceAt(A1) == WhiteRook {
			moves = append(moves, NewMove(E1, C1, WhiteKing, NoPiece, NoType, QueenSideCastle))
		}
	} else if piece == BlackKing && kingSquare == E8 {
		// Black king-side castling
		if pos.HasTag(BlackCanCastleKingSide) &&
			pos.Board.PieceAt(F8) == NoPiece && pos.Board.PieceAt(G8) == NoPiece &&
			pos.Board.PieceAt(H8) == BlackRook {
			moves = append(moves, NewMove(E8, G8, BlackKing, NoPiece, NoType, KingSideCastle))
		}
		// Black queen-side castling
		if pos.HasTag(BlackCanCastleQueenSide) &&
			pos.Board.PieceAt(D8) == NoPiece && pos.Board.PieceAt(C8) == NoPiece && pos.Board.PieceAt(B8) == NoPiece &&
			pos.Board.PieceAt(A8) == BlackRook {
			moves = append(moves, NewMove(E8, C8, BlackKing, NoPiece, NoType, QueenSideCastle))
		}
	}
	
	return moves
}

// Movement direction constants for sliding pieces
var (
	bishopDirections = []int{-9, -7, 7, 9}     // diagonal moves
	rookDirections   = []int{-8, -1, 1, 8}     // horizontal and vertical moves
	queenDirections  = []int{-9, -8, -7, -1, 1, 7, 8, 9} // all directions
)

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// IsLegalMove checks if a move is legal (doesn't leave king in check)
func IsLegalMove(pos *Position, move Move) bool {
	// Special handling for castling - check intermediate squares
	if move.IsCastle() {
		color := move.MovingPiece().Color()
		
		// King must not be in check before castling
		if isInCheck(pos, color) {
			return false
		}
		
		// Check intermediate and final squares
		var intermediateSquare, finalSquare Square
		if move.IsKingSideCastle() {
			if color == White {
				intermediateSquare = F1
				finalSquare = G1
			} else {
				intermediateSquare = F8
				finalSquare = G8
			}
		} else { // Queen-side castle
			if color == White {
				intermediateSquare = D1
				finalSquare = C1
			} else {
				intermediateSquare = D8
				finalSquare = C8
			}
		}
		
		// Check if intermediate square is attacked
		if isSquareAttacked(pos, intermediateSquare, color.Other()) {
			return false
		}
		
		// Check if final square is attacked
		if isSquareAttacked(pos, finalSquare, color.Other()) {
			return false
		}
		
		return true
	}
	
	// For non-castling moves, use the standard approach
	// Make the move
	ep, tag, hc, _ := pos.MakeMove(move)
	
	// Check if our king is in check after the move
	legal := !isInCheck(pos, move.MovingPiece().Color())
	
	// Unmake the move
	pos.UnMakeMove(move, tag, ep, hc)
	
	return legal
}

// isSquareAttacked checks if a square is attacked by pieces of the given color
func isSquareAttacked(pos *Position, square Square, attackingColor Color) bool {
	// Temporarily place a king of the opposite color on the square
	originalPiece := pos.Board.PieceAt(square)
	var testKing Piece
	if attackingColor == White {
		testKing = BlackKing
	} else {
		testKing = WhiteKing
	}
	
	pos.Board.UpdateSquare(square, testKing, originalPiece)
	attacked := isInCheck(pos, testKing.Color())
	pos.Board.UpdateSquare(square, originalPiece, testKing)
	
	return attacked
}

// IsInCheckDirect checks if the king of the given color is in check (public version)
func IsInCheckDirect(pos *Position, color Color) bool {
	return isInCheck(pos, color)
}

// isInCheck checks if the king of the given color is in check
func isInCheck(pos *Position, color Color) bool {
	// Find the king
	var kingPiece Piece
	if color == White {
		kingPiece = WhiteKing
	} else {
		kingPiece = BlackKing
	}
	
	kingBitboard := pos.Board.GetBitboardOf(kingPiece)
	if kingBitboard == 0 {
		return false // No king found
	}
	
	kingSquare := Square(bits.TrailingZeros64(kingBitboard))
	
	// Safety check for valid square
	if kingSquare > 63 {
		return false // Invalid square
	}
	
	// Check if any enemy piece can attack the king
	enemyColor := color.Other()
	
	// Check pawn attacks
	if color == White {
		// White king being attacked by black pawns - check squares where black pawns would be to attack king
		// Black pawns attack from squares king+7 and king+9
		leftAttack := (SquareMask[kingSquare] << 7) & ^FileMasks[FileA]  // black pawn would be at king+7
		rightAttack := (SquareMask[kingSquare] << 9) & ^FileMasks[FileH] // black pawn would be at king+9
		if (leftAttack|rightAttack)&pos.Board.GetBitboardOf(BlackPawn) != 0 {
			return true
		}
	} else {
		// Black king being attacked by white pawns - check squares where white pawns would be to attack king
		// White pawns attack from squares king-7 and king-9
		leftAttack := (SquareMask[kingSquare] >> 7) & ^FileMasks[FileH]  // white pawn would be at king-7
		rightAttack := (SquareMask[kingSquare] >> 9) & ^FileMasks[FileA] // white pawn would be at king-9
		if (leftAttack|rightAttack)&pos.Board.GetBitboardOf(WhitePawn) != 0 {
			return true
		}
	}
	
	// Check knight attacks
	enemyKnights := pos.Board.GetBitboardOf(GetPiece(Knight, enemyColor))
	if KnightAttacks[kingSquare]&enemyKnights != 0 {
		return true
	}
	
	// Check king attacks (for adjacent king positions)
	enemyKing := pos.Board.GetBitboardOf(GetPiece(King, enemyColor))
	if KingAttacks[kingSquare]&enemyKing != 0 {
		return true
	}
	
	// Check sliding piece attacks (bishop, rook, queen)
	
	// Check diagonal attacks (bishop, queen)
	for _, direction := range bishopDirections {
		for i := 1; i < 8; i++ {
			to := int(kingSquare) + direction*i
			
			if to < 0 || to > 63 {
				break
			}
			
			// Check bounds wrapping
			fromFile := int(kingSquare) % 8
			toFile := to % 8
			if abs(fromFile-toFile) > i {
				break
			}
			
			toSquare := Square(to)
			piece := pos.Board.PieceAt(toSquare)
			
			if piece != NoPiece {
				if piece.Color() == enemyColor && (piece.Type() == Bishop || piece.Type() == Queen) {
					return true
				}
				break
			}
		}
	}
	
	// Check straight attacks (rook, queen)
	for _, direction := range rookDirections {
		for i := 1; i < 8; i++ {
			to := int(kingSquare) + direction*i
			
			if to < 0 || to > 63 {
				break
			}
			
			// Check bounds wrapping for horizontal moves
			if direction == -1 || direction == 1 {
				fromFile := int(kingSquare) % 8
				toFile := to % 8
				if abs(fromFile-toFile) != i {
					break
				}
			}
			
			toSquare := Square(to)
			piece := pos.Board.PieceAt(toSquare)
			
			if piece != NoPiece {
				if piece.Color() == enemyColor && (piece.Type() == Rook || piece.Type() == Queen) {
					return true
				}
				break
			}
		}
	}
	
	return false
}

// Buffer-based move generation to eliminate allocations (safe optimization)

// GenerateMovesIntoBuffer generates moves into the provided buffer, returning the number of moves added
func GenerateMovesIntoBuffer(pos *Position, buffer []Move) int {
	moveCount := 0
	turn := pos.Turn()
	
	if turn == White {
		moveCount += generateWhitePawnMovesIntoBuffer(pos, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, WhiteKnight, generateKnightMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, WhiteBishop, generateBishopMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, WhiteRook, generateRookMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, WhiteQueen, generateQueenMovesIntoBuffer, buffer[moveCount:])
		moveCount += generateKingMovesIntoBuffer(pos, WhiteKing, buffer[moveCount:])
	} else {
		moveCount += generateBlackPawnMovesIntoBuffer(pos, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, BlackKnight, generateKnightMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, BlackBishop, generateBishopMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, BlackRook, generateRookMovesIntoBuffer, buffer[moveCount:])
		moveCount += generatePieceMovesIntoBuffer(pos, BlackQueen, generateQueenMovesIntoBuffer, buffer[moveCount:])
		moveCount += generateKingMovesIntoBuffer(pos, BlackKing, buffer[moveCount:])
	}
	
	return moveCount
}

// GenerateCapturesIntoBuffer generates captures into the provided buffer, returning the number of captures added
func GenerateCapturesIntoBuffer(pos *Position, buffer []Move) int {
	captureCount := 0
	turn := pos.Turn()
	
	if turn == White {
		captureCount += generateWhitePawnCapturesIntoBuffer(pos, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, WhiteKnight, generateKnightCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, WhiteBishop, generateBishopCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, WhiteRook, generateRookCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, WhiteQueen, generateQueenCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generateKingCapturesIntoBuffer(pos, WhiteKing, buffer[captureCount:])
	} else {
		captureCount += generateBlackPawnCapturesIntoBuffer(pos, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, BlackKnight, generateKnightCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, BlackBishop, generateBishopCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, BlackRook, generateRookCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generatePieceCapturesIntoBuffer(pos, BlackQueen, generateQueenCapturesIntoBuffer, buffer[captureCount:])
		captureCount += generateKingCapturesIntoBuffer(pos, BlackKing, buffer[captureCount:])
	}
	
	return captureCount
}

// Helper functions for buffer-based generation

func generatePieceMovesIntoBuffer(pos *Position, piece Piece, generator func(*Position, Square, []Move) int, buffer []Move) int {
	moveCount := 0
	pieceBitboard := pos.Board.GetBitboardOf(piece)
	
	for pieceBitboard != 0 {
		square := Square(bits.TrailingZeros64(pieceBitboard))
		moveCount += generator(pos, square, buffer[moveCount:])
		pieceBitboard &^= SquareMask[int(square)]
	}
	
	return moveCount
}

func generatePieceCapturesIntoBuffer(pos *Position, piece Piece, generator func(*Position, Square, []Move) int, buffer []Move) int {
	captureCount := 0
	pieceBitboard := pos.Board.GetBitboardOf(piece)
	
	for pieceBitboard != 0 {
		square := Square(bits.TrailingZeros64(pieceBitboard))
		captureCount += generator(pos, square, buffer[captureCount:])
		pieceBitboard &^= SquareMask[int(square)]
	}
	
	return captureCount
}

// Buffer-based piece generators (using existing move generation - safe fallback)

func generateKnightMovesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	count := 0
	attacks := KnightAttacks[from]
	piece := pos.Board.PieceAt(from)
	friendlyPieces := pos.Board.GetWhitePieces()
	if piece.Color() == Black {
		friendlyPieces = pos.Board.GetBlackPieces()
	}
	
	attacks &^= friendlyPieces
	
	for attacks != 0 {
		to := Square(bits.TrailingZeros64(attacks))
		capturedPiece := pos.Board.PieceAt(to)
		tag := MoveTag(0)
		if capturedPiece != NoPiece {
			tag = Capture
		}
		buffer[count] = NewMove(from, to, piece, capturedPiece, NoType, tag)
		count++
		attacks &^= SquareMask[int(to)]
	}
	
	return count
}

func generateBishopMovesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateBishopMoves(pos, from)
	copy(buffer, moves)
	return len(moves)
}

func generateRookMovesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateRookMoves(pos, from)
	copy(buffer, moves)
	return len(moves)
}

func generateQueenMovesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateQueenMoves(pos, from)
	copy(buffer, moves)
	return len(moves)
}

func generateKnightCapturesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	captures := make([]Move, 0, 8)
	attacks := KnightAttacks[from]
	piece := pos.Board.PieceAt(from)
	enemyPieces := pos.Board.GetBlackPieces()
	if piece.Color() == Black {
		enemyPieces = pos.Board.GetWhitePieces()
	}
	
	attacks &= enemyPieces
	
	for attacks != 0 {
		to := Square(bits.TrailingZeros64(attacks))
		capturedPiece := pos.Board.PieceAt(to)
		captures = append(captures, NewMove(from, to, piece, capturedPiece, NoType, Capture))
		attacks &^= SquareMask[int(to)]
	}
	
	copy(buffer, captures)
	return len(captures)
}

func generateBishopCapturesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateBishopMoves(pos, from)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}

func generateRookCapturesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateRookMoves(pos, from)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}

func generateQueenCapturesIntoBuffer(pos *Position, from Square, buffer []Move) int {
	moves := generateQueenMoves(pos, from)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}

func generateWhitePawnMovesIntoBuffer(pos *Position, buffer []Move) int {
	moves := generateWhitePawnMoves(pos)
	copy(buffer, moves)
	return len(moves)
}

func generateBlackPawnMovesIntoBuffer(pos *Position, buffer []Move) int {
	moves := generateBlackPawnMoves(pos)
	copy(buffer, moves)
	return len(moves)
}

func generateWhitePawnCapturesIntoBuffer(pos *Position, buffer []Move) int {
	moves := generateWhitePawnMoves(pos)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}

func generateBlackPawnCapturesIntoBuffer(pos *Position, buffer []Move) int {
	moves := generateBlackPawnMoves(pos)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}

func generateKingMovesIntoBuffer(pos *Position, piece Piece, buffer []Move) int {
	moves := generateKingMoves(pos, piece)
	copy(buffer, moves)
	return len(moves)
}

func generateKingCapturesIntoBuffer(pos *Position, piece Piece, buffer []Move) int {
	moves := generateKingMoves(pos, piece)
	count := 0
	for _, move := range moves {
		if move.IsCapture() {
			buffer[count] = move
			count++
		}
	}
	return count
}