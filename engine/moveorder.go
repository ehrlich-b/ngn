package engine

import (
	"sort"
)

// History table for move ordering heuristic
// historyTable[from][to] stores the history score for moves from 'from' to 'to'
var historyTable [64][64]int

// Killer moves table for move ordering heuristic
// killerMoves[depth][slot] stores killer moves at each depth (2 killer moves per depth)
var killerMoves [MAX_DEPTH][2]Move

// Counter moves table for move ordering heuristic
// counterMoves[from][to] stores the best counter move to moves from 'from' to 'to'
var counterMoves [64][64]Move

// Track the last move played for counter move heuristic
var lastMovePlayed Move

// UpdateHistoryTable increases the history score for a move that caused a cutoff
func UpdateHistoryTable(move Move, depth int) {
	if !move.IsCapture() && move.PromoType() == NoType {
		from := move.Source()
		to := move.Destination()
		// Depth-based bonus: deeper searches are more important
		bonus := depth * depth
		historyTable[from][to] += bonus
		
		// Prevent overflow by capping the history score
		if historyTable[from][to] > 8192 {
			// Age the history table by dividing all values by 2
			ageHistoryTable()
		}
	}
}

// GetHistoryScore returns the history score for a move
func GetHistoryScore(move Move) int {
	if !move.IsCapture() && move.PromoType() == NoType {
		from := move.Source()
		to := move.Destination()
		return historyTable[from][to]
	}
	return 0
}

// ClearHistoryTable resets the history table (used between games)
func ClearHistoryTable() {
	for i := range historyTable {
		for j := range historyTable[i] {
			historyTable[i][j] = 0
		}
	}
}

// ageHistoryTable divides all history values by 2 to prevent overflow
func ageHistoryTable() {
	for i := range historyTable {
		for j := range historyTable[i] {
			historyTable[i][j] /= 2
		}
	}
}

// UpdateKillerMoves stores a move that caused a beta cutoff as a killer move
func UpdateKillerMoves(move Move, depth int) {
	// Only store quiet moves as killer moves
	if !move.IsCapture() && move.PromoType() == NoType && depth >= 0 && depth < MAX_DEPTH {
		// Check if this move is already stored as first killer
		if killerMoves[depth][0] == move {
			return
		}
		
		// Shift moves down: second killer becomes first, new move becomes second
		killerMoves[depth][1] = killerMoves[depth][0]
		killerMoves[depth][0] = move
	}
}

// IsKillerMove checks if a move is a killer move at the given depth
func IsKillerMove(move Move, depth int) bool {
	if depth < 0 || depth >= MAX_DEPTH {
		return false
	}
	return killerMoves[depth][0] == move || killerMoves[depth][1] == move
}

// ClearKillerMoves resets the killer moves table
func ClearKillerMoves() {
	for i := range killerMoves {
		killerMoves[i][0] = EmptyMove
		killerMoves[i][1] = EmptyMove
	}
}

// UpdateCounterMove stores a move as the best counter to the given previous move
func UpdateCounterMove(counterMove Move, previousMove Move) {
	// Only store quiet moves as counter moves
	if !counterMove.IsCapture() && counterMove.PromoType() == NoType && previousMove != EmptyMove {
		from := previousMove.Source()
		to := previousMove.Destination()
		counterMoves[from][to] = counterMove
	}
}

// GetCounterMove returns the counter move for the given previous move
func GetCounterMove(previousMove Move) Move {
	if previousMove == EmptyMove {
		return EmptyMove
	}
	from := previousMove.Source()
	to := previousMove.Destination()
	return counterMoves[from][to]
}

// IsCounterMove checks if a move is the counter move to the previous move
func IsCounterMove(move Move, previousMove Move) bool {
	return GetCounterMove(previousMove) == move && move != EmptyMove
}

// SetLastMovePlayed sets the last move played for counter move heuristic
func SetLastMovePlayed(move Move) {
	lastMovePlayed = move
}

// GetLastMovePlayed returns the last move played
func GetLastMovePlayed() Move {
	return lastMovePlayed
}

// ClearCounterMoves resets the counter moves table
func ClearCounterMoves() {
	for i := range counterMoves {
		for j := range counterMoves[i] {
			counterMoves[i][j] = EmptyMove
		}
	}
	lastMovePlayed = EmptyMove
}

type moveScore struct {
	move  Move
	score int
}

func orderMoves(moves []Move) []Move {
	return orderMovesWithDepth(moves, 0)
}

func orderMovesWithDepth(moves []Move, depth int) []Move {
	return orderMovesWithDepthAndPrevious(moves, depth, GetLastMovePlayed())
}

func orderMovesWithDepthAndTTMove(moves []Move, depth int, ttMove Move, ttHit bool) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), ttMove, ttHit)
}

func orderMovesWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool) []Move {
	if len(moves) == 0 {
		return moves
	}
	
	moveScores := make([]moveScore, len(moves))
	
	for i, move := range moves {
		moveScores[i] = moveScore{
			move:  move,
			score: scoreMoveForOrderingWithDepthPreviousAndTTMove(move, depth, previousMove, ttMove, ttHit),
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

func orderMovesWithDepthAndPrevious(moves []Move, depth int, previousMove Move) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, previousMove, EmptyMove, false)
}

func scoreMoveForOrdering(move Move) int {
	return scoreMoveForOrderingWithDepth(move, 0)
}

func scoreMoveForOrderingWithDepth(move Move, depth int) int {
	return scoreMoveForOrderingWithDepthAndPrevious(move, depth, EmptyMove)
}

func scoreMoveForOrderingWithDepthPreviousAndTTMove(move Move, depth int, previousMove Move, ttMove Move, ttHit bool) int {
	// Transposition table move gets highest priority
	if ttHit && move == ttMove {
		return 50000 // Higher than any other move type
	}
	
	return scoreMoveForOrderingWithDepthAndPrevious(move, depth, previousMove)
}

func scoreMoveForOrderingWithDepthAndPrevious(move Move, depth int, previousMove Move) int {
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
	
	// For quiet moves, apply heuristics in order of priority
	if !move.IsCapture() && move.PromoType() == NoType && !move.IsEnPassant() {
		// Counter moves get highest priority among quiet moves
		if IsCounterMove(move, previousMove) {
			score += 8500 // Between captures and killer moves
		}
		
		// Killer moves get high priority but below counter moves
		if IsKillerMove(move, depth) {
			score += 8000
		}
		
		// History heuristic for quiet moves (lowest priority)
		score += GetHistoryScore(move)
	}
	
	return score
}