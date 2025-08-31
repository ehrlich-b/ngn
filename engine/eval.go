package engine

import (
	"math/bits"
)

// Helper function to get set bits from a bitboard
func getSetBits(bitboard uint64) []int {
	var setBits []int
	for bitboard != 0 {
		bit := bits.TrailingZeros64(bitboard)
		setBits = append(setBits, bit)
		bitboard &= bitboard - 1 // Clear the lowest set bit
	}
	return setBits
}

// Helper function for trailing zeros
func trailingZeros(bitboard uint64) int {
	return bits.TrailingZeros64(bitboard)
}

// Helper function for absolute value (avoid redefinition)
func evalAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Piece-Square Tables for positional evaluation
// Values from white's perspective, mirrored for black

// Pawn piece-square table (encourages central pawns and pawn advancement)
var pawnPST = [64]int{
	0,   0,   0,   0,   0,   0,   0,   0,
	50,  50,  50,  50,  50,  50,  50,  50,
	10,  10,  20,  30,  30,  20,  10,  10,
	5,   5,   10,  25,  25,  10,  5,   5,
	0,   0,   0,   20,  20,  0,   0,   0,
	5,   -5,  -10, 0,   0,   -10, -5,  5,
	5,   10,  10,  -20, -20, 10,  10,  5,
	0,   0,   0,   0,   0,   0,   0,   0,
}

// Knight piece-square table (encourages central knight placement)
var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0,   0,   0,   0,   -20, -40,
	-30, 0,   10,  15,  15,  10,  0,   -30,
	-30, 5,   15,  20,  20,  15,  5,   -30,
	-30, 0,   15,  20,  20,  15,  0,   -30,
	-30, 5,   10,  15,  15,  10,  5,   -30,
	-40, -20, 0,   5,   5,   0,   -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

// Bishop piece-square table (encourages central placement and long diagonals)
var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 0,   0,   0,   0,   0,   0,   -10,
	-10, 0,   5,   10,  10,  5,   0,   -10,
	-10, 5,   5,   10,  10,  5,   5,   -10,
	-10, 0,   10,  10,  10,  10,  0,   -10,
	-10, 10,  10,  10,  10,  10,  10,  -10,
	-10, 5,   0,   0,   0,   0,   5,   -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

// Rook piece-square table (encourages central files and 7th rank)
var rookPST = [64]int{
	0,  0,  0,  0,  0,  0,  0,  0,
	5,  10, 10, 10, 10, 10, 10, 5,
	-5, 0,  0,  0,  0,  0,  0,  -5,
	-5, 0,  0,  0,  0,  0,  0,  -5,
	-5, 0,  0,  0,  0,  0,  0,  -5,
	-5, 0,  0,  0,  0,  0,  0,  -5,
	-5, 0,  0,  0,  0,  0,  0,  -5,
	0,  0,  0,  5,  5,  0,  0,  0,
}

// Queen piece-square table (slight centralization bonus)
var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0,   0,   0,  0,  0,   0,   -10,
	-10, 0,   5,   5,  5,  5,   0,   -10,
	-5,  0,   5,   5,  5,  5,   0,   -5,
	0,   0,   5,   5,  5,  5,   0,   -5,
	-10, 5,   5,   5,  5,  5,   0,   -10,
	-10, 0,   5,   0,  0,  0,   0,   -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

// King piece-square table for middlegame (safety first)
var kingMiddlegamePST = [64]int{
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	20,  20,  0,   0,   0,   0,   20,  20,
	20,  30,  10,  0,   0,   10,  30,  20,
}

// King piece-square table for endgame (centralization becomes important)
var kingEndgamePST = [64]int{
	-50, -40, -30, -20, -20, -30, -40, -50,
	-30, -20, -10, 0,   0,   -10, -20, -30,
	-30, -10, 20,  30,  30,  20,  -10, -30,
	-30, -10, 30,  40,  40,  30,  -10, -30,
	-30, -10, 30,  40,  40,  30,  -10, -30,
	-30, -10, 20,  30,  30,  20,  -10, -30,
	-30, -30, 0,   0,   0,   0,   -30, -30,
	-50, -30, -30, -30, -30, -30, -30, -50,
}

func getPieceSquareValue(piece Piece, square Square, isEndgame bool) int {
	color := piece.Color()
	var pstIndex int
	
	// Mirror the square for black pieces (flip vertically)
	if color == Black {
		pstIndex = int(square ^ 56) // XOR with 56 to flip rank
	} else {
		pstIndex = int(square)
	}
	
	var value int
	switch piece.Type() {
	case Pawn:
		value = pawnPST[pstIndex]
	case Knight:
		value = knightPST[pstIndex]
	case Bishop:
		value = bishopPST[pstIndex]
	case Rook:
		value = rookPST[pstIndex]
	case Queen:
		value = queenPST[pstIndex]
	case King:
		if isEndgame {
			value = kingEndgamePST[pstIndex]
		} else {
			value = kingMiddlegamePST[pstIndex]
		}
	}
	
	// Negate for black pieces
	if color == Black {
		value = -value
	}
	
	return value
}

// Helper function to check if we're in the endgame
// Simple heuristic: endgame when both sides have <= 13 points of material (excluding pawns and kings)
func isEndgame(board *Bitboard) bool {
	whiteMaterial := PopCount(board.GetBitboardOf(WhiteQueen))*1000 +
		PopCount(board.GetBitboardOf(WhiteRook))*525 +
		PopCount(board.GetBitboardOf(WhiteBishop))*330 +
		PopCount(board.GetBitboardOf(WhiteKnight))*320
	
	blackMaterial := PopCount(board.GetBitboardOf(BlackQueen))*1000 +
		PopCount(board.GetBitboardOf(BlackRook))*525 +
		PopCount(board.GetBitboardOf(BlackBishop))*330 +
		PopCount(board.GetBitboardOf(BlackKnight))*320
	
	return whiteMaterial <= 1300 && blackMaterial <= 1300
}

// Helper function to count doubled pawns on a file
func countDoubledPawns(board *Bitboard, color Color, file int) int {
	var pawnBitboard uint64
	if color == White {
		pawnBitboard = board.GetBitboardOf(WhitePawn)
	} else {
		pawnBitboard = board.GetBitboardOf(BlackPawn)
	}
	
	fileMask := uint64(0x0101010101010101) << file // File A to H
	pawnsOnFile := PopCount(pawnBitboard & fileMask)
	
	if pawnsOnFile > 1 {
		return pawnsOnFile - 1 // Return number of "extra" pawns (doubled)
	}
	return 0
}

// Helper function to count isolated pawns
func countIsolatedPawns(board *Bitboard, color Color) int {
	var pawnBitboard uint64
	if color == White {
		pawnBitboard = board.GetBitboardOf(WhitePawn)
	} else {
		pawnBitboard = board.GetBitboardOf(BlackPawn)
	}
	
	isolated := 0
	for file := 0; file < 8; file++ {
		fileMask := uint64(0x0101010101010101) << file
		if (pawnBitboard & fileMask) == 0 {
			continue // No pawns on this file
		}
		
		// Check adjacent files for supporting pawns
		hasSupport := false
		if file > 0 {
			leftFileMask := uint64(0x0101010101010101) << (file - 1)
			if (pawnBitboard & leftFileMask) != 0 {
				hasSupport = true
			}
		}
		if file < 7 {
			rightFileMask := uint64(0x0101010101010101) << (file + 1)
			if (pawnBitboard & rightFileMask) != 0 {
				hasSupport = true
			}
		}
		
		if !hasSupport {
			isolated += PopCount(pawnBitboard & fileMask)
		}
	}
	
	return isolated
}

// Helper function to count passed pawns (simplified version)
func countPassedPawns(board *Bitboard, color Color) int {
	var pawnBitboard uint64
	var opponentPawnBitboard uint64
	
	if color == White {
		pawnBitboard = board.GetBitboardOf(WhitePawn)
		opponentPawnBitboard = board.GetBitboardOf(BlackPawn)
	} else {
		pawnBitboard = board.GetBitboardOf(BlackPawn)
		opponentPawnBitboard = board.GetBitboardOf(WhitePawn)
	}
	
	passed := 0
	pawnSquares := getSetBits(pawnBitboard)
	
	for _, square := range pawnSquares {
		file := int(square % 8)
		rank := int(square / 8)
		
		// Define the "passed pawn" area based on color
		var blockedByOpponent bool
		
		// Check if any opponent pawns are in front of this pawn or on adjacent files
		for _, opponentSquare := range getSetBits(opponentPawnBitboard) {
			opponentFile := int(opponentSquare % 8)
			opponentRank := int(opponentSquare / 8)
			
			// Check if opponent pawn is in the path or adjacent files ahead
			if evalAbs(opponentFile-file) <= 1 {
				if color == White && opponentRank > rank {
					blockedByOpponent = true
					break
				} else if color == Black && opponentRank < rank {
					blockedByOpponent = true
					break
				}
			}
		}
		
		if !blockedByOpponent {
			passed++
		}
	}
	
	return passed
}



// King safety evaluation - checks squares around the king
func evaluateKingSafety(board *Bitboard, color Color) int {
	var kingBitboard uint64
	var pawnBitboard uint64
	
	if color == White {
		kingBitboard = board.GetBitboardOf(WhiteKing)
		pawnBitboard = board.GetBitboardOf(WhitePawn)
	} else {
		kingBitboard = board.GetBitboardOf(BlackKing)
		pawnBitboard = board.GetBitboardOf(BlackPawn)
	}
	
	if kingBitboard == 0 {
		return 0
	}
	
	kingSquare := trailingZeros(kingBitboard)
	kingFile := kingSquare % 8
	kingRank := kingSquare / 8
	
	safety := 0
	
	// Pawn shield bonus - check for pawns in front of king
	for fileOffset := -1; fileOffset <= 1; fileOffset++ {
		file := int(kingFile) + fileOffset
		if file < 0 || file > 7 {
			continue
		}
		
		// Check for pawns in front of king (1 or 2 squares ahead)
		for rankOffset := 1; rankOffset <= 2; rankOffset++ {
			var checkRank int
			if color == White {
				checkRank = int(kingRank) + rankOffset
			} else {
				checkRank = int(kingRank) - rankOffset
			}
			
			if checkRank < 0 || checkRank > 7 {
				continue
			}
			
			checkSquare := checkRank*8 + file
			if (pawnBitboard & (1 << checkSquare)) != 0 {
				if rankOffset == 1 {
					safety += 10 // Close pawn shield
				} else {
					safety += 5 // Distant pawn shield
				}
				break // Found pawn on this file
			}
		}
	}
	
	return safety
}


// Enhanced evaluation function
func Evaluate(board *Bitboard) (eval int) {
	
	endgame := isEndgame(board)
	
	// 1. Material counting with piece values
	for _, piece := range Pieces {
		if piece == NoPiece {
			continue
		}
		
		pieceBitboard := board.GetBitboardOf(piece)
		color := piece.Color()
		val := piece.Weight()
		colorMul := 1
		if color == Black {
			colorMul = -1
		}
		
		// Material value
		if piece.Type() != King {
			eval += colorMul * int(val) * PopCount(pieceBitboard)
		}
		
		// 2. Piece-square table values
		squares := getSetBits(pieceBitboard)
		for _, square := range squares {
			eval += getPieceSquareValue(piece, Square(square), endgame)
		}
	}
	
	// 3. Pawn structure evaluation
	whiteDoubledPawns := 0
	blackDoubledPawns := 0
	for file := 0; file < 8; file++ {
		whiteDoubledPawns += countDoubledPawns(board, White, file)
		blackDoubledPawns += countDoubledPawns(board, Black, file)
	}
	eval += -(whiteDoubledPawns * 50) // Penalty for doubled pawns
	eval += (blackDoubledPawns * 50)
	
	whiteIsolatedPawns := countIsolatedPawns(board, White)
	blackIsolatedPawns := countIsolatedPawns(board, Black)
	eval += -(whiteIsolatedPawns * 15) // Penalty for isolated pawns
	eval += (blackIsolatedPawns * 15)
	
	whitePassedPawns := countPassedPawns(board, White)
	blackPassedPawns := countPassedPawns(board, Black)
	eval += whitePassedPawns * 25 // Bonus for passed pawns
	eval += -(blackPassedPawns * 25)
	
	// 4. King safety evaluation
	whiteKingSafety := evaluateKingSafety(board, White)
	blackKingSafety := evaluateKingSafety(board, Black)
	if !endgame { // King safety more important in middlegame
		eval += whiteKingSafety
		eval += -blackKingSafety
	}
	
	
	return eval
}