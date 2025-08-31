package engine

import (
	"math"
	"time"
)

const (
	INFINITY     = int(math.MaxInt32)
	MATE_VALUE   = 30000
	MATE_IN_MAX  = MATE_VALUE - 1000
	MAX_DEPTH    = 100
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
	return SearchIterativeDeepening(pos, depth)
}

func SearchIterativeDeepening(pos *Position, maxDepth int) *SearchInfo {
	// Advance the age of the transposition table for each search
	TranspositionTable.AdvanceAge()
	
	info := &SearchInfo{
		Nodes:     0,
		Depth:     maxDepth,
		BestMove:  EmptyMove,
		BestScore: -INFINITY,
		Stopped:   false,
	}
	
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
		alpha := -INFINITY
		beta := INFINITY
		
		orderedMoves := orderMoves(moves)
		
		for _, move := range orderedMoves {
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
	
	orderedMoves := orderMoves(moves)
	
	for _, move := range orderedMoves {
		// Check if search was stopped
		if info.Stopped {
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
	
	// Check time limit if we have a time manager (check every 1024 nodes for performance)
	if info.TimeManager != nil && info.Nodes%1024 == 0 {
		if info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			return Evaluate(&pos.Board)
		}
	}
	
	if depth > 64 { // Reasonable max depth
		return Evaluate(&pos.Board)
	}
	
	if depth == 0 {
		return quiescence(pos, alpha, beta, info)
	}
	
	moves := GenerateLegalMoves(pos)
	if len(moves) == 0 {
		if isInCheck(pos, pos.Turn()) {
			return -MATE_VALUE + (info.Depth - depth)
		}
		return 0
	}
	
	orderedMoves := orderMoves(moves)
	bestScore := -INFINITY
	
	for _, move := range orderedMoves {
		ep, tag, hc, _ := pos.MakeMove(move)
		score := -alphaBeta(pos, depth-1, -beta, -alpha, info)
		pos.UnMakeMove(move, tag, ep, hc)
		
		info.Nodes++
		
		if score > bestScore {
			bestScore = score
		}
		
		if score >= beta {
			return score
		}
		
		if score > alpha {
			alpha = score
		}
	}
	
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
	moves := GenerateLegalMoves(pos)
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
