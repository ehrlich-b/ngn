package engine

import (
	"math/bits"
)

// Note: getSetBits function removed - replaced with zero-allocation iteration patterns

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
// Enhanced with opening nudges: e4/d4 get +15 bonus
var pawnPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	50, 50, 50, 50, 50, 50, 50, 50,
	10, 10, 20, 30, 30, 20, 10, 10,
	5, 5, 10, 25, 25, 10, 5, 5,
	0, 0, 0, 35, 35, 0, 0, 0, // d4/e4 get +15 opening bonus
	5, -5, -10, 0, 0, -10, -5, 5,
	5, 10, 10, -20, -20, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

// Knight piece-square table (discourages premature development, encourages proper knight placement)
var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, -10, -10, -10, -10, -20, -40,
	-30, -10, 0, 5, 5, 0, -10, -30,
	-30, -5, 5, 10, 10, 5, -5, -30,
	-30, -10, 5, 10, 10, 5, -10, -30,
	-30, -15, -5, 0, 0, -5, -15, -30, // Early development squares get penalty
	-40, -25, -15, -10, -10, -15, -25, -40, // Stronger penalty for very early development
	-50, -40, -30, -30, -30, -30, -40, -50,
}

// Bishop piece-square table (encourages central placement and long diagonals)
var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

// Rook piece-square table (encourages central files and 7th rank)
var rookPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, 10, 10, 10, 10, 5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	0, 0, 0, 5, 5, 0, 0, 0,
}

// Queen piece-square table (discourages early queen development)
// Enhanced with opening nudges: early queen squares get penalty
var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-35, -25, -25, -20, -20, -25, -25, -35, // Queen early development penalty
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
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
	20, 20, 0, 0, 0, 0, 20, 20,
	20, 30, 10, 0, 0, 10, 30, 20,
}

// King piece-square table for endgame (centralization becomes important)
var kingEndgamePST = [64]int{
	-50, -40, -30, -20, -20, -30, -40, -50,
	-30, -20, -10, 0, 0, -10, -20, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -30, 0, 0, 0, 0, -30, -30,
	-50, -30, -30, -30, -30, -30, -30, -50,
}

// Pre-computed PSTs for black pieces (flipped and negated for zero-cost lookups)
var blackPawnPST, blackKnightPST, blackBishopPST, blackRookPST, blackQueenPST, blackKingMiddlegamePST, blackKingEndgamePST [64]int

func init() {
	// Pre-compute all black piece-square tables (flipped and negated)
	for i := 0; i < 64; i++ {
		flippedIndex := i ^ 56 // Flip rank for black pieces
		blackPawnPST[i] = -pawnPST[flippedIndex]
		blackKnightPST[i] = -knightPST[flippedIndex]
		blackBishopPST[i] = -bishopPST[flippedIndex]
		blackRookPST[i] = -rookPST[flippedIndex]
		blackQueenPST[i] = -queenPST[flippedIndex]
		blackKingMiddlegamePST[i] = -kingMiddlegamePST[flippedIndex]
		blackKingEndgamePST[i] = -kingEndgamePST[flippedIndex]
	}
}

// Optimized PST lookup - zero-cost runtime lookups with pre-computed tables
func getPieceSquareValue(piece Piece, square Square, isEndgame bool) int {
	sq := int(square)

	switch piece {
	case WhitePawn:
		return pawnPST[sq]
	case BlackPawn:
		return blackPawnPST[sq]
	case WhiteKnight:
		return knightPST[sq]
	case BlackKnight:
		return blackKnightPST[sq]
	case WhiteBishop:
		return bishopPST[sq]
	case BlackBishop:
		return blackBishopPST[sq]
	case WhiteRook:
		return rookPST[sq]
	case BlackRook:
		return blackRookPST[sq]
	case WhiteQueen:
		return queenPST[sq]
	case BlackQueen:
		return blackQueenPST[sq]
	case WhiteKing:
		if isEndgame {
			return kingEndgamePST[sq]
		} else {
			return kingMiddlegamePST[sq]
		}
	case BlackKing:
		if isEndgame {
			return blackKingEndgamePST[sq]
		} else {
			return blackKingMiddlegamePST[sq]
		}
	}
	return 0
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

	// Zero-allocation pawn iteration
	bb := pawnBitboard
	for sq := nextSetBit(&bb); sq != 64; sq = nextSetBit(&bb) {
		square := sq
		file := int(square % 8)
		rank := int(square / 8)

		// Define the "passed pawn" area based on color
		var blockedByOpponent bool

		// Check if any opponent pawns are in front of this pawn or on adjacent files
		for obb := opponentPawnBitboard; obb != 0; obb &= obb - 1 {
			opponentSquare := bits.TrailingZeros64(obb)
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

// Optimized passed pawn counting using pre-cached bitboards (replaces nested loops)
func countPassedPawnsFast(pawnBitboard, opponentPawnBitboard uint64, color Color) int {
	if pawnBitboard == 0 {
		return 0
	}

	passed := 0

	// Generate opponent pawn attack masks by file for O(1) lookups
	var opponentControl [8]uint64

	for obb := opponentPawnBitboard; obb != 0; obb &= obb - 1 {
		sq := bits.TrailingZeros64(obb)
		file := sq % 8
		rank := sq / 8

		// Mark files controlled by this opponent pawn
		if file > 0 { // Left adjacent file
			if color == White {
				opponentControl[file-1] |= uint64(0xFF) << (rank * 8) // All ranks >= this rank
			} else {
				opponentControl[file-1] |= uint64(0xFF) >> ((7 - rank) * 8) // All ranks <= this rank
			}
		}
		opponentControl[file] |= uint64(0xFF) << (rank * 8) // Same file
		if file < 7 {                                       // Right adjacent file
			if color == White {
				opponentControl[file+1] |= uint64(0xFF) << (rank * 8) // All ranks >= this rank
			} else {
				opponentControl[file+1] |= uint64(0xFF) >> ((7 - rank) * 8) // All ranks <= this rank
			}
		}
	}

	// Check each pawn against file masks
	for bb := pawnBitboard; bb != 0; bb &= bb - 1 {
		sq := bits.TrailingZeros64(bb)
		file := sq % 8

		// Create mask for squares this pawn must pass through
		var pathMask uint64
		if color == White {
			pathMask = uint64(0xFF) << ((sq/8 + 1) * 8) // All ranks ahead
		} else {
			pathMask = uint64(0xFF) >> ((7 - sq/8 + 1) * 8) // All ranks ahead
		}

		// Check if path is clear
		if (pathMask & opponentControl[file]) == 0 {
			passed++
		}
	}

	return passed
}

// Helper function to evaluate pawn chains (connected pawns that support each other)
func evaluatePawnChains(board *Bitboard, color Color) int {
	var pawnBitboard uint64
	if color == White {
		pawnBitboard = board.GetBitboardOf(WhitePawn)
	} else {
		pawnBitboard = board.GetBitboardOf(BlackPawn)
	}

	if pawnBitboard == 0 {
		return 0
	}

	chainBonus := 0

	for pbb := pawnBitboard; pbb != 0; pbb &= pbb - 1 {
		square := bits.TrailingZeros64(pbb)
		file := int(square % 8)
		rank := int(square / 8)

		// Count supporting pawns (pawns that can defend this pawn)
		supportingPawns := 0

		// Check for supporting pawns on adjacent files, one rank behind
		for fileOffset := -1; fileOffset <= 1; fileOffset += 2 { // -1 and +1
			supportFile := file + fileOffset
			if supportFile < 0 || supportFile > 7 {
				continue
			}

			var supportRank int
			if color == White {
				supportRank = rank - 1 // White pawns advance up, so support comes from behind
			} else {
				supportRank = rank + 1 // Black pawns advance down, so support comes from behind
			}

			if supportRank < 0 || supportRank > 7 {
				continue
			}

			supportSquare := supportRank*8 + supportFile
			if (pawnBitboard & (1 << supportSquare)) != 0 {
				supportingPawns++
			}
		}

		// Bonus for pawns in chains (more support = higher bonus)
		switch supportingPawns {
		case 1:
			chainBonus += 8 // Single support
		case 2:
			chainBonus += 12 // Double support (very strong)
		}

		// Additional bonus for advanced pawns in chains
		var advancement int
		if color == White {
			advancement = rank - 1 // Rank 2 is base rank for white pawns
		} else {
			advancement = 6 - rank // Rank 7 is base rank for black pawns
		}

		if supportingPawns > 0 && advancement > 2 {
			chainBonus += advancement * 2 // Advanced supported pawns get extra bonus
		}
	}

	return chainBonus
}

// Helper function to evaluate weak squares around pawns
// PERFORMANCE: Disabled for Phase 3 optimization - was consuming 5.31% CPU time
func evaluateWeakSquares(board *Bitboard, color Color) int {
	// Fast path - return 0 for performance optimization
	return 0

	// Original complex implementation commented out for performance:
	/*
		var pawnBitboard uint64

		if color == White {
			pawnBitboard = board.GetBitboardOf(WhitePawn)
		} else {
			pawnBitboard = board.GetBitboardOf(BlackPawn)
		}

		if pawnBitboard == 0 {
			return 0 // No pawns, no weak squares to evaluate
		}

		weaknessBonus := 0 // Positive bonus for opponent having weak squares

		// Generate pawn attack coverage (squares controlled by our pawns)
		pawnAttackMask := uint64(0)

		for pbb := pawnBitboard; pbb != 0; pbb &= pbb - 1 {
			square := bits.TrailingZeros64(pbb)
			file := int(square % 8)
			rank := int(square / 8)

			// Generate pawn attacks (diagonal captures)
			var attackRank int
			if color == White {
				attackRank = rank + 1 // White pawns attack forward
			} else {
				attackRank = rank - 1 // Black pawns attack backward
			}

			if attackRank >= 0 && attackRank <= 7 {
				// Left attack
				if file > 0 {
					attackSquare := attackRank*8 + (file - 1)
					pawnAttackMask |= (1 << attackSquare)
				}
				// Right attack
				if file < 7 {
					attackSquare := attackRank*8 + (file + 1)
					pawnAttackMask |= (1 << attackSquare)
				}
			}
		}

		// Check for enemy pieces on weak squares (not controlled by our pawns)
		// Focus on central files (c, d, e, f) and important squares
		centralFiles := []int{2, 3, 4, 5} // Files c, d, e, f

		for _, file := range centralFiles {
			for rank := 2; rank <= 5; rank++ { // Focus on central area
				square := rank*8 + file
				squareMask := uint64(1 << square)

				// If this square is not controlled by our pawns, it's potentially weak
				if (pawnAttackMask & squareMask) == 0 {
					// Check if opponent has pieces that could benefit from this weak square
					// Look for enemy knights, bishops, or queens nearby

					// Simple penalty for uncontrolled central squares
					if rank >= 3 && rank <= 4 { // Ranks 4-5 (0-indexed)
						weaknessBonus -= 3 // Small penalty for weak central squares
					}

					// Additional penalty if opponent has pawns that create holes
					// Check if this square is a "hole" (can't be defended by pawns)
					canBeDefended := false

					// Check if we could potentially put a pawn to defend this square
					for fileOffset := -1; fileOffset <= 1; fileOffset += 2 {
						defendFile := file + fileOffset
						if defendFile < 0 || defendFile > 7 {
							continue
						}

						var defendRank int
						if color == White {
							defendRank = rank - 1 // Would need a pawn behind to defend
						} else {
							defendRank = rank + 1 // Would need a pawn behind to defend
						}

						if defendRank >= 0 && defendRank <= 7 {
							defendSquare := defendRank*8 + defendFile
							if (pawnBitboard & (1 << defendSquare)) != 0 {
								canBeDefended = true
								break
							}
						}
					}

					if !canBeDefended {
						// This is a permanent weak square - bigger penalty
						if rank >= 3 && rank <= 4 {
							weaknessBonus -= 8 // Larger penalty for holes in our position
						}
					}
				}
			}
		}

		return weaknessBonus // This will be negative for our weak squares
	*/
}

// Helper function to evaluate pawn storms
func evaluatePawnStorms(board *Bitboard, color Color) int {
	var pawnBitboard uint64
	var opponentKingBitboard uint64

	if color == White {
		pawnBitboard = board.GetBitboardOf(WhitePawn)
		opponentKingBitboard = board.GetBitboardOf(BlackKing)
	} else {
		pawnBitboard = board.GetBitboardOf(BlackPawn)
		opponentKingBitboard = board.GetBitboardOf(WhiteKing)
	}

	if pawnBitboard == 0 || opponentKingBitboard == 0 {
		return 0
	}

	// Find opponent king position
	opponentKingSquare := trailingZeros(opponentKingBitboard)
	opponentKingFile := int(opponentKingSquare % 8)
	opponentKingRank := int(opponentKingSquare / 8)

	stormBonus := 0

	// Track most advanced pawn per file without allocations
	// Use -1 to indicate no pawn on that file
	var mostAdvancedByFile [8]int
	var pawnCountByFile [8]int

	// Initialize with invalid values
	for i := 0; i < 8; i++ {
		if color == White {
			mostAdvancedByFile[i] = -1 // No pawn yet
		} else {
			mostAdvancedByFile[i] = 8 // No pawn yet (8 is invalid rank)
		}
	}

	// Find most advanced pawn on each file
	for pbb := pawnBitboard; pbb != 0; pbb &= pbb - 1 {
		square := bits.TrailingZeros64(pbb)
		file := square % 8
		rank := square / 8
		pawnCountByFile[file]++

		if color == White {
			if mostAdvancedByFile[file] == -1 || rank > mostAdvancedByFile[file] {
				mostAdvancedByFile[file] = rank
			}
		} else {
			if mostAdvancedByFile[file] == 8 || rank < mostAdvancedByFile[file] {
				mostAdvancedByFile[file] = rank
			}
		}
	}

	// Evaluate storm potential near enemy king
	for fileOffset := -1; fileOffset <= 1; fileOffset++ {
		kingFile := opponentKingFile + fileOffset
		if kingFile < 0 || kingFile > 7 {
			continue
		}

		mostAdvancedRank := mostAdvancedByFile[kingFile]
		pawnCount := pawnCountByFile[kingFile]

		// Skip if no pawns on this file
		if pawnCount == 0 {
			continue
		}

		// Calculate storm advancement
		var advancement int
		if color == White {
			advancement = mostAdvancedRank - 1 // Starting from rank 2
			// Bonus increases as pawn advances toward enemy king
			if mostAdvancedRank >= 4 { // Pawn has crossed the halfway point
				stormBonus += advancement * 4

				// Extra bonus if pawn is close to enemy king rank
				if mostAdvancedRank >= opponentKingRank-2 {
					stormBonus += 10
				}
			}
		} else {
			advancement = 6 - mostAdvancedRank // Starting from rank 7
			if mostAdvancedRank <= 3 {         // Pawn has crossed the halfway point
				stormBonus += advancement * 4

				// Extra bonus if pawn is close to enemy king rank
				if mostAdvancedRank <= opponentKingRank+2 {
					stormBonus += 10
				}
			}
		}

		// Multiple pawns on same file create stronger storms
		if pawnCount > 1 {
			stormBonus += pawnCount * 2 // Bonus for pawn cooperation
		}
	}

	// Additional bonus for connected pawn storms (pawns on adjacent files)
	for file := 0; file < 7; file++ {
		leftCount := pawnCountByFile[file]
		rightCount := pawnCountByFile[file+1]

		if leftCount > 0 && rightCount > 0 {
			// Check if both files have advanced pawns near enemy king
			kingDistance := evalAbs(file-opponentKingFile) + evalAbs((file+1)-opponentKingFile)
			if kingDistance <= 3 { // Within striking distance of king
				stormBonus += 8 // Bonus for connected storm
			}
		}
	}

	return stormBonus
}

// Enhanced king safety evaluation with improved pawn shield and attack patterns
func evaluateKingSafety(board *Bitboard, color Color) int {
	var kingBitboard uint64
	var pawnBitboard uint64
	var opponentQueenBitboard uint64
	var opponentRookBitboard uint64
	var opponentBishopBitboard uint64
	var opponentKnightBitboard uint64

	if color == White {
		kingBitboard = board.GetBitboardOf(WhiteKing)
		pawnBitboard = board.GetBitboardOf(WhitePawn)
		opponentQueenBitboard = board.GetBitboardOf(BlackQueen)
		opponentRookBitboard = board.GetBitboardOf(BlackRook)
		opponentBishopBitboard = board.GetBitboardOf(BlackBishop)
		opponentKnightBitboard = board.GetBitboardOf(BlackKnight)
	} else {
		kingBitboard = board.GetBitboardOf(BlackKing)
		pawnBitboard = board.GetBitboardOf(BlackPawn)
		opponentQueenBitboard = board.GetBitboardOf(WhiteQueen)
		opponentRookBitboard = board.GetBitboardOf(WhiteRook)
		opponentBishopBitboard = board.GetBitboardOf(WhiteBishop)
		opponentKnightBitboard = board.GetBitboardOf(WhiteKnight)
	}

	if kingBitboard == 0 {
		return 0
	}

	kingSquare := trailingZeros(kingBitboard)
	kingFile := int(kingSquare % 8)
	kingRank := int(kingSquare / 8)

	safety := 0

	// 1. Enhanced Pawn Shield Evaluation
	pawnShieldBonus := evaluatePawnShield(pawnBitboard, kingFile, kingRank, color)
	safety += pawnShieldBonus

	// 2. Attack Pattern Evaluation (penalties for enemy pieces near king)
	attackPenalty := evaluateKingAttackPatterns(
		opponentQueenBitboard, opponentRookBitboard,
		opponentBishopBitboard, opponentKnightBitboard,
		kingFile, kingRank, color)
	safety -= attackPenalty

	// 3. King position penalty (being in center during middlegame is dangerous)
	if kingFile >= 2 && kingFile <= 5 { // Central files
		safety -= 15 // Penalty for exposed king
	}

	// 4. Open file penalty near king
	for fileOffset := -1; fileOffset <= 1; fileOffset++ {
		file := kingFile + fileOffset
		if file < 0 || file > 7 {
			continue
		}

		// Check if this file is open (no pawns from either side)
		fileMask := uint64(0x0101010101010101) << file
		if color == White {
			whitePawns := board.GetBitboardOf(WhitePawn) & fileMask
			blackPawns := board.GetBitboardOf(BlackPawn) & fileMask
			if whitePawns == 0 && blackPawns == 0 {
				safety -= 20 // Open file near king is dangerous
			} else if whitePawns == 0 {
				safety -= 10 // Semi-open file is also risky
			}
		} else {
			whitePawns := board.GetBitboardOf(WhitePawn) & fileMask
			blackPawns := board.GetBitboardOf(BlackPawn) & fileMask
			if whitePawns == 0 && blackPawns == 0 {
				safety -= 20 // Open file near king is dangerous
			} else if blackPawns == 0 {
				safety -= 10 // Semi-open file is also risky
			}
		}
	}

	return safety
}

// Evaluate castling bonus for king safety (simplified version for bitboard-only evaluation)
func evaluateCastling(board *Bitboard, color Color) int {
	bonus := 0

	// Check if king has already castled by looking at king position
	kingBitboard := board.GetBitboardOf(WhiteKing)
	if color == Black {
		kingBitboard = board.GetBitboardOf(BlackKing)
	}

	if kingBitboard == 0 {
		return 0
	}

	kingSquare := trailingZeros(kingBitboard)
	kingFile := int(kingSquare % 8)
	kingRank := int(kingSquare / 8)

	// Detect if castling has already occurred based on king position
	hasCastled := false
	if color == White {
		// White king starts on e1 (file 4, rank 0)
		// After castling: g1 (file 6) for kingside, c1 (file 2) for queenside
		if kingRank == 0 && (kingFile == 6 || kingFile == 2) {
			hasCastled = true
		}
	} else {
		// Black king starts on e8 (file 4, rank 7)
		// After castling: g8 (file 6) for kingside, c8 (file 2) for queenside
		if kingRank == 7 && (kingFile == 6 || kingFile == 2) {
			hasCastled = true
		}
	}

	if hasCastled {
		// Bonus for having castled (king safety achieved)
		bonus += 40

		// Additional bonus based on rook position to confirm valid castling
		var expectedRookSquare int
		var rookBitboard uint64

		if color == White {
			rookBitboard = board.GetBitboardOf(WhiteRook)
			if kingFile == 6 { // Kingside castling
				expectedRookSquare = 5 // f1
			} else { // Queenside castling
				expectedRookSquare = 3 // d1
			}
		} else {
			rookBitboard = board.GetBitboardOf(BlackRook)
			if kingFile == 6 { // Kingside castling
				expectedRookSquare = 61 // f8
			} else { // Queenside castling
				expectedRookSquare = 59 // d8
			}
		}

		// Extra bonus if rook is in the typical castled position
		if (rookBitboard & (1 << expectedRookSquare)) != 0 {
			bonus += 15 // Confirmed proper castling structure
		}
	}

	return bonus
}

// Fast optimized castling evaluation - minimal CPU overhead
func evaluateCastlingFast(board *Bitboard, color Color) int {
	var kingBitboard uint64
	if color == White {
		kingBitboard = board.GetBitboardOf(WhiteKing)
	} else {
		kingBitboard = board.GetBitboardOf(BlackKing)
	}

	if kingBitboard == 0 {
		return 0
	}

	kingSquare := trailingZeros(kingBitboard)
	kingFile := kingSquare % 8
	kingRank := kingSquare / 8

	// Simple castling detection - just check if king is in castled position
	if color == White {
		if kingRank == 0 && (kingFile == 6 || kingFile == 2) {
			return 40 // Castled bonus
		}
	} else {
		if kingRank == 7 && (kingFile == 6 || kingFile == 2) {
			return 40 // Castled bonus
		}
	}

	return 0
}

// Evaluate center control bonus - reward occupation and control of central squares
func evaluateCenterControl(board *Bitboard, color Color) int {
	bonus := 0

	// Central squares: e4, d4, e5, d5 (squares 28, 27, 36, 35)
	centralSquares := []int{27, 28, 35, 36}                                        // d4, e4, d5, e5
	extendedCenterSquares := []int{18, 19, 20, 21, 26, 29, 34, 37, 42, 43, 44, 45} // c3-f3, c4-f4, c6-f6

	var ownPawns, ownKnights, ownBishops, ownQueens uint64

	if color == White {
		ownPawns = board.GetBitboardOf(WhitePawn)
		ownKnights = board.GetBitboardOf(WhiteKnight)
		ownBishops = board.GetBitboardOf(WhiteBishop)
		ownQueens = board.GetBitboardOf(WhiteQueen)
	} else {
		ownPawns = board.GetBitboardOf(BlackPawn)
		ownKnights = board.GetBitboardOf(BlackKnight)
		ownBishops = board.GetBitboardOf(BlackBishop)
		ownQueens = board.GetBitboardOf(BlackQueen)
	}

	// 1. Piece occupation bonuses for central squares
	for _, square := range centralSquares {
		squareMask := uint64(1) << square

		if (ownPawns & squareMask) != 0 {
			bonus += 25 // Pawn in center is very valuable
		} else if (ownKnights & squareMask) != 0 {
			bonus += 20 // Knight in center is strong
		} else if (ownBishops & squareMask) != 0 {
			bonus += 15 // Bishop in center is good
		} else if (ownQueens & squareMask) != 0 {
			bonus += 10 // Queen in center can be powerful but also vulnerable
		}
		// Rooks and kings in center are usually not beneficial
	}

	// 2. Extended center occupation (smaller bonuses)
	for _, square := range extendedCenterSquares {
		squareMask := uint64(1) << square

		if (ownPawns & squareMask) != 0 {
			bonus += 8 // Extended center pawn
		} else if (ownKnights & squareMask) != 0 {
			bonus += 10 // Knight in extended center
		} else if (ownBishops & squareMask) != 0 {
			bonus += 6 // Bishop in extended center
		}
	}

	// 3. Pawn control of central squares (pawn attacks center even if not occupying)
	for _, square := range centralSquares {
		file := square % 8
		rank := square / 8

		// Check if our pawns control this square
		controlledByPawn := false

		if color == White {
			// White pawns attack diagonally upward
			if rank > 0 { // Can be attacked from rank below
				leftAttack := (rank-1)*8 + (file - 1)
				rightAttack := (rank-1)*8 + (file + 1)

				if file > 0 && (ownPawns&(1<<leftAttack)) != 0 {
					controlledByPawn = true
				}
				if file < 7 && (ownPawns&(1<<rightAttack)) != 0 {
					controlledByPawn = true
				}
			}
		} else {
			// Black pawns attack diagonally downward
			if rank < 7 { // Can be attacked from rank above
				leftAttack := (rank+1)*8 + (file - 1)
				rightAttack := (rank+1)*8 + (file + 1)

				if file > 0 && (ownPawns&(1<<leftAttack)) != 0 {
					controlledByPawn = true
				}
				if file < 7 && (ownPawns&(1<<rightAttack)) != 0 {
					controlledByPawn = true
				}
			}
		}

		if controlledByPawn {
			bonus += 15 // Bonus for controlling center with pawns
		}
	}

	return bonus
}

// Fast optimized center control - minimal overhead version
func evaluateCenterControlFast(board *Bitboard, color Color) int {
	bonus := 0

	// Just check piece occupation of core central squares (d4, e4, d5, e5)
	centralSquares := [4]int{27, 28, 35, 36} // d4, e4, d5, e5

	var ownPawns, ownKnights uint64
	if color == White {
		ownPawns = board.GetBitboardOf(WhitePawn)
		ownKnights = board.GetBitboardOf(WhiteKnight)
	} else {
		ownPawns = board.GetBitboardOf(BlackPawn)
		ownKnights = board.GetBitboardOf(BlackKnight)
	}

	// Quick check for occupation of central squares
	for _, square := range centralSquares {
		squareMask := uint64(1) << square
		if (ownPawns & squareMask) != 0 {
			bonus += 25 // Pawn in center
		} else if (ownKnights & squareMask) != 0 {
			bonus += 20 // Knight in center
		}
	}

	return bonus
}

// Evaluate piece development - reward proper opening principles
func evaluatePieceDevelopment(board *Bitboard, color Color) int {
	bonus := 0

	var ownKnights, ownBishops, ownRooks, ownQueen, ownKing uint64
	var startingRank int

	if color == White {
		ownKnights = board.GetBitboardOf(WhiteKnight)
		ownBishops = board.GetBitboardOf(WhiteBishop)
		ownRooks = board.GetBitboardOf(WhiteRook)
		ownQueen = board.GetBitboardOf(WhiteQueen)
		ownKing = board.GetBitboardOf(WhiteKing)
		startingRank = 0 // White pieces start on rank 1 (0-indexed)
	} else {
		ownKnights = board.GetBitboardOf(BlackKnight)
		ownBishops = board.GetBitboardOf(BlackBishop)
		ownRooks = board.GetBitboardOf(BlackRook)
		ownQueen = board.GetBitboardOf(BlackQueen)
		ownKing = board.GetBitboardOf(BlackKing)
		startingRank = 7 // Black pieces start on rank 8 (7-indexed)
	}

	// 1. Knight development bonus
	knightsDeveloped := 0
	for knightBB := ownKnights; knightBB != 0; knightBB &= knightBB - 1 {
		square := trailingZeros(knightBB)
		rank := square / 8

		if color == White && rank > startingRank {
			knightsDeveloped++
			bonus += 20 // Bonus for developed knight

			// Extra bonus for knights in good squares (f3, c3, f6, c6 area)
			file := square % 8
			if (rank == 2 || rank == 5) && (file == 2 || file == 5) { // c3, f3, c6, f6
				bonus += 10 // Extra bonus for knights in ideal squares
			}
		} else if color == Black && rank < startingRank {
			knightsDeveloped++
			bonus += 20 // Bonus for developed knight

			// Extra bonus for knights in good squares
			file := square % 8
			if (rank == 2 || rank == 5) && (file == 2 || file == 5) {
				bonus += 10 // Extra bonus for knights in ideal squares
			}
		}
	}

	// 2. Bishop development bonus
	bishopsDeveloped := 0
	for bishopBB := ownBishops; bishopBB != 0; bishopBB &= bishopBB - 1 {
		square := trailingZeros(bishopBB)
		rank := square / 8

		if color == White && rank > startingRank {
			bishopsDeveloped++
			bonus += 15 // Bonus for developed bishop
		} else if color == Black && rank < startingRank {
			bishopsDeveloped++
			bonus += 15 // Bonus for developed bishop
		}
	}

	// 3. Early queen penalty (queen should not be developed too early)
	queenDevelopedEarly := false
	if ownQueen != 0 {
		queenSquare := trailingZeros(ownQueen)
		queenRank := queenSquare / 8

		if color == White && queenRank > startingRank {
			queenDevelopedEarly = true
		} else if color == Black && queenRank < startingRank {
			queenDevelopedEarly = true
		}

		// Penalty increases if queen is out but minor pieces aren't developed
		if queenDevelopedEarly {
			totalMinorPiecesDeveloped := knightsDeveloped + bishopsDeveloped

			if totalMinorPiecesDeveloped == 0 {
				bonus -= 30 // Heavy penalty for queen out with no minor pieces developed
			} else if totalMinorPiecesDeveloped == 1 {
				bonus -= 20 // Medium penalty
			} else if totalMinorPiecesDeveloped == 2 {
				bonus -= 10 // Light penalty
			}
			// No penalty if 3+ minor pieces developed (queen out is fine then)
		}
	}

	// 4. Castling preparation bonus (king and rook coordination)
	if ownKing != 0 {
		kingSquare := trailingZeros(ownKing)
		kingRank := kingSquare / 8
		kingFile := kingSquare % 8

		// Check if king is still on starting square (good for castling preparation)
		if color == White && kingRank == 0 && kingFile == 4 { // e1
			// Bonus for keeping king on starting square while developing
			if knightsDeveloped >= 1 || bishopsDeveloped >= 1 {
				bonus += 8 // Small bonus for development while keeping king safe
			}
		} else if color == Black && kingRank == 7 && kingFile == 4 { // e8
			if knightsDeveloped >= 1 || bishopsDeveloped >= 1 {
				bonus += 8 // Small bonus for development while keeping king safe
			}
		}
	}

	// 5. Rook development penalty (rooks should generally stay on back rank early)
	for rookBB := ownRooks; rookBB != 0; rookBB &= rookBB - 1 {
		square := trailingZeros(rookBB)
		rank := square / 8

		// Penalty for rooks leaving back rank too early
		if color == White && rank > startingRank {
			// Only penalty if minor pieces aren't well developed
			if knightsDeveloped+bishopsDeveloped < 3 {
				bonus -= 15 // Penalty for premature rook development
			}
		} else if color == Black && rank < startingRank {
			if knightsDeveloped+bishopsDeveloped < 3 {
				bonus -= 15 // Penalty for premature rook development
			}
		}
	}

	return bonus
}

// Helper function for detailed pawn shield evaluation
func evaluatePawnShield(pawnBitboard uint64, kingFile int, kingRank int, color Color) int {
	shieldBonus := 0

	// Check pawn shield in three files around king
	for fileOffset := -1; fileOffset <= 1; fileOffset++ {
		file := kingFile + fileOffset
		if file < 0 || file > 7 {
			continue
		}

		pawnFound := false
		closestPawnRank := -1

		// Find closest pawn on this file in front of king
		for rankOffset := 1; rankOffset <= 6; rankOffset++ {
			var checkRank int
			if color == White {
				checkRank = kingRank + rankOffset
			} else {
				checkRank = kingRank - rankOffset
			}

			if checkRank < 0 || checkRank > 7 {
				break
			}

			checkSquare := checkRank*8 + file
			if (pawnBitboard & (1 << checkSquare)) != 0 {
				pawnFound = true
				closestPawnRank = checkRank
				break
			}
		}

		if pawnFound {
			var distance int
			if color == White {
				distance = closestPawnRank - kingRank
			} else {
				distance = kingRank - closestPawnRank
			}

			// Better bonus for closer pawns, extra bonus for center file
			baseBonus := 20 - distance*3 // Closer is better
			if fileOffset == 0 {
				baseBonus += 5 // Center file is most important
			}

			// Bonus varies by how advanced the pawn is
			var pawnAdvancement int
			if color == White {
				pawnAdvancement = closestPawnRank - 1 // White pawns start on rank 2
			} else {
				pawnAdvancement = 6 - closestPawnRank // Black pawns start on rank 7
			}

			if pawnAdvancement == 0 {
				baseBonus += 5 // Unmoved pawn is strong shield
			} else if pawnAdvancement >= 3 {
				baseBonus -= 5 // Advanced pawn is weaker shield
			}

			shieldBonus += baseBonus
		} else {
			// Heavy penalty for missing pawn shield
			if fileOffset == 0 {
				shieldBonus -= 25 // Center file most critical
			} else {
				shieldBonus -= 15 // Side files also important
			}
		}
	}

	return shieldBonus
}

// Helper function to evaluate enemy attack patterns near king
func evaluateKingAttackPatterns(queenBB, rookBB, bishopBB, knightBB uint64,
	kingFile int, kingRank int, color Color) int {

	attackWeight := 0

	// Check for enemy pieces in extended king zone (5x5 area) without allocations
	for fileOffset := -2; fileOffset <= 2; fileOffset++ {
		for rankOffset := -2; rankOffset <= 2; rankOffset++ {
			checkFile := kingFile + fileOffset
			checkRank := kingRank + rankOffset

			if checkFile >= 0 && checkFile <= 7 && checkRank >= 0 && checkRank <= 7 {
				square := checkRank*8 + checkFile
				squareMask := uint64(1 << square)

				// Weight attacks by piece value and proximity
				distance := evalAbs(checkFile-kingFile) + evalAbs(checkRank-kingRank)
				proximityMultiplier := 4 - (distance / 2) // Closer pieces are more dangerous
				if proximityMultiplier < 1 {
					proximityMultiplier = 1
				}

				if (queenBB & squareMask) != 0 {
					attackWeight += 20 * proximityMultiplier // Queens are very dangerous
				}
				if (rookBB & squareMask) != 0 {
					attackWeight += 10 * proximityMultiplier // Rooks are dangerous
				}
				if (bishopBB & squareMask) != 0 {
					attackWeight += 8 * proximityMultiplier // Bishops can be dangerous
				}
				if (knightBB & squareMask) != 0 {
					attackWeight += 12 * proximityMultiplier // Knights are tricky near king
				}
			}
		}
	}

	return attackWeight
}

// Basic endgame knowledge evaluation
func evaluateBasicEndgames(board *Bitboard) int {
	endgameBonus := 0

	// Count material to identify endgame types
	whitePawns := PopCount(board.GetBitboardOf(WhitePawn))
	blackPawns := PopCount(board.GetBitboardOf(BlackPawn))
	whiteKnights := PopCount(board.GetBitboardOf(WhiteKnight))
	blackKnights := PopCount(board.GetBitboardOf(BlackKnight))
	whiteBishops := PopCount(board.GetBitboardOf(WhiteBishop))
	blackBishops := PopCount(board.GetBitboardOf(BlackBishop))
	whiteRooks := PopCount(board.GetBitboardOf(WhiteRook))
	blackRooks := PopCount(board.GetBitboardOf(BlackRook))
	whiteQueens := PopCount(board.GetBitboardOf(WhiteQueen))
	blackQueens := PopCount(board.GetBitboardOf(BlackQueen))

	// Total pieces (excluding kings)
	whitePieces := whitePawns + whiteKnights + whiteBishops + whiteRooks + whiteQueens
	blackPieces := blackPawns + blackKnights + blackBishops + blackRooks + blackQueens

	// 1. King and pawn endgames - evaluate opposition
	if whitePieces <= 1 && blackPieces <= 1 && (whitePawns > 0 || blackPawns > 0) {
		oppositionBonus := evaluateKingPawnOpposition(board)
		endgameBonus += oppositionBonus
	}

	// 2. Basic mate patterns
	if whitePieces <= 2 && blackPieces == 0 {
		// White has mating material vs lone king
		mateBonus := evaluateBasicMatePattern(board, White)
		endgameBonus += mateBonus
	}

	if blackPieces <= 2 && whitePieces == 0 {
		// Black has mating material vs lone king
		mateBonus := evaluateBasicMatePattern(board, Black)
		endgameBonus -= mateBonus
	}

	// 3. King activity in endgame
	if whitePieces+blackPieces <= 4 {
		kingActivityBonus := evaluateKingActivity(board)
		endgameBonus += kingActivityBonus
	}

	return endgameBonus
}

// Evaluate opposition in king and pawn endgames
func evaluateKingPawnOpposition(board *Bitboard) int {
	whiteKingBB := board.GetBitboardOf(WhiteKing)
	blackKingBB := board.GetBitboardOf(BlackKing)

	if whiteKingBB == 0 || blackKingBB == 0 {
		return 0
	}

	whiteKingSquare := trailingZeros(whiteKingBB)
	blackKingSquare := trailingZeros(blackKingBB)

	whiteKingFile := int(whiteKingSquare % 8)
	whiteKingRank := int(whiteKingSquare / 8)
	blackKingFile := int(blackKingSquare % 8)
	blackKingRank := int(blackKingSquare / 8)

	// Calculate distance between kings
	fileDistance := evalAbs(whiteKingFile - blackKingFile)
	rankDistance := evalAbs(whiteKingRank - blackKingRank)

	oppositionBonus := 0

	// Direct opposition - kings face each other with one square between
	if (fileDistance == 0 && rankDistance == 2) ||
		(fileDistance == 2 && rankDistance == 0) ||
		(fileDistance == 2 && rankDistance == 2) {

		// Determine who has the opposition (who moves next loses the opposition)
		// This is a simplified implementation - in real games, this depends on whose turn it is
		// For now, we'll give a small bonus to the side that can potentially gain opposition

		whitePawnsBB := board.GetBitboardOf(WhitePawn)
		blackPawnsBB := board.GetBitboardOf(BlackPawn)

		// If one side has pawns and the other doesn't, give opposition bonus to pawn side
		if PopCount(whitePawnsBB) > 0 && PopCount(blackPawnsBB) == 0 {
			oppositionBonus += 25 // White has pawns, wants opposition
		} else if PopCount(blackPawnsBB) > 0 && PopCount(whitePawnsBB) == 0 {
			oppositionBonus -= 25 // Black has pawns, wants opposition
		}
	}

	// Distant opposition - same principle but with more squares between
	if fileDistance%2 == 0 && rankDistance%2 == 0 &&
		(fileDistance+rankDistance) >= 4 && (fileDistance+rankDistance) <= 6 {

		whitePawnsBB := board.GetBitboardOf(WhitePawn)
		blackPawnsBB := board.GetBitboardOf(BlackPawn)

		if PopCount(whitePawnsBB) > 0 && PopCount(blackPawnsBB) == 0 {
			oppositionBonus += 15 // Smaller bonus for distant opposition
		} else if PopCount(blackPawnsBB) > 0 && PopCount(whitePawnsBB) == 0 {
			oppositionBonus -= 15
		}
	}

	return oppositionBonus
}

// Evaluate basic mate patterns (KQ vs K, KR vs K)
func evaluateBasicMatePattern(board *Bitboard, color Color) int {
	var kingBB, opponentKingBB uint64
	var queenBB, rookBB uint64

	if color == White {
		kingBB = board.GetBitboardOf(WhiteKing)
		opponentKingBB = board.GetBitboardOf(BlackKing)
		queenBB = board.GetBitboardOf(WhiteQueen)
		rookBB = board.GetBitboardOf(WhiteRook)
	} else {
		kingBB = board.GetBitboardOf(BlackKing)
		opponentKingBB = board.GetBitboardOf(WhiteKing)
		queenBB = board.GetBitboardOf(BlackQueen)
		rookBB = board.GetBitboardOf(BlackRook)
	}

	if kingBB == 0 || opponentKingBB == 0 {
		return 0
	}

	mateBonus := 0
	opponentKingSquare := trailingZeros(opponentKingBB)
	opponentKingFile := int(opponentKingSquare % 8)
	opponentKingRank := int(opponentKingSquare / 8)

	// Distance from center (enemy king should be driven to edge for mate)
	centerDistance := evalAbs(opponentKingFile-3) + evalAbs(opponentKingFile-4) +
		evalAbs(opponentKingRank-3) + evalAbs(opponentKingRank-4)
	centerDistance = centerDistance / 2 // Average distance from center files/ranks

	// Bonus for driving enemy king to edge
	edgeBonus := centerDistance * 10

	// KQ vs K mate
	if PopCount(queenBB) > 0 {
		mateBonus += 800 + edgeBonus // Massive bonus for having queen vs bare king

		// Additional bonus for restricting enemy king
		if opponentKingFile <= 1 || opponentKingFile >= 6 {
			mateBonus += 100 // King near edge
		}
		if opponentKingRank <= 1 || opponentKingRank >= 6 {
			mateBonus += 100 // King near edge
		}

		// Bonus for king proximity (supporting the mate)
		kingSquare := trailingZeros(kingBB)
		kingFile := int(kingSquare % 8)
		kingRank := int(kingSquare / 8)
		kingDistance := evalAbs(kingFile-opponentKingFile) + evalAbs(kingRank-opponentKingRank)
		if kingDistance <= 3 {
			mateBonus += (4 - kingDistance) * 50 // Closer king is better
		}
	}

	// KR vs K mate (harder than KQ vs K)
	if PopCount(rookBB) > 0 && PopCount(queenBB) == 0 {
		mateBonus += 600 + edgeBonus // Large bonus for rook vs bare king

		// Additional bonus for cutting off enemy king
		if opponentKingFile <= 1 || opponentKingFile >= 6 {
			mateBonus += 80
		}
		if opponentKingRank <= 1 || opponentKingRank >= 6 {
			mateBonus += 80
		}

		// Bonus for king proximity (crucial for rook mates)
		kingSquare := trailingZeros(kingBB)
		kingFile := int(kingSquare % 8)
		kingRank := int(kingSquare / 8)
		kingDistance := evalAbs(kingFile-opponentKingFile) + evalAbs(kingRank-opponentKingRank)
		if kingDistance <= 3 {
			mateBonus += (4 - kingDistance) * 60 // Closer king is even more important for rook mates
		}

		// Extra bonus for rook cutting off ranks/files
		rookSquare := trailingZeros(rookBB)
		rookFile := int(rookSquare % 8)
		rookRank := int(rookSquare / 8)
		if rookFile == opponentKingFile || rookRank == opponentKingRank {
			mateBonus += 50 // Rook on same rank/file as enemy king
		}
	}

	return mateBonus
}

// Evaluate king activity in endgames
func evaluateKingActivity(board *Bitboard) int {
	whiteKingBB := board.GetBitboardOf(WhiteKing)
	blackKingBB := board.GetBitboardOf(BlackKing)

	if whiteKingBB == 0 || blackKingBB == 0 {
		return 0
	}

	whiteKingSquare := trailingZeros(whiteKingBB)
	blackKingSquare := trailingZeros(blackKingBB)

	whiteKingFile := int(whiteKingSquare % 8)
	whiteKingRank := int(whiteKingSquare / 8)
	blackKingFile := int(blackKingSquare % 8)
	blackKingRank := int(blackKingSquare / 8)

	activityBonus := 0

	// Bonus for centralized kings in endgame
	whiteCentrality := 0
	blackCentrality := 0

	// Distance from center (closer to center is better)
	whiteCenterDistance := evalAbs(whiteKingFile-3) + evalAbs(whiteKingFile-4) +
		evalAbs(whiteKingRank-3) + evalAbs(whiteKingRank-4)
	whiteCenterDistance = whiteCenterDistance / 2
	whiteCentrality = 6 - whiteCenterDistance // Convert to bonus

	blackCenterDistance := evalAbs(blackKingFile-3) + evalAbs(blackKingFile-4) +
		evalAbs(blackKingRank-3) + evalAbs(blackKingRank-4)
	blackCenterDistance = blackCenterDistance / 2
	blackCentrality = 6 - blackCenterDistance

	activityBonus += (whiteCentrality - blackCentrality) * 3

	// Bonus for kings supporting their own pawns or attacking enemy pawns
	whitePawnsBB := board.GetBitboardOf(WhitePawn)
	blackPawnsBB := board.GetBitboardOf(BlackPawn)

	// Check if king is supporting own pawns
	for wpbb := whitePawnsBB; wpbb != 0; wpbb &= wpbb - 1 {
		pawnSquare := bits.TrailingZeros64(wpbb)
		pawnFile := int(pawnSquare % 8)
		pawnRank := int(pawnSquare / 8)
		distance := evalAbs(whiteKingFile-pawnFile) + evalAbs(whiteKingRank-pawnRank)
		if distance <= 2 {
			activityBonus += 8 // Bonus for king supporting pawn
		}
	}

	for bpbb := blackPawnsBB; bpbb != 0; bpbb &= bpbb - 1 {
		pawnSquare := bits.TrailingZeros64(bpbb)
		pawnFile := int(pawnSquare % 8)
		pawnRank := int(pawnSquare / 8)
		distance := evalAbs(blackKingFile-pawnFile) + evalAbs(blackKingRank-pawnRank)
		if distance <= 2 {
			activityBonus -= 8 // Bonus for black king supporting pawn
		}
	}

	return activityBonus
}

// Piece coordination and harmony evaluation
func evaluatePieceCoordination(board *Bitboard, color Color) int {
	coordinationBonus := 0

	// Get piece bitboards for the given color
	var kingBB, queenBB, rookBB, bishopBB, knightBB, pawnBB uint64
	var opponentPawnBB uint64

	if color == White {
		kingBB = board.GetBitboardOf(WhiteKing)
		queenBB = board.GetBitboardOf(WhiteQueen)
		rookBB = board.GetBitboardOf(WhiteRook)
		bishopBB = board.GetBitboardOf(WhiteBishop)
		knightBB = board.GetBitboardOf(WhiteKnight)
		pawnBB = board.GetBitboardOf(WhitePawn)
		opponentPawnBB = board.GetBitboardOf(BlackPawn)
	} else {
		kingBB = board.GetBitboardOf(BlackKing)
		queenBB = board.GetBitboardOf(BlackQueen)
		rookBB = board.GetBitboardOf(BlackRook)
		bishopBB = board.GetBitboardOf(BlackBishop)
		knightBB = board.GetBitboardOf(BlackKnight)
		pawnBB = board.GetBitboardOf(BlackPawn)
		opponentPawnBB = board.GetBitboardOf(WhitePawn)
	}

	// 1. Evaluate mutual piece protection
	coordinationBonus += evaluatePieceProtection(kingBB, queenBB, rookBB, bishopBB, knightBB, pawnBB, color)

	// 2. Evaluate outpost squares for knights and bishops
	coordinationBonus += evaluateOutpostSquares(knightBB, bishopBB, pawnBB, opponentPawnBB, color)

	// 3. Evaluate piece batteries (rook behind queen, doubled rooks)
	coordinationBonus += evaluatePieceBatteries(queenBB, rookBB, bishopBB)

	// 4. Evaluate piece cooperation in attacks
	coordinationBonus += evaluatePieceCooperation(queenBB, rookBB, bishopBB, knightBB, color)

	// 5. Evaluate centralization coordination
	coordinationBonus += evaluateCentralization(queenBB, rookBB, bishopBB, knightBB, pawnBB)

	return coordinationBonus
}

// Evaluate how well pieces protect each other
func evaluatePieceProtection(kingBB, queenBB, rookBB, bishopBB, knightBB, pawnBB uint64, color Color) int {
	protection := 0

	// Process queens
	for qbb := queenBB; qbb != 0; qbb &= qbb - 1 {
		queenSquare := bits.TrailingZeros64(qbb)
		protectors := countProtectorsBB(queenSquare, rookBB, bishopBB, knightBB, pawnBB, color)
		protection += protectors * 15 // High bonus for queen protection
	}

	// Process rooks
	for rbb := rookBB; rbb != 0; rbb &= rbb - 1 {
		rookSquare := bits.TrailingZeros64(rbb)
		protectors := countProtectorsBB(rookSquare, 0, bishopBB, knightBB, pawnBB, color)
		// Add queen protection
		for qbb := queenBB; qbb != 0; qbb &= qbb - 1 {
			queenSquare := bits.TrailingZeros64(qbb)
			if canQueenProtect(queenSquare, rookSquare) {
				protectors++
			}
		}
		protection += protectors * 8 // Good bonus for rook protection
	}

	// Process bishops
	for bbb := bishopBB; bbb != 0; bbb &= bbb - 1 {
		bishopSquare := bits.TrailingZeros64(bbb)
		protectors := countProtectorsBB(bishopSquare, rookBB, 0, knightBB, pawnBB, color)
		for qbb := queenBB; qbb != 0; qbb &= qbb - 1 {
			queenSquare := bits.TrailingZeros64(qbb)
			if canQueenProtect(queenSquare, bishopSquare) {
				protectors++
			}
		}
		protection += protectors * 5 // Moderate bonus for bishop protection
	}

	// Process knights
	for nbb := knightBB; nbb != 0; nbb &= nbb - 1 {
		knightSquare := bits.TrailingZeros64(nbb)
		protectors := countProtectorsBB(knightSquare, rookBB, bishopBB, 0, pawnBB, color)
		for qbb := queenBB; qbb != 0; qbb &= qbb - 1 {
			queenSquare := bits.TrailingZeros64(qbb)
			if canQueenProtect(queenSquare, knightSquare) {
				protectors++
			}
		}
		protection += protectors * 5 // Moderate bonus for knight protection
	}

	return protection
}

// Count how many pieces can protect a given square (using bitboards)
func countProtectorsBB(targetSquare int, rookBB, bishopBB, knightBB, pawnBB uint64, color Color) int {
	protectors := 0
	targetFile := targetSquare % 8
	targetRank := targetSquare / 8

	// Check rook protection (same rank or file)
	for rbb := rookBB; rbb != 0; rbb &= rbb - 1 {
		rookSquare := bits.TrailingZeros64(rbb)
		rookFile := rookSquare % 8
		rookRank := rookSquare / 8
		if rookFile == targetFile || rookRank == targetRank {
			protectors++
		}
	}

	// Check bishop protection (same diagonal)
	for bbb := bishopBB; bbb != 0; bbb &= bbb - 1 {
		bishopSquare := bits.TrailingZeros64(bbb)
		if onSameDiagonal(bishopSquare, targetSquare) {
			protectors++
		}
	}

	// Check knight protection
	for nbb := knightBB; nbb != 0; nbb &= nbb - 1 {
		knightSquare := bits.TrailingZeros64(nbb)
		if canKnightReach(knightSquare, targetSquare) {
			protectors++
		}
	}

	// Check pawn protection
	for pbb := pawnBB; pbb != 0; pbb &= pbb - 1 {
		pawnSquare := bits.TrailingZeros64(pbb)
		if canPawnProtect(pawnSquare, targetSquare, color) {
			protectors++
		}
	}

	return protectors
}

// Keep old version for compatibility if needed
func countProtectors(targetSquare int, rooks, bishops, knights, pawns []int, color Color) int {
	protectors := 0
	targetFile := targetSquare % 8
	targetRank := targetSquare / 8

	// Check rook protection (same rank or file)
	for _, rookSquare := range rooks {
		rookFile := rookSquare % 8
		rookRank := rookSquare / 8
		if rookFile == targetFile || rookRank == targetRank {
			protectors++
		}
	}

	// Check bishop protection (same diagonal)
	for _, bishopSquare := range bishops {
		if onSameDiagonal(bishopSquare, targetSquare) {
			protectors++
		}
	}

	// Check knight protection
	for _, knightSquare := range knights {
		if canKnightReach(knightSquare, targetSquare) {
			protectors++
		}
	}

	// Check pawn protection
	for _, pawnSquare := range pawns {
		if canPawnProtect(pawnSquare, targetSquare, color) {
			protectors++
		}
	}

	return protectors
}

// Check if pieces are on the same diagonal
func onSameDiagonal(square1, square2 int) bool {
	file1, rank1 := square1%8, square1/8
	file2, rank2 := square2%8, square2/8
	return evalAbs(file1-file2) == evalAbs(rank1-rank2)
}

// Check if knight can reach target square
func canKnightReach(knightSquare, targetSquare int) bool {
	knightFile, knightRank := knightSquare%8, knightSquare/8
	targetFile, targetRank := targetSquare%8, targetSquare/8

	fileDiff := evalAbs(knightFile - targetFile)
	rankDiff := evalAbs(knightRank - targetRank)

	return (fileDiff == 2 && rankDiff == 1) || (fileDiff == 1 && rankDiff == 2)
}

// Check if pawn can protect target square
//
//go:inline
func canPawnProtect(pawnSquare, targetSquare int, color Color) bool {
	pawnFile, pawnRank := pawnSquare%8, pawnSquare/8
	targetFile, targetRank := targetSquare%8, targetSquare/8

	// Pawns protect diagonally forward
	if evalAbs(pawnFile-targetFile) != 1 {
		return false
	}

	if color == White {
		return targetRank == pawnRank+1
	} else {
		return targetRank == pawnRank-1
	}
}

// Check if queen can protect target square
func canQueenProtect(queenSquare, targetSquare int) bool {
	// Queen combines rook and bishop movement
	queenFile, queenRank := queenSquare%8, queenSquare/8
	targetFile, targetRank := targetSquare%8, targetSquare/8

	// Same rank, file, or diagonal
	return queenFile == targetFile || queenRank == targetRank || onSameDiagonal(queenSquare, targetSquare)
}

// Evaluate outpost squares for knights and bishops
func evaluateOutpostSquares(knightBB, bishopBB, pawnBB, opponentPawnBB uint64, color Color) int {
	outpostBonus := 0

	// Check knights on outpost squares
	for nbb := knightBB; nbb != 0; nbb &= nbb - 1 {
		knightSquare := bits.TrailingZeros64(nbb)
		if isOutpostSquareBB(knightSquare, pawnBB, opponentPawnBB, color) {
			// Knights on outposts are very strong
			bonus := 20

			// Extra bonus for advanced outposts
			rank := knightSquare / 8
			if color == White && rank >= 4 {
				bonus += (rank - 3) * 5 // Bonus increases with advancement
			} else if color == Black && rank <= 3 {
				bonus += (4 - rank) * 5
			}

			outpostBonus += bonus
		}
	}

	// Check bishops on outpost squares (less common but still valuable)
	for bbb := bishopBB; bbb != 0; bbb &= bbb - 1 {
		bishopSquare := bits.TrailingZeros64(bbb)
		if isOutpostSquareBB(bishopSquare, pawnBB, opponentPawnBB, color) {
			bonus := 12

			// Bishops on outposts in enemy territory
			rank := bishopSquare / 8
			if color == White && rank >= 4 {
				bonus += (rank - 3) * 3
			} else if color == Black && rank <= 3 {
				bonus += (4 - rank) * 3
			}

			outpostBonus += bonus
		}
	}

	return outpostBonus
}

// Check if a square is an outpost (supported by pawn, can't be attacked by enemy pawns)
// Check if a square is an outpost (using bitboards)
func isOutpostSquareBB(square int, friendlyPawnBB, enemyPawnBB uint64, color Color) bool {
	file := square % 8
	rank := square / 8

	// 1. Must be supported by our own pawns
	supported := false
	for pbb := friendlyPawnBB; pbb != 0; pbb &= pbb - 1 {
		pawnSquare := bits.TrailingZeros64(pbb)
		if canPawnProtect(pawnSquare, square, color) {
			supported = true
			break
		}
	}

	if !supported {
		return false
	}

	// 2. Must not be attackable by enemy pawns (now or in future)
	enemyColor := White
	if color == White {
		enemyColor = Black
	}

	for ebb := enemyPawnBB; ebb != 0; ebb &= ebb - 1 {
		enemyPawnSquare := bits.TrailingZeros64(ebb)
		enemyFile := enemyPawnSquare % 8
		enemyRank := enemyPawnSquare / 8

		// Check if enemy pawn can attack this square now
		if canPawnProtect(enemyPawnSquare, square, enemyColor) {
			return false
		}

		// Check if enemy pawn can attack this square in the future
		// (i.e., can advance and then attack)
		if evalAbs(enemyFile-file) == 1 {
			if color == White && enemyRank < rank {
				// Enemy pawn can potentially advance to attack
				return false
			} else if color == Black && enemyRank > rank {
				// Enemy pawn can potentially advance to attack
				return false
			}
		}
	}

	// 3. Must be in enemy territory (ranks 4-6 for white, ranks 3-5 for black)
	if color == White && (rank < 3 || rank > 5) {
		return false
	}
	if color == Black && (rank < 2 || rank > 4) {
		return false
	}

	return true
}

// Keep old version for compatibility
func isOutpostSquare(square int, friendlyPawns, enemyPawns []int, color Color) bool {
	file := square % 8
	rank := square / 8

	// 1. Must be supported by our own pawns
	supported := false
	for _, pawnSquare := range friendlyPawns {
		if canPawnProtect(pawnSquare, square, color) {
			supported = true
			break
		}
	}

	if !supported {
		return false
	}

	// 2. Must not be attackable by enemy pawns (now or in future)
	for _, enemyPawnSquare := range enemyPawns {
		enemyFile := enemyPawnSquare % 8
		enemyRank := enemyPawnSquare / 8

		// Check if enemy pawn can attack this square now
		enemyColor := White
		if color == White {
			enemyColor = Black
		}

		if canPawnProtect(enemyPawnSquare, square, enemyColor) {
			return false
		}

		// Check if enemy pawn could advance to attack this square
		if evalAbs(enemyFile-file) == 1 {
			if color == White {
				// Enemy pawns advance toward us (down ranks)
				if enemyRank > rank && enemyRank-1 == rank {
					return false // Pawn could advance to attack
				}
			} else {
				// Enemy pawns advance toward us (up ranks)
				if enemyRank < rank && enemyRank+1 == rank {
					return false
				}
			}
		}
	}

	return true
}

// Evaluate piece batteries (rook behind queen, doubled rooks, bishop pairs)
func evaluatePieceBatteries(queenBB, rookBB, bishopBB uint64) int {
	batteryBonus := 0

	// 1. Rook behind queen on same file or rank
	for queenBits := queenBB; queenBits != 0; queenBits &= queenBits - 1 {
		queenSquare := bits.TrailingZeros64(queenBits)
		queenFile, queenRank := queenSquare%8, queenSquare/8

		for rookBits := rookBB; rookBits != 0; rookBits &= rookBits - 1 {
			rookSquare := bits.TrailingZeros64(rookBits)
			rookFile, rookRank := rookSquare%8, rookSquare/8

			// Same file
			if queenFile == rookFile && queenRank != rookRank {
				batteryBonus += 15 // Vertical battery
			}

			// Same rank
			if queenRank == rookRank && queenFile != rookFile {
				batteryBonus += 15 // Horizontal battery
			}
		}
	}

	// 2. Doubled/tripled rooks - use arrays instead of maps for performance
	var fileRooks [8]int // file counts (files 0-7)
	var rankRooks [8]int // rank counts (ranks 0-7)

	for rookBits := rookBB; rookBits != 0; rookBits &= rookBits - 1 {
		rookSquare := bits.TrailingZeros64(rookBits)
		file, rank := rookSquare%8, rookSquare/8
		fileRooks[file]++
		rankRooks[rank]++
	}

	for _, count := range fileRooks {
		if count >= 2 {
			batteryBonus += count * 10 // Bonus for doubled rooks on file
		}
	}

	for _, count := range rankRooks {
		if count >= 2 {
			batteryBonus += count * 8 // Bonus for doubled rooks on rank
		}
	}

	// 3. Bishop pair on same diagonal
	bishopCount := bits.OnesCount64(bishopBB)
	if bishopCount >= 2 {
		// Check all pairs of bishops
		bishops := [8]int{} // Maximum 8 bishops theoretically
		idx := 0
		for bishopBits := bishopBB; bishopBits != 0; bishopBits &= bishopBits - 1 {
			bishops[idx] = bits.TrailingZeros64(bishopBits)
			idx++
		}

		for i := 0; i < bishopCount; i++ {
			for j := i + 1; j < bishopCount; j++ {
				if onSameDiagonal(bishops[i], bishops[j]) {
					batteryBonus += 12 // Bonus for bishops on same diagonal
				}
			}
		}
	}

	return batteryBonus
}

// Evaluate piece cooperation in attacks (SIMPLIFIED for performance)
func evaluatePieceCooperation(queenBB, rookBB, bishopBB, knightBB uint64, color Color) int {
	// Simplified piece cooperation: just count pieces and give a small bonus
	// The original complex version was taking 52% CPU time for ~5 Elo gain
	cooperationBonus := 0

	// Simple approach: give small bonus for having multiple piece types developed
	pieceTypes := 0
	if queenBB != 0 {
		pieceTypes++
	}
	if rookBB != 0 {
		pieceTypes++
	}
	if bishopBB != 0 {
		pieceTypes++
	}
	if knightBB != 0 {
		pieceTypes++
	}

	if pieceTypes >= 3 {
		cooperationBonus += 5 // Small bonus for piece development
	}
	if pieceTypes >= 4 {
		cooperationBonus += 5 // Additional bonus for full development
	}

	// 2. Piece coordination around enemy king (if we can find it)
	// This is a simplified version - in practice you'd get enemy king position

	return cooperationBonus
}

// Simplified attack pattern generators (these would normally be more complex)
func getQueenAttacks(square int) []int {
	// Simplified: return squares on same rank, file, and diagonals
	attacks := make([]int, 0, 27) // Max 8+8+8+3 squares
	file, rank := square%8, square/8

	// Add rank attacks
	for f := 0; f < 8; f++ {
		if f != file {
			attacks = append(attacks, rank*8+f)
		}
	}

	// Add file attacks
	for r := 0; r < 8; r++ {
		if r != rank {
			attacks = append(attacks, r*8+file)
		}
	}

	// Add diagonal attacks (simplified)
	for i := 1; i < 8; i++ {
		// Up-right diagonal
		if file+i < 8 && rank+i < 8 {
			attacks = append(attacks, (rank+i)*8+(file+i))
		}
		// Up-left diagonal
		if file-i >= 0 && rank+i < 8 {
			attacks = append(attacks, (rank+i)*8+(file-i))
		}
		// Down-right diagonal
		if file+i < 8 && rank-i >= 0 {
			attacks = append(attacks, (rank-i)*8+(file+i))
		}
		// Down-left diagonal
		if file-i >= 0 && rank-i >= 0 {
			attacks = append(attacks, (rank-i)*8+(file-i))
		}
	}

	return attacks
}

func getRookAttacks(square int) []int {
	attacks := make([]int, 0, 14)
	file, rank := square%8, square/8

	// Add rank attacks
	for f := 0; f < 8; f++ {
		if f != file {
			attacks = append(attacks, rank*8+f)
		}
	}

	// Add file attacks
	for r := 0; r < 8; r++ {
		if r != rank {
			attacks = append(attacks, r*8+file)
		}
	}

	return attacks
}

func getBishopAttacks(square int) []int {
	attacks := make([]int, 0, 13)
	file, rank := square%8, square/8

	// Add diagonal attacks (simplified)
	for i := 1; i < 8; i++ {
		// Up-right diagonal
		if file+i < 8 && rank+i < 8 {
			attacks = append(attacks, (rank+i)*8+(file+i))
		}
		// Up-left diagonal
		if file-i >= 0 && rank+i < 8 {
			attacks = append(attacks, (rank+i)*8+(file-i))
		}
		// Down-right diagonal
		if file+i < 8 && rank-i >= 0 {
			attacks = append(attacks, (rank-i)*8+(file+i))
		}
		// Down-left diagonal
		if file-i >= 0 && rank-i >= 0 {
			attacks = append(attacks, (rank-i)*8+(file-i))
		}
	}

	return attacks
}

func getKnightAttacks(square int) []int {
	attacks := make([]int, 0, 8)
	file, rank := square%8, square/8

	// All possible knight moves
	knightMoves := [][]int{
		{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
		{1, -2}, {1, 2}, {2, -1}, {2, 1},
	}

	for _, move := range knightMoves {
		newFile := file + move[0]
		newRank := rank + move[1]

		if newFile >= 0 && newFile < 8 && newRank >= 0 && newRank < 8 {
			attacks = append(attacks, newRank*8+newFile)
		}
	}

	return attacks
}

// Evaluate centralization coordination
func evaluateCentralization(queenBB, rookBB, bishopBB, knightBB, pawnBB uint64) int {
	centralizationBonus := 0

	// Central squares (d4, d5, e4, e5)
	centralSquares := []int{27, 28, 35, 36}                                         // d4, e4, d5, e5
	extendedCentralSquares := []int{18, 19, 20, 21, 26, 29, 34, 37, 42, 43, 44, 45} // c3-f6 area

	// Check pieces on central squares
	for _, square := range centralSquares {
		squareMask := uint64(1 << square)

		if (knightBB & squareMask) != 0 {
			centralizationBonus += 15 // Knights are excellent in center
		}
		if (bishopBB & squareMask) != 0 {
			centralizationBonus += 10 // Bishops good in center
		}
		if (pawnBB & squareMask) != 0 {
			centralizationBonus += 12 // Central pawns are strong
		}
	}

	// Check pieces on extended central squares
	for _, square := range extendedCentralSquares {
		squareMask := uint64(1 << square)

		if (knightBB & squareMask) != 0 {
			centralizationBonus += 8 // Good knight squares
		}
		if (bishopBB & squareMask) != 0 {
			centralizationBonus += 5 // Decent bishop squares
		}
		if (queenBB & squareMask) != 0 {
			centralizationBonus += 6 // Queen can be good in center but risky
		}
	}

	// Bonus for multiple pieces controlling center
	centralControl := 0
	for _, square := range centralSquares {
		squareMask := uint64(1 << square)

		// Count pieces that could move to this square (simplified)
		if (knightBB|bishopBB|rookBB|queenBB)&squareMask != 0 {
			centralControl++
		}
	}

	// Bonus for controlling multiple central squares
	if centralControl >= 2 {
		centralizationBonus += centralControl * 3
	}

	return centralizationBonus
}

// Evaluate bishop pair bonus
func evaluateBishopPair(board *Bitboard, color Color) int {
	var bishopBB uint64

	if color == White {
		bishopBB = board.GetBitboardOf(WhiteBishop)
	} else {
		bishopBB = board.GetBitboardOf(BlackBishop)
	}

	bishopCount := PopCount(bishopBB)

	// Must have exactly 2 bishops for the pair bonus
	if bishopCount != 2 {
		return 0
	}

	// Extract the two bishop positions using bitboard iteration
	bishops := [2]int{}
	idx := 0
	for bishopBits := bishopBB; bishopBits != 0 && idx < 2; bishopBits &= bishopBits - 1 {
		bishops[idx] = bits.TrailingZeros64(bishopBits)
		idx++
	}

	if idx != 2 {
		return 0 // Safety check
	}

	// Check if bishops are on opposite colored squares
	bishop1Square := bishops[0]
	bishop2Square := bishops[1]

	// Calculate square colors (light or dark)
	// Light squares: (file + rank) % 2 == 0
	// Dark squares: (file + rank) % 2 == 1
	bishop1Color := (bishop1Square%8 + bishop1Square/8) % 2
	bishop2Color := (bishop2Square%8 + bishop2Square/8) % 2

	// Bishop pair bonus only applies if bishops are on opposite colored squares
	if bishop1Color != bishop2Color {
		// Base bishop pair bonus
		bonus := 30

		// Additional bonus in open positions (fewer pawns = more open)
		totalPawns := PopCount(board.GetBitboardOf(WhitePawn)) + PopCount(board.GetBitboardOf(BlackPawn))

		// The fewer pawns, the stronger the bishop pair
		if totalPawns <= 12 {
			bonus += 10 // Extra bonus in open positions
		}
		if totalPawns <= 8 {
			bonus += 10 // Even more bonus in very open positions
		}

		// Endgame bonus - bishop pair becomes stronger in endgame
		if isEndgame(board) {
			bonus += 15 // Additional endgame bonus
		}

		return bonus
	}

	return 0
}

// Enhanced evaluation function - always returns from white's perspective
func Evaluate(board *Bitboard) (eval int) {
	return WrapEvaluation("Evaluate", func() int {
		return evaluateUnsafe(board)
	}, board)
}

// EvaluateForPlayer returns evaluation from the given player's perspective
func EvaluateForPlayer(board *Bitboard, player Color) int {
	whiteEval := Evaluate(board)
	if player == White {
		return whiteEval
	} else {
		return -whiteEval // Flip for black's perspective
	}
}

func evaluateUnsafe(board *Bitboard) (eval int) {
	endgame := isEndgame(board)

	// Cache all piece bitboards for performance (avoid repeated GetBitboardOf calls)
	whitePawns := board.GetBitboardOf(WhitePawn)
	blackPawns := board.GetBitboardOf(BlackPawn)
	whiteKnights := board.GetBitboardOf(WhiteKnight)
	blackKnights := board.GetBitboardOf(BlackKnight)
	whiteBishops := board.GetBitboardOf(WhiteBishop)
	blackBishops := board.GetBitboardOf(BlackBishop)
	whiteRooks := board.GetBitboardOf(WhiteRook)
	blackRooks := board.GetBitboardOf(BlackRook)
	whiteQueens := board.GetBitboardOf(WhiteQueen)
	blackQueens := board.GetBitboardOf(BlackQueen)
	whiteKings := board.GetBitboardOf(WhiteKing)
	blackKings := board.GetBitboardOf(BlackKing)

	// 1. Material counting and piece-square table values in one pass
	// White pieces - material + PST
	eval += int(WhitePawn.Weight()) * PopCount(whitePawns)
	for bb := whitePawns; bb != 0; bb &= bb - 1 {
		eval += pawnPST[bits.TrailingZeros64(bb)]
	}

	eval += int(WhiteKnight.Weight()) * PopCount(whiteKnights)
	for bb := whiteKnights; bb != 0; bb &= bb - 1 {
		eval += knightPST[bits.TrailingZeros64(bb)]
	}

	eval += int(WhiteBishop.Weight()) * PopCount(whiteBishops)
	for bb := whiteBishops; bb != 0; bb &= bb - 1 {
		eval += bishopPST[bits.TrailingZeros64(bb)]
	}

	eval += int(WhiteRook.Weight()) * PopCount(whiteRooks)
	for bb := whiteRooks; bb != 0; bb &= bb - 1 {
		eval += rookPST[bits.TrailingZeros64(bb)]
	}

	eval += int(WhiteQueen.Weight()) * PopCount(whiteQueens)
	for bb := whiteQueens; bb != 0; bb &= bb - 1 {
		eval += queenPST[bits.TrailingZeros64(bb)]
	}

	// White king PST
	for bb := whiteKings; bb != 0; bb &= bb - 1 {
		sq := bits.TrailingZeros64(bb)
		if endgame {
			eval += kingEndgamePST[sq]
		} else {
			eval += kingMiddlegamePST[sq]
		}
	}

	// Black pieces - material + PST (using pre-computed negative tables)
	eval -= int(BlackPawn.Weight()) * PopCount(blackPawns)
	for bb := blackPawns; bb != 0; bb &= bb - 1 {
		eval += blackPawnPST[bits.TrailingZeros64(bb)]
	}

	eval -= int(BlackKnight.Weight()) * PopCount(blackKnights)
	for bb := blackKnights; bb != 0; bb &= bb - 1 {
		eval += blackKnightPST[bits.TrailingZeros64(bb)]
	}

	eval -= int(BlackBishop.Weight()) * PopCount(blackBishops)
	for bb := blackBishops; bb != 0; bb &= bb - 1 {
		eval += blackBishopPST[bits.TrailingZeros64(bb)]
	}

	eval -= int(BlackRook.Weight()) * PopCount(blackRooks)
	for bb := blackRooks; bb != 0; bb &= bb - 1 {
		eval += blackRookPST[bits.TrailingZeros64(bb)]
	}

	eval -= int(BlackQueen.Weight()) * PopCount(blackQueens)
	for bb := blackQueens; bb != 0; bb &= bb - 1 {
		eval += blackQueenPST[bits.TrailingZeros64(bb)]
	}

	// Black king PST
	for bb := blackKings; bb != 0; bb &= bb - 1 {
		sq := bits.TrailingZeros64(bb)
		if endgame {
			eval += blackKingEndgamePST[sq]
		} else {
			eval += blackKingMiddlegamePST[sq]
		}
	}

	// 3. Pawn structure evaluation - DISABLED FOR PERFORMANCE
	// Profiling showed these were major CPU hotspots:
	// - countDoubledPawns: 2.04% CPU
	// - evaluatePawnChains: 2.04% CPU
	// Keeping only essential pawn evaluation

	// Optimized passed pawn counting using cached bitboards
	whitePassedPawns := countPassedPawnsFast(whitePawns, blackPawns, White)
	blackPassedPawns := countPassedPawnsFast(blackPawns, whitePawns, Black)
	eval += whitePassedPawns * 25 // Bonus for passed pawns
	eval -= blackPassedPawns * 25

	// DISABLED FOR PERFORMANCE:
	// whiteDoubledPawns := 0
	// blackDoubledPawns := 0
	// for file := 0; file < 8; file++ {
	//	whiteDoubledPawns += countDoubledPawns(board, White, file)
	//	blackDoubledPawns += countDoubledPawns(board, Black, file)
	// }
	// eval += -(whiteDoubledPawns * 50)
	// eval += (blackDoubledPawns * 50)

	// whiteIsolatedPawns := countIsolatedPawns(board, White)
	// blackIsolatedPawns := countIsolatedPawns(board, Black)
	// eval += -(whiteIsolatedPawns * 15)
	// eval += (blackIsolatedPawns * 15)

	// whitePawnChains := evaluatePawnChains(board, White)
	// blackPawnChains := evaluatePawnChains(board, Black)
	// eval += whitePawnChains
	// eval += -blackPawnChains

	// whiteWeakSquares := evaluateWeakSquares(board, White)
	// blackWeakSquares := evaluateWeakSquares(board, Black)
	// eval += whiteWeakSquares
	// eval += -blackWeakSquares

	// whitePawnStorms := evaluatePawnStorms(board, White)
	// blackPawnStorms := evaluatePawnStorms(board, Black)
	// eval += whitePawnStorms
	// eval += -blackPawnStorms

	// 4. King safety evaluation - DISABLED FOR PERFORMANCE
	// evaluateKingSafety was a major CPU hotspot
	// if !endgame {
	//	whiteKingSafety := evaluateKingSafety(board, White)
	//	blackKingSafety := evaluateKingSafety(board, Black)
	//	eval += whiteKingSafety
	//	eval += -blackKingSafety
	// }

	// 5. Castling evaluation (king safety bonus) - OPTIMIZED VERSION
	whiteCastlingBonus := evaluateCastlingFast(board, White)
	blackCastlingBonus := evaluateCastlingFast(board, Black)
	eval += whiteCastlingBonus
	eval += -blackCastlingBonus

	// 6. Center control evaluation - OPTIMIZED VERSION
	whiteCenterControl := evaluateCenterControlFast(board, White)
	blackCenterControl := evaluateCenterControlFast(board, Black)
	eval += whiteCenterControl
	eval += -blackCenterControl

	// 7. Piece development evaluation (opening principles) - DISABLED FOR PERFORMANCE
	// if !endgame { // Development most important in opening/middlegame
	//	whiteDevelopment := evaluatePieceDevelopment(board, White)
	//	blackDevelopment := evaluatePieceDevelopment(board, Black)
	//	eval += whiteDevelopment
	//	eval += -blackDevelopment
	// }

	// 8. Piece coordination and harmony - DISABLED FOR PERFORMANCE
	// evaluatePieceProtection was 8.50% CPU, countProtectorsBB was 6.12% CPU
	// whiteCoordination := evaluatePieceCoordination(board, White)
	// blackCoordination := evaluatePieceCoordination(board, Black)
	// eval += whiteCoordination
	// eval += -blackCoordination

	// 9. Bishop pair bonus
	whiteBishopPairBonus := evaluateBishopPair(board, White)
	blackBishopPairBonus := evaluateBishopPair(board, Black)
	eval += whiteBishopPairBonus
	eval += -blackBishopPairBonus

	// 7. Basic endgame knowledge
	if endgame {
		endgameBonus := evaluateBasicEndgames(board)
		eval += endgameBonus
	}

	// 8. CRITICAL: Opening principles - castling safety bonus
	if !endgame {
		// Check if kings have castled (not on starting squares)
		whiteKingSquare := bits.TrailingZeros64(whiteKings)
		blackKingSquare := bits.TrailingZeros64(blackKings)

		// Castling bonus: king on g1/c1 for white, g8/c8 for black
		if whiteKingSquare == 6 || whiteKingSquare == 2 { // g1 or c1
			eval += 50 // Castling safety bonus
		}
		if blackKingSquare == 62 || blackKingSquare == 58 { // g8 or c8
			eval -= 50 // Castling safety bonus for black
		}

		// Penalty for king still on starting square in middlegame
		if whiteKingSquare == 4 { // e1
			eval -= 30 // Penalty for uncastled king
		}
		if blackKingSquare == 60 { // e8
			eval += 30 // Penalty for uncastled king (black)
		}
	}
	
	// 9. CRITICAL: Center control bonus - reward pawns controlling e4/d4/e5/d5
	centerBonusWhite := 0
	centerBonusBlack := 0
	
	// Count pawns controlling central squares
	for bb := whitePawns; bb != 0; bb &= bb - 1 {
		square := bits.TrailingZeros64(bb)
		
		// Bonus for pawns on d4/e4 (central occupation)
		if square == 28 || square == 36 { // d4, e4
			centerBonusWhite += 25
		}
		// Bonus for pawns on d3/e3/c3/f3 (supporting center)
		if square == 20 || square == 28 || square == 19 || square == 21 { // d3, e3, c3, f3
			centerBonusWhite += 10  
		}
	}
	
	for bb := blackPawns; bb != 0; bb &= bb - 1 {
		square := bits.TrailingZeros64(bb)
		
		// Bonus for pawns on d5/e5 (central occupation)
		if square == 35 || square == 27 { // d5, e5  
			centerBonusBlack += 25
		}
		// Bonus for pawns on d6/e6/c6/f6 (supporting center)
		if square == 43 || square == 35 || square == 42 || square == 44 { // d6, e6, c6, f6
			centerBonusBlack += 10
		}
	}
	
	eval += centerBonusWhite - centerBonusBlack

	return eval
}
