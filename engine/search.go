package engine

import (
	"math"
)

const (
	INFINITY     = int(math.MaxInt32)
	MATE_VALUE   = 30000
	MATE_IN_MAX  = MATE_VALUE - 1000
	MAX_DEPTH    = 100
)

type SearchInfo struct {
	Nodes     uint64
	Depth     int
	BestMove  Move
	BestScore int
	Stopped   bool
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

func SearchFixed(pos *Position, depth int) *SearchInfo {
	info := &SearchInfo{
		Nodes:     0,
		Depth:     depth,
		BestMove:  EmptyMove,
		BestScore: -INFINITY,
		Stopped:   false,
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
	originalAlpha := alpha
	hash := pos.Hash()
	
	// Probe transposition table
	ttMove, ttEval, ttDepth, ttNodeType, ttFound := TranspositionTable.Get(hash)
	
	if ttFound && ttDepth >= int8(depth) {
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
	
	// Try hash move first if we have one
	if ttFound && ttMove != EmptyMove {
		// Move hash move to front
		for i, move := range orderedMoves {
			if move == ttMove {
				orderedMoves[0], orderedMoves[i] = orderedMoves[i], orderedMoves[0]
				break
			}
		}
	}
	
	bestMove := EmptyMove
	bestScore := -INFINITY
	
	for _, move := range orderedMoves {
		ep, tag, hc, _ := pos.MakeMove(move)
		score := -alphaBeta(pos, depth-1, -beta, -alpha, info)
		pos.UnMakeMove(move, tag, ep, hc)
		
		info.Nodes++
		
		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		
		if score >= beta {
			// Store as lower bound (beta cutoff)
			TranspositionTable.Set(hash, bestMove, int16(score), int8(depth), LowerBound)
			return score
		}
		
		if score > alpha {
			alpha = score
		}
	}
	
	// Store in transposition table
	var nodeType NodeType
	if bestScore <= originalAlpha {
		nodeType = UpperBound
	} else {
		nodeType = Exact
	}
	
	TranspositionTable.Set(hash, bestMove, int16(bestScore), int8(depth), nodeType)
	
	return bestScore
}

func quiescence(pos *Position, alpha int, beta int, info *SearchInfo) int {
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
		inCheck := isInCheck(pos, pos.Turn().Other())
		pos.UnMakeMove(move, tag, ep, hc)
		
		if inCheck {
			capturesAndChecks = append(capturesAndChecks, move)
		}
	}
	
	// Search captures and checks
	for _, move := range capturesAndChecks {
		ep, tag, hc, _ := pos.MakeMove(move)
		score := -quiescence(pos, -beta, -alpha, info)
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
