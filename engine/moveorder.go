package engine

import (
	"sort"
)

type moveScore struct {
	move  Move
	score int
}

func orderMoves(moves []Move) []Move {
	if len(moves) == 0 {
		return moves
	}
	
	moveScores := make([]moveScore, len(moves))
	
	for i, move := range moves {
		moveScores[i] = moveScore{
			move:  move,
			score: scoreMoveForOrdering(move),
		}
	}
	
	// Sort by score descending (higher scores first)
	sort.Slice(moveScores, func(i, j int) bool {
		return moveScores[i].score > moveScores[j].score
	})
	
	orderedMoves := make([]Move, len(moves))
	for i, ms := range moveScores {
		orderedMoves[i] = ms.move
	}
	
	return orderedMoves
}

func scoreMoveForOrdering(move Move) int {
	score := 0
	
	// Prioritize captures using MVV-LVA (Most Valuable Victim - Least Valuable Attacker)
	if move.IsCapture() {
		victim := move.CapturedPiece()
		attacker := move.MovingPiece()
		
		// Higher value victims are better
		score += int(victim.Weight()) * 10
		
		// Lower value attackers are better (subtract attacker value)
		score -= int(attacker.Weight())
		
		// Base score for captures
		score += 10000
	}
	
	// Prioritize promotions
	if move.PromoType() != NoType {
		score += 9000
		score += int(GetPiece(move.PromoType(), White).Weight()) * 10
	}
	
	// Castling gets a modest bonus
	if move.IsCastle() {
		score += 5000
	}
	
	// En passant captures
	if move.IsEnPassant() {
		score += 10000 + 100 // Equivalent to capturing a pawn
	}
	
	return score
}