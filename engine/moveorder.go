package engine

// History table for move ordering heuristic
// historyTable[from][to] stores the history score for moves from 'from' to 'to'
var historyTable [64][64]int

// Killer moves table for move ordering heuristic
// killerMoves[depth][slot] stores killer moves at each depth (2 killer moves per depth)
var killerMoves [MaximumDepth][2]Move

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
	if !move.IsCapture() && move.PromoType() == NoType && depth >= 0 && depth < MaximumDepth {
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
	if depth < 0 || depth >= MaximumDepth {
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

// Single-pass move ordering with enhanced SEE integration
// Orders moves in one pass using scores, with captures ordered by SEE value
func orderMovesSinglePass(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, pos *Position) []Move {
	if len(moves) == 0 {
		return moves
	}

	// Use zero-allocation buffer version internally
	var buffer [256]Move
	count := orderMovesSinglePassIntoBuffer(moves, depth, previousMove, ttMove, ttHit, buffer[:], pos)

	// Return a copy of the ordered moves (only allocation, but tests need owned slice)
	result := make([]Move, count)
	copy(result, buffer[:count])
	return result
}

// Zero-allocation single-pass ordering into caller-provided buffer
func orderMovesSinglePassIntoBuffer(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move, pos *Position) int {
	n := len(moves)
	if n == 0 || len(out) < n {
		return 0
	}

	// Use a fixed-size array for scoring to avoid allocations
	var scores [256]int

	// Single pass: score all moves
	for i, move := range moves {
		score := 0

		// TT move gets highest priority
		if ttHit && move == ttMove {
			score = 100000
		} else {
			// Calculate score based on move type and SEE
			if move.IsCapture() || move.IsEnPassant() {
				// Use SEE to order captures precisely
				seeValue := staticExchangeEvaluation(pos, move)
				if seeValue >= 0 {
					// Winning/equal captures: base score + SEE value for fine ordering
					score = 80000 + seeValue + 1000
				} else {
					// Losing captures: still better than quiet moves, but ordered by SEE
					score = 60000 + seeValue + 1000
				}
			} else if move.PromoType() != NoType {
				// Promotions: high priority, ordered by promotion piece value
				score = 70000 + int(GetPiece(move.PromoType(), move.MovingPiece().Color()).Weight())
			} else if move.IsCastle() {
				// Castling: moderate priority
				score = 50000
			} else {
				// Quiet moves: use heuristics
				score = 0

				// Counter moves get highest priority among quiet moves
				if IsCounterMove(move, previousMove) {
					score += 30000
				}

				// Killer moves get high priority
				if IsKillerMove(move, depth) {
					score += 25000
				}

				// History heuristic for remaining quiet moves
				score += GetHistoryScore(move)
			}
		}

		scores[i] = score
		out[i] = move
	}

	// Sort the output buffer by score (descending) using selection sort
	for i := 0; i < n-1; i++ {
		maxIdx := i
		for j := i + 1; j < n; j++ {
			if scores[j] > scores[maxIdx] {
				maxIdx = j
			}
		}
		// Swap both scores and moves
		if maxIdx != i {
			scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
			out[i], out[maxIdx] = out[maxIdx], out[i]
		}
	}

	return n
}

func orderMoves(moves []Move, pos *Position) []Move {
	return orderMovesWithDepth(moves, 0, pos)
}

func orderMovesWithDepth(moves []Move, depth int, pos *Position) []Move {
	return orderMovesWithDepthAndPrevious(moves, depth, GetLastMovePlayed(), pos)
}

func orderMovesWithDepthAndTTMove(moves []Move, depth int, ttMove Move, ttHit bool, pos *Position) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), ttMove, ttHit, pos)
}

func orderMovesWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, pos *Position) []Move {
	return orderMovesSinglePass(moves, depth, previousMove, ttMove, ttHit, pos)
}

// Zero-allocation ordering into caller-provided buffer
// Returns number of moves written to out (equals len(moves))
func orderMovesIntoBufferWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move, pos *Position) int {
	return orderMovesSinglePassIntoBuffer(moves, depth, previousMove, ttMove, ttHit, out, pos)
}

func orderMovesIntoBufferWithDepth(moves []Move, depth int, out []Move, pos *Position) int {
	return orderMovesIntoBufferWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), EmptyMove, false, out, pos)
}

func orderMovesWithDepthAndPrevious(moves []Move, depth int, previousMove Move, pos *Position) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, previousMove, EmptyMove, false, pos)
}
