package engine

import (
	"fmt"
	"sort"
	"testing"
)

// TestSearchDiagnostics creates a position where the engine should easily capture a free piece
// and tracks exactly what the search is considering and why it's making bad decisions
func TestSearchDiagnostics(t *testing.T) {
	// Position where white can capture a free rook with Qg4xg7
	// This should be an obvious tactical gain
	fen := "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 0 1"
	
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatal("Failed to parse FEN:", err)
	}
	
	// Generate legal moves and show what's available
	t.Logf("=== DIAGNOSTIC TEST FOR POSITION ===")
	t.Logf("FEN: %s", fen)
	t.Logf("White to move, should see capture opportunities")
	
	// 1. Test move generation - are good moves even being generated?
	moves := GenerateLegalMoves(pos)
	t.Logf("\n=== LEGAL MOVES GENERATED (%d total) ===", len(moves))
	
	type moveInfo struct {
		move     Move
		notation string
		isCapture bool
		captured string
		score    int
	}
	
	var moveList []moveInfo
	
	for _, move := range moves {
		notation := move.ToString()
		isCapture := move.IsCapture()
		captured := ""
		if isCapture {
			captured = move.CapturedPiece().Name()
		}
		
		// Get a quick evaluation of the move
		ep, tag, hc, _ := pos.MakeMove(move)
		score := EvaluateForPlayer(&pos.Board, White)  // Evaluate from white's perspective
		pos.UnMakeMove(move, tag, ep, hc)
		
		moveList = append(moveList, moveInfo{
			move:     move,
			notation: notation,
			isCapture: isCapture,
			captured: captured,
			score:    score,
		})
	}
	
	// Sort moves by evaluation score (best first)
	sort.Slice(moveList, func(i, j int) bool {
		return moveList[i].score > moveList[j].score
	})
	
	t.Logf("Top 10 moves by static evaluation:")
	for i, info := range moveList {
		if i >= 10 { break }
		capInfo := ""
		if info.isCapture {
			capInfo = fmt.Sprintf(" (captures %s)", info.captured)
		}
		t.Logf("%2d. %s%s - Score: %d", i+1, info.notation, capInfo, info.score)
	}
	
	// 2. Test what the search algorithm chooses
	t.Logf("\n=== SEARCH RESULTS ===")
	
	// Run search at different depths to see consistency
	for depth := 1; depth <= 4; depth++ {
		searchInfo := SearchFixed(pos, depth, nil)
		bestMove := searchInfo.BestMove
		bestScore := searchInfo.BestScore
		
		moveNotation := "none"
		if bestMove != EmptyMove {
			moveNotation = bestMove.ToString()
		}
		
		t.Logf("Depth %d: Best move = %s, Score = %d, Nodes = %d", 
			depth, moveNotation, bestScore, searchInfo.Nodes)
	}
	
	// 4. Check evaluation components
	t.Logf("\n=== POSITION EVALUATION BREAKDOWN ===")
	
	currentEval := Evaluate(&pos.Board)
	t.Logf("Current position evaluation: %d (positive = good for white)", currentEval)
	
	// Test if evaluation is sane by checking basic material count
	materialBalance := 0
	
	// Count white pieces
	whiteMaterial := PopCount(pos.Board.GetBitboardOf(WhitePawn)) * 100 +
		PopCount(pos.Board.GetBitboardOf(WhiteKnight)) * 320 +
		PopCount(pos.Board.GetBitboardOf(WhiteBishop)) * 330 +
		PopCount(pos.Board.GetBitboardOf(WhiteRook)) * 500 +
		PopCount(pos.Board.GetBitboardOf(WhiteQueen)) * 900
	
	// Count black pieces
	blackMaterial := PopCount(pos.Board.GetBitboardOf(BlackPawn)) * 100 +
		PopCount(pos.Board.GetBitboardOf(BlackKnight)) * 320 +
		PopCount(pos.Board.GetBitboardOf(BlackBishop)) * 330 +
		PopCount(pos.Board.GetBitboardOf(BlackRook)) * 500 +
		PopCount(pos.Board.GetBitboardOf(BlackQueen)) * 900
	
	materialBalance = whiteMaterial - blackMaterial
	t.Logf("Pure material balance: %d (white: %d, black: %d)", materialBalance, whiteMaterial, blackMaterial)
	
	// 5. Test if the engine sees a simple tactical sequence
	t.Logf("\n=== TACTICAL ANALYSIS ===")
	
	// Look for captures in the top-rated moves
	captureCount := 0
	for i, info := range moveList {
		if i >= 5 { break } // Top 5 moves
		if info.isCapture {
			captureCount++
			t.Logf("Found capture in top 5: %s captures %s", info.notation, info.captured)
		}
	}
	
	if captureCount == 0 {
		t.Logf("⚠️  WARNING: No captures found in top 5 moves")
	} else {
		t.Logf("✅ Found %d captures in top 5 moves", captureCount)
	}
}

// TestEvaluationConsistency checks if evaluation is behaving consistently
func TestEvaluationConsistency(t *testing.T) {
	positions := []struct {
		name string
		fen  string
		expectation string
	}{
		{
			name: "Starting position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			expectation: "roughly equal",
		},
		{
			name: "White up a queen",
			fen:  "rnbk1bnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQ - 0 1",
			expectation: "strongly positive",
		},
		{
			name: "Black up a queen", 
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNB1KBNR w KQ - 0 1",
			expectation: "strongly negative",
		},
	}
	
	for _, tc := range positions {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatal("Failed to parse FEN:", err)
			}
			
			eval := Evaluate(&pos.Board)
			t.Logf("%s: Evaluation = %d (%s)", tc.name, eval, tc.expectation)
			
			// Sanity checks
			switch tc.expectation {
			case "roughly equal":
				if absInt(eval) > 100 {
					t.Logf("⚠️  WARNING: Expected roughly equal but got %d", eval)
				}
			case "strongly positive":
				if eval < 700 { // Should be at least queen value minus some
					t.Logf("⚠️  WARNING: Expected strongly positive but got %d", eval)
				}
			case "strongly negative":
				if eval > -700 {
					t.Logf("⚠️  WARNING: Expected strongly negative but got %d", eval)
				}
			}
		})
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
