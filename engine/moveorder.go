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
    
    // Staged move ordering - no allocations, no sorting
    orderedMoves := make([]Move, 0, len(moves))
	
	// Stage 1: TT move (if exists)
	if ttHit && ttMove != EmptyMove {
		for _, move := range moves {
			if move == ttMove {
				orderedMoves = append(orderedMoves, move)
				break
			}
		}
	}
	
	// Stage 2: Good captures (MVV-LVA positive)
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue // Already added in Stage 1
		}
		if move.IsCapture() || move.IsEnPassant() {
			victim := move.CapturedPiece()
			attacker := move.MovingPiece()
			// Add if capture is likely good (victim >= attacker or special case)
			if victim.Weight() >= attacker.Weight() || move.IsEnPassant() {
				orderedMoves = append(orderedMoves, move)
			}
		}
	}
	
	// Stage 3: Promotions
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue
		}
		if move.PromoType() != NoType && !move.IsCapture() {
			orderedMoves = append(orderedMoves, move)
		}
	}
	
	// Stage 4: Killer moves
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue
		}
		if !move.IsCapture() && move.PromoType() == NoType && !move.IsEnPassant() && IsKillerMove(move, depth) {
			orderedMoves = append(orderedMoves, move)
		}
	}
	
	// Stage 5: Counter moves
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue
		}
		if !move.IsCapture() && move.PromoType() == NoType && !move.IsEnPassant() && !IsKillerMove(move, depth) && IsCounterMove(move, previousMove) {
			orderedMoves = append(orderedMoves, move)
		}
	}
	
	// Stage 6: Castling
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue
		}
		if move.IsCastle() {
			orderedMoves = append(orderedMoves, move)
		}
	}
	
	// Stage 7: Bad captures (MVV-LVA negative)
	for _, move := range moves {
		if move == ttMove && ttHit {
			continue
		}
		if move.IsCapture() && !move.IsEnPassant() {
			victim := move.CapturedPiece()
			attacker := move.MovingPiece()
			// Add if capture is likely bad (victim < attacker)
			if victim.Weight() < attacker.Weight() {
				orderedMoves = append(orderedMoves, move)
			}
		}
	}
	
	// Stage 8: Remaining quiet moves (add in history score order without allocations)
	// Use a simple O(n²) selection approach to avoid allocations for small sets
	remainingMoves := make([]bool, len(moves))  // Track which moves we've added
	
	// Mark moves we've already added
	for i, move := range moves {
		if move == ttMove && ttHit {
			remainingMoves[i] = true
			continue
		}
		if move.IsCapture() || move.IsEnPassant() || move.PromoType() != NoType || 
		   move.IsCastle() || IsKillerMove(move, depth) || IsCounterMove(move, previousMove) {
			remainingMoves[i] = true
		}
	}
	
	// Add remaining quiet moves in history score order (without allocations)
	for remainingCount := 0; remainingCount < len(moves); remainingCount++ {
		bestIdx := -1
		bestScore := -1
		
		for i, move := range moves {
			if remainingMoves[i] {
				continue
			}
			
			score := GetHistoryScore(move)
			if score > bestScore {
				bestScore = score
				bestIdx = i
			}
		}
		
		if bestIdx == -1 {
			break // No more moves to add
		}
		
		orderedMoves = append(orderedMoves, moves[bestIdx])
		remainingMoves[bestIdx] = true
	}
	
    return orderedMoves
}

// Zero-allocation ordering into caller-provided buffer
// Returns number of moves written to out (equals len(moves))
func orderMovesIntoBufferWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move) int {
    n := len(moves)
    if n == 0 {
        return 0
    }
    // Track which indices are already emitted
    var used [256]bool
    outCount := 0

    // Stage 1: TT move
    if ttHit && ttMove != EmptyMove {
        for i, m := range moves {
            if m == ttMove {
                out[outCount] = m
                outCount++
                used[i] = true
                break
            }
        }
    }

    // Helper to emit if not used yet
    emit := func(i int) {
        if !used[i] {
            out[outCount] = moves[i]
            outCount++
            used[i] = true
        }
    }

    // Stage 2: Good captures (MVV-LVA positive) and en passant
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if m.IsCapture() || m.IsEnPassant() {
            victim := m.CapturedPiece()
            attacker := m.MovingPiece()
            if m.IsEnPassant() || victim.Weight() >= attacker.Weight() {
                emit(i)
            }
        }
    }

    // Stage 3: Promotions (non-capturing)
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if m.PromoType() != NoType && !m.IsCapture() {
            emit(i)
        }
    }

    // Stage 4: Killer moves
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if !m.IsCapture() && m.PromoType() == NoType && !m.IsEnPassant() && IsKillerMove(m, depth) {
            emit(i)
        }
    }

    // Stage 5: Counter moves
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if !m.IsCapture() && m.PromoType() == NoType && !m.IsEnPassant() && !IsKillerMove(m, depth) && IsCounterMove(m, previousMove) {
            emit(i)
        }
    }

    // Stage 6: Castling
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if m.IsCastle() {
            emit(i)
        }
    }

    // Stage 7: Bad captures (MVV-LVA negative)
    for i, m := range moves {
        if used[i] || (ttHit && m == ttMove) {
            continue
        }
        if m.IsCapture() && !m.IsEnPassant() {
            victim := m.CapturedPiece()
            attacker := m.MovingPiece()
            if victim.Weight() < attacker.Weight() {
                emit(i)
            }
        }
    }

    // Stage 8: Remaining quiet moves by history score (selection loop)
    // Avoid allocations by using selection style with used[]
    for {
        bestIdx := -1
        bestScore := -1
        for i, m := range moves {
            if used[i] || (ttHit && m == ttMove) {
                continue
            }
            if m.IsCapture() || m.IsEnPassant() || m.PromoType() != NoType || m.IsCastle() || IsKillerMove(m, depth) || IsCounterMove(m, previousMove) {
                continue // should have been handled already
            }
            s := GetHistoryScore(m)
            if s > bestScore {
                bestScore = s
                bestIdx = i
            }
        }
        if bestIdx == -1 {
            break
        }
        emit(bestIdx)
    }

    return outCount
}

func orderMovesIntoBufferWithDepth(moves []Move, depth int, out []Move) int {
    return orderMovesIntoBufferWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), EmptyMove, false, out)
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
