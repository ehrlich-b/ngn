package engine

import (
	"math"
	"time"
)

const (
	INFINITY            = int(math.MaxInt32)
	MATE_VALUE          = 30000
	MATE_IN_MAX         = MATE_VALUE - 1000
	NULL_MOVE_R         = 3  // Null move reduction depth
	NULL_MOVE_MIN_DEPTH = 3  // Minimum depth for null move pruning
	LMR_FULL_DEPTH      = 3  // Minimum depth for LMR
	LMR_MOVE_THRESHOLD  = 3  // Start reducing after this many moves
	FUTILITY_MARGIN     = 200  // Futility pruning margin
)

// Search debug logging removed for production

type SearchInfo struct {
	Nodes       uint64
	Depth       int
	BestMove    Move
	BestScore   int
	Stopped     bool
	TimeManager *TimeManager
	PV          [MaximumDepth]Move  // Principal variation array
	PVLength    int                 // Length of current PV
}

func Search(pos *Position, depth int) *SearchInfo {
	return WrapSearch("Search", func() *SearchInfo {
		return SearchIterativeDeepening(pos, depth, nil)
	}, pos, depth)
}

func SearchIterativeDeepening(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
	return WrapSearch("SearchIterativeDeepening", func() *SearchInfo {
		return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager)
	}, pos, maxDepth)
}

func searchIterativeDeepeningUnsafe(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
	// Advance the age of the transposition table for each search
	TranspositionTable.AdvanceAge()
	
	info := &SearchInfo{
		Nodes:       0,
		Depth:       maxDepth,
		BestMove:    EmptyMove,
		BestScore:   -INFINITY,
		Stopped:     false,
		TimeManager: timeManager,
	}
	
	// Generate moves once for iterative deepening
	moves := GenerateLegalMoves(pos)
	if len(moves) == 0 {
		if isInCheck(pos, pos.Turn()) {
			info.BestScore = -MATE_VALUE
		} else {
			info.BestScore = 0
		}
		return info
	}
	
	// Iterative deepening - start from depth 1 and go up to maxDepth
	previousScore := 0 // Track previous iteration score for aspiration windows
	
	for currentDepth := 1; currentDepth <= maxDepth && !info.Stopped; currentDepth++ {
		// Check time before starting each new depth iteration
		if info.TimeManager != nil {
			if info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}
		}
		
		// Aspiration Windows: use previous iteration score to set a narrow window
		var alpha, beta int
		if currentDepth == 1 {
			// Use full window for first iteration
			alpha = -INFINITY
			beta = INFINITY
		} else {
			// Use aspiration window based on previous iteration
			window := 50 // 50 centipawn window
			alpha = previousScore - window
			beta = previousScore + window
		}
		
		// Aspiration window re-search loop
		iterationComplete := false
		attempts := 0
		maxAttempts := 4 // Prevent infinite loops
		
		for !iterationComplete && attempts < maxAttempts {
			attempts++
			
			// Check time before attempting search
			if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}
			
			// Root search with aspiration window
			orderedMoves := orderMovesWithDepth(moves, currentDepth)
			iterationBestScore := -INFINITY
			iterationBestMove := EmptyMove
			
			for _, move := range orderedMoves {
				// Check time before each move
				if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
					info.Stopped = true
					break
				}
				
				ep, tag, hc, _ := pos.MakeMove(move)
				score := -alphaBeta(pos, currentDepth-1, -beta, -alpha, info)
				pos.UnMakeMove(move, tag, ep, hc)
				
				info.Nodes++
				
				if score > iterationBestScore {
					iterationBestScore = score
					iterationBestMove = move
				}
				
				if score > alpha {
					alpha = score
				}
				
				// Beta cutoff
				if alpha >= beta {
					break
				}
			}
			
			// Check if we need to widen the window
			if iterationBestScore <= (previousScore - 50) && currentDepth > 1 {
				// Fail low - widen alpha
				alpha -= 100
				if attempts == maxAttempts-1 {
					alpha = -INFINITY // Use full window on last attempt
				}
			} else if iterationBestScore >= (previousScore + 50) && currentDepth > 1 {
				// Fail high - widen beta  
				beta += 100
				if attempts == maxAttempts-1 {
					beta = INFINITY // Use full window on last attempt
				}
			} else {
				// Search completed successfully
				info.BestScore = iterationBestScore
				info.BestMove = iterationBestMove
				previousScore = iterationBestScore
				iterationComplete = true
			}
		}
		
		// Store the current depth result - this gives us a valid answer
		// even if the search is interrupted at higher depths
		info.Depth = currentDepth
	}
	
	return info
}

func SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	return WrapSearch("SearchFixed", func() *SearchInfo {
		return searchFixedUnsafe(pos, depth, timeManager)
	}, pos, depth)
}

func searchFixedUnsafe(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	
	info := &SearchInfo{
		Nodes:       0,  // Fresh node count for each depth
		Depth:       depth,
		BestMove:    EmptyMove,
		BestScore:   -INFINITY,
		Stopped:     false,
		TimeManager: timeManager,
	}
	
	alpha := -INFINITY
	beta := INFINITY
	
	moves := GenerateLegalMoves(pos)
	if len(moves) == 0 {
		if isInCheck(pos, pos.Turn()) {
			info.BestScore = -MATE_VALUE
		} else {
			info.BestScore = 0
		}
		return info
	}
	
	orderedMoves := orderMovesWithDepth(moves, depth)
	
	for _, move := range orderedMoves {
		// Check if search was stopped
		if info.Stopped {
			break
		}
		
		// Check time before each move in fixed depth search
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}
		
		ep, tag, hc, _ := pos.MakeMove(move)
		score := -alphaBeta(pos, depth-1, -beta, -alpha, info)
		pos.UnMakeMove(move, tag, ep, hc)
		
		info.Nodes++
		
		if score > info.BestScore {
			info.BestScore = score
			info.BestMove = move
		}
		
		if score > alpha {
			alpha = score
		}
		
		if alpha >= beta {
			break
		}
	}
	
	return info
}

func alphaBeta(pos *Position, depth int, alpha int, beta int, info *SearchInfo) int {
	// Safety checks - reasonable limits
	if info.Nodes > 50000000 { // 50M nodes
		info.Stopped = true
		return Evaluate(&pos.Board)
	}
	
	// Check time limit if we have a time manager (now cheap to call frequently)
	if info.TimeManager != nil {
		if info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			return Evaluate(&pos.Board)
		}
	}
	
	// Safety check for excessive depth (use MaximumDepth constant)
	if depth > MaximumDepth {
		return Evaluate(&pos.Board)
	}
	
	// Safety check for negative depth (bug prevention)
	if depth < 0 {
		return Evaluate(&pos.Board)
	}
	
	if depth == 0 {
		return quiescence(pos, alpha, beta, info)
	}
	
	// Probe transposition table
	origAlpha := alpha
	hash := pos.Hash()
	
	// NOTE: Threefold repetition checking removed from hot path for performance
	// This should only be checked at root level, not during hypothetical search nodes
	ttMove, ttEval, ttDepth, ttNodeType, ttHit := TranspositionTable.Get(hash)
	
	if ttHit && ttDepth >= int8(depth) {
		switch ttNodeType {
		case Exact:
			return int(ttEval)
		case LowerBound:
			if int(ttEval) >= beta {
				return int(ttEval)
			}
		case UpperBound:
			if int(ttEval) <= alpha {
				return int(ttEval)
			}
		}
	}
	
	// Mate distance pruning - don't look for mates that are too far away
	currentDepth := info.Depth - depth
	mateAlpha := -MATE_VALUE + currentDepth
	mateBeta := MATE_VALUE - currentDepth - 1
	
	if alpha < mateAlpha {
		alpha = mateAlpha
	}
	if beta > mateBeta {
		beta = mateBeta
	}
	if alpha >= beta {
		return alpha
	}
	
	// Check if we're in check before attempting null move
	inCheck := isInCheck(pos, pos.Turn())
	
	// Futility pruning for leaf nodes (depth 1)
	var futilityPrune bool
	var staticEval int
	if depth == 1 && !inCheck {
		staticEval = Evaluate(&pos.Board)
		if staticEval + FUTILITY_MARGIN < alpha {
			futilityPrune = true
		}
	}
	
	// Null move pruning - only if we're not in check and depth is sufficient
	if !inCheck && depth >= NULL_MOVE_MIN_DEPTH && beta - alpha > 1 {
		// Make the null move
		oldEP := pos.MakeNullMove()
		
		// Search with reduced depth
		score := -alphaBeta(pos, depth-NULL_MOVE_R-1, -beta, -beta+1, info)
		
		// Unmake the null move
		pos.UnMakeNullMove(oldEP)
		
		// If null move search caused a beta cutoff, we can prune
		if score >= beta {
			return beta
		}
	}
	
	// Use pseudo-legal moves for performance, check legality only when trying moves
	moves := GenerateMoves(pos)
	if len(moves) == 0 {
		if inCheck {
			return -MATE_VALUE + (info.Depth - depth)
		}
		return 0
	}
	
	// Order moves, prioritizing TT move if available
	orderedMoves := orderMovesWithDepthAndTTMove(moves, depth, ttMove, ttHit)
	bestScore := -INFINITY
	bestMove := EmptyMove
	moveCount := 0
	
	for _, move := range orderedMoves {
		moveCount++
		
		// Check time before evaluating each move (now cheap to call)
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}
		
		// Futility pruning - skip quiet moves if position is hopeless
		if futilityPrune && !move.IsCapture() {
			continue
		}
		
		ep, tag, hc, _ := pos.MakeMove(move)
		
		// Check if move is legal (doesn't leave our king in check)
		if isInCheck(pos, move.MovingPiece().Color()) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue // Skip illegal move
		}
		
		var score int
		
		// Check for check extensions
		givesCheck := isInCheck(pos, pos.Turn())
		extension := 0
		if givesCheck {
			extension = 1  // Extend search by 1 ply when giving check
		}
		
		// Principal Variation Search (PVS) implementation
		if moveCount == 1 {
			// Search first move (presumably best) with full window
			score = -alphaBeta(pos, depth-1+extension, -beta, -alpha, info)
		} else {
			// For remaining moves, use Principal Variation Search
			
			// Late Move Reductions (LMR) for non-PV moves
			reduction := 0
			if depth >= LMR_FULL_DEPTH && moveCount > LMR_MOVE_THRESHOLD && 
			   !move.IsCapture() && !inCheck && !givesCheck {
				reduction = 1
			}
			
			// Search with null window first (faster)
			score = -alphaBeta(pos, depth-1-reduction+extension, -alpha-1, -alpha, info)
			
			// If null window search fails high, re-search with full window
			if score > alpha && score < beta {
				if reduction > 0 {
					// First re-search with full depth but still null window
					score = -alphaBeta(pos, depth-1+extension, -alpha-1, -alpha, info)
				}
				// If still fails high, do full window search
				if score > alpha {
					score = -alphaBeta(pos, depth-1+extension, -beta, -alpha, info)
				}
			}
		}
		
		pos.UnMakeMove(move, tag, ep, hc)
		
		info.Nodes++
		
		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		
		if score >= beta {
			// Update history table for quiet moves that cause beta cutoffs
			UpdateHistoryTable(move, depth)
			// Update killer moves for quiet moves that cause beta cutoffs
			UpdateKillerMoves(move, depth)
			// Update counter move for quiet moves that cause beta cutoffs
			UpdateCounterMove(move, GetLastMovePlayed())
			
			// Store in transposition table as Lower Bound (Cut-Node)
			TranspositionTable.Set(hash, move, int16(score), int8(depth), LowerBound)
			return score
		}
		
		if score > alpha {
			alpha = score
		}
	}
	
	// Store in transposition table
	var nodeType NodeType
	if bestScore <= origAlpha {
		// No move improved alpha - Upper Bound (All-Node)
		nodeType = UpperBound
	} else {
		// Best move improved alpha - Exact (PV-Node)
		nodeType = Exact
	}
	
	TranspositionTable.Set(hash, bestMove, int16(bestScore), int8(depth), nodeType)
	return bestScore
}

func quiescence(pos *Position, alpha int, beta int, info *SearchInfo) int {
	return quiescenceWithDepth(pos, alpha, beta, info, 0)
}

func quiescenceWithDepth(pos *Position, alpha int, beta int, info *SearchInfo, qDepth int) int {
	// Prevent infinite recursion - reduce depth limit for stability
	if qDepth >= 6 { // More conservative quiescence depth
		return Evaluate(&pos.Board)
	}
	
	// Stand pat - evaluate current position
	standPat := Evaluate(&pos.Board)
	info.Nodes++
	
	if standPat >= beta {
		return beta
	}
	
	if standPat > alpha {
		alpha = standPat
	}
	
	// Only consider captures and checks in quiescence search
	// Start with captures (more efficient than generating all moves)
	capturesAndChecks := GenerateCaptures(pos)
	
	// Also consider moves that give check (but only if not too deep to avoid explosion)
	if qDepth < 2 { // Very conservative check extension depth
		moves := GenerateMoves(pos)
		for _, move := range moves {
			// Skip if already a capture
			if move.IsCapture() {
				continue
			}
			
			// Check if move gives check
			ep, tag, hc, _ := pos.MakeMove(move)
			givesCheck := isInCheck(pos, pos.Turn())
			pos.UnMakeMove(move, tag, ep, hc)
			
			if givesCheck {
				capturesAndChecks = append(capturesAndChecks, move)
			}
		}
	}
	
	// Search captures and checks
	for _, move := range capturesAndChecks {
		ep, tag, hc, _ := pos.MakeMove(move)
		
		// Check if move is legal (doesn't leave our king in check)
		if isInCheck(pos, move.MovingPiece().Color()) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue // Skip illegal move
		}
		
		score := -quiescenceWithDepth(pos, -beta, -alpha, info, qDepth+1)
		pos.UnMakeMove(move, tag, ep, hc)
		
		if score >= beta {
			return beta
		}
		
		if score > alpha {
			alpha = score
		}
	}
	
	return alpha
}
