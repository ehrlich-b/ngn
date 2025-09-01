package engine

import (
	"math"
	"time"
)

const (
	INFINITY            = int(math.MaxInt32)
	MATE_VALUE          = 30000
	MATE_IN_MAX         = MATE_VALUE - 1000
	MAX_DEPTH           = 100
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
}

func Search(pos *Position, depth int) *SearchInfo {
	return SearchIterativeDeepening(pos, depth, nil)
}

func SearchIterativeDeepening(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
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
	for currentDepth := 1; currentDepth <= maxDepth && !info.Stopped; currentDepth++ {
		// Check time before starting each new depth iteration
		if info.TimeManager != nil {
			if info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}
		}
		
		alpha := -INFINITY
		beta := INFINITY
		
		orderedMoves := orderMovesWithDepth(moves, currentDepth)
		
		for _, move := range orderedMoves {
			// Check time before each move in iterative deepening
			if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}
			
			ep, tag, hc, _ := pos.MakeMove(move)
			score := -alphaBeta(pos, currentDepth-1, -beta, -alpha, info)
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
		
		// Store the current depth result - this gives us a valid answer
		// even if the search is interrupted at higher depths
		info.Depth = currentDepth
	}
	
	return info
}

func SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	
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
	
	// Safety check for excessive depth (use MAX_DEPTH constant)
	if depth > MAX_DEPTH {
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
		
		// Late Move Reductions (LMR)
		if depth >= LMR_FULL_DEPTH && moveCount > LMR_MOVE_THRESHOLD && 
		   !move.IsCapture() && !inCheck && !givesCheck {
			
			// Reduce depth by 1 for late moves
			reduction := 1
			
			// Try reduced depth search first
			score = -alphaBeta(pos, depth-1-reduction+extension, -alpha-1, -alpha, info)
			
			// If reduced search suggests the move might be good, do full search
			if score > alpha {
				score = -alphaBeta(pos, depth-1+extension, -beta, -alpha, info)
			}
		} else {
			// Full depth search for important moves
			score = -alphaBeta(pos, depth-1+extension, -beta, -alpha, info)
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
	// Prevent infinite recursion
	if qDepth >= 8 { // Reasonable quiescence depth
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
	// Use pseudo-legal moves for performance
	moves := GenerateMoves(pos)
	capturesAndChecks := make([]Move, 0, len(moves))
	
	for _, move := range moves {
		// Include captures
		if move.IsCapture() {
			capturesAndChecks = append(capturesAndChecks, move)
			continue
		}
		
		// Include moves that give check
		ep, tag, hc, _ := pos.MakeMove(move)
		inCheck := isInCheck(pos, pos.Turn())
		pos.UnMakeMove(move, tag, ep, hc)
		
		if inCheck {
			capturesAndChecks = append(capturesAndChecks, move)
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
