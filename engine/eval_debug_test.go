package engine

import (
	"testing"
)

// TestEvaluationDebug debugs why capturing a free pawn gets negative evaluation
func TestEvaluationDebug(t *testing.T) {
	// Position where white knight can capture black pawn on e5
	fen := "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 0 1"
	
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatal("Failed to parse FEN:", err)
	}
	
	t.Logf("=== EVALUATION DEBUG ===")
	t.Logf("FEN: %s", fen)
	
	// Evaluate the position before the capture
	beforeEval := Evaluate(&pos.Board)
	t.Logf("Evaluation before capture: %d", beforeEval)
	
	// Make the capture move f3xe5
	move := NewMove(F3, E5, WhiteKnight, BlackPawn, NoType, Capture) // f3 to e5
	
	t.Logf("Making move: %s", move.ToString())
	t.Logf("Move details: from=%d, to=%d, piece=%v, captured=%v", 
		move.Source(), move.Destination(), move.MovingPiece(), move.CapturedPiece())
	
	ep, tag, hc, _ := pos.MakeMove(move)
	
	// Evaluate after the capture
	afterEval := Evaluate(&pos.Board)
	t.Logf("Evaluation after capture: %d (black's perspective)", afterEval)
	
	// Convert to white's perspective
	whiteAfterEval := -afterEval
	t.Logf("Evaluation after capture: %d (white's perspective)", whiteAfterEval)
	
	improvement := whiteAfterEval - beforeEval
	t.Logf("Improvement: %d", improvement)
	
	if improvement < 0 {
		t.Logf("❌ CRITICAL BUG: Capturing free pawn makes evaluation WORSE by %d", -improvement)
	} else {
		t.Logf("✅ Capturing free pawn improves evaluation by %d", improvement)
	}
	
	pos.UnMakeMove(move, tag, ep, hc)
	
	// Let's also test what happens if we just move the knight somewhere else
	t.Logf("\n=== COMPARISON: NON-CAPTURE KNIGHT MOVE ===")
	
	// Move knight to h4 instead
	nonCaptureMove := NewMove(F3, H4, WhiteKnight, NoPiece, NoType, 0) // f3 to h4
	
	t.Logf("Making non-capture move: %s", nonCaptureMove.ToString())
	
	ep2, tag2, hc2, _ := pos.MakeMove(nonCaptureMove)
	afterEval2 := Evaluate(&pos.Board)
	pos.UnMakeMove(nonCaptureMove, tag2, ep2, hc2)
	
	whiteAfterEval2 := -afterEval2
	improvement2 := whiteAfterEval2 - beforeEval
	
	t.Logf("Non-capture knight move: %d -> improvement: %d", whiteAfterEval2, improvement2)
	
	if improvement2 > improvement {
		t.Logf("❌ CRITICAL BUG: Non-capture move scored %d better than capture!", improvement2 - improvement)
	}
	
	// Let's debug the material count specifically
	t.Logf("\n=== MATERIAL COUNT DEBUG ===")
	
	// Before capture
	whitePawns := PopCount(pos.Board.GetBitboardOf(WhitePawn))
	blackPawns := PopCount(pos.Board.GetBitboardOf(BlackPawn))
	whiteKnights := PopCount(pos.Board.GetBitboardOf(WhiteKnight))
	blackKnights := PopCount(pos.Board.GetBitboardOf(BlackKnight))
	
	t.Logf("Before - White pawns: %d, Black pawns: %d", whitePawns, blackPawns)
	t.Logf("Before - White knights: %d, Black knights: %d", whiteKnights, blackKnights)
	
	// After capture
	pos.MakeMove(move)
	
	whitePawnsAfter := PopCount(pos.Board.GetBitboardOf(WhitePawn))
	blackPawnsAfter := PopCount(pos.Board.GetBitboardOf(BlackPawn))
	whiteKnightsAfter := PopCount(pos.Board.GetBitboardOf(WhiteKnight))
	blackKnightsAfter := PopCount(pos.Board.GetBitboardOf(BlackKnight))
	
	t.Logf("After - White pawns: %d, Black pawns: %d", whitePawnsAfter, blackPawnsAfter)
	t.Logf("After - White knights: %d, Black knights: %d", whiteKnightsAfter, blackKnightsAfter)
	
	materialDiff := (blackPawns - blackPawnsAfter) * 100  // Should be +100 for capturing a pawn
	t.Logf("Material difference: %d (should be +100)", materialDiff)
	
	pos.UnMakeMove(move, tag, ep, hc)
}