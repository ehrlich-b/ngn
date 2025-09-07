package engine

import (
	"testing"
)

func TestMaterialEvaluation(t *testing.T) {
	// Test basic material counting
	board := Bitboard{}

	// Starting position should be roughly balanced
	board = StartingBoard()
	eval := Evaluate(&board)

	// Starting position should be close to 0 (may not be exactly 0 due to PST values)
	if eval > 100 || eval < -100 {
		t.Errorf("Starting position evaluation should be close to 0, got %d", eval)
	}

	// Test position with material advantage for white
	board = Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)
	board.UpdateSquare(D1, WhiteQueen, NoPiece) // White has extra queen

	eval = Evaluate(&board)
	if eval <= 900 { // Should be strongly positive for white (queen = 1000 - some PST penalty)
		t.Errorf("White should have large material advantage, got %d", eval)
	}
}

func TestPieceSquareTables(t *testing.T) {
	// Test that central squares are valued higher for pieces
	board := Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)

	// Knight on edge vs center
	board.UpdateSquare(A1, WhiteKnight, NoPiece)
	evalEdge := Evaluate(&board)

	board.UpdateSquare(A1, NoPiece, WhiteKnight)
	board.UpdateSquare(E4, WhiteKnight, NoPiece) // Central knight
	evalCenter := Evaluate(&board)

	if evalCenter <= evalEdge {
		t.Errorf("Central knight should be valued higher than edge knight. Edge: %d, Center: %d", evalEdge, evalCenter)
	}
}

func TestPawnStructureEvaluation(t *testing.T) {
	// Test doubled pawns penalty
	board := Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)

	// Two pawns on same file (doubled)
	board.UpdateSquare(E2, WhitePawn, NoPiece)
	board.UpdateSquare(E3, WhitePawn, NoPiece)
	evalDoubled := Evaluate(&board)

	// Two pawns on different files (better structure)
	board.UpdateSquare(E3, NoPiece, WhitePawn) // Remove doubled pawn
	board.UpdateSquare(D2, WhitePawn, NoPiece) // Add pawn on different file
	evalNormal := Evaluate(&board)

	// Normal structure should be better than doubled pawns
	if evalDoubled >= evalNormal {
		t.Errorf("Doubled pawns should be worse than normal pawn structure. Doubled: %d, Normal: %d", evalDoubled, evalNormal)
	}

	// Test isolated pawn penalty
	board = Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)

	// Connected pawns
	board.UpdateSquare(D2, WhitePawn, NoPiece)
	board.UpdateSquare(E2, WhitePawn, NoPiece)
	evalConnected := Evaluate(&board)

	// Isolated pawn
	board.UpdateSquare(D2, NoPiece, WhitePawn)
	evalIsolated := Evaluate(&board)

	if evalIsolated >= evalConnected {
		t.Errorf("Isolated pawn should be penalized. Connected: %d, Isolated: %d", evalConnected, evalIsolated)
	}
}

func TestPassedPawnBonus(t *testing.T) {
	// Test passed pawn bonus
	board := Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)

	// White pawn with black pawn blocking
	board.UpdateSquare(E4, WhitePawn, NoPiece)
	board.UpdateSquare(E5, BlackPawn, NoPiece) // Blocking pawn
	evalBlocked := Evaluate(&board)

	// Remove blocking pawn - now it's passed
	board.UpdateSquare(E5, NoPiece, BlackPawn)
	evalPassed := Evaluate(&board)

	if evalPassed <= evalBlocked {
		t.Errorf("Passed pawn should get bonus. Blocked: %d, Passed: %d", evalBlocked, evalPassed)
	}
}

func TestKingSafety(t *testing.T) {
	// Test king safety with pawn shield
	board := Bitboard{}

	// King without pawn shield
	board.UpdateSquare(G1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)
	evalNoShield := Evaluate(&board)

	// King with pawn shield
	board.UpdateSquare(F2, WhitePawn, NoPiece)
	board.UpdateSquare(G2, WhitePawn, NoPiece)
	board.UpdateSquare(H2, WhitePawn, NoPiece)
	evalWithShield := Evaluate(&board)

	if evalWithShield <= evalNoShield {
		t.Errorf("King with pawn shield should be safer. No shield: %d, With shield: %d", evalNoShield, evalWithShield)
	}
}

func TestEndgameDetection(t *testing.T) {
	// Test endgame detection
	board := Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)

	// Middlegame - lots of pieces
	board.UpdateSquare(D1, WhiteQueen, NoPiece)
	board.UpdateSquare(A1, WhiteRook, NoPiece)
	board.UpdateSquare(C1, WhiteBishop, NoPiece)
	board.UpdateSquare(B1, WhiteKnight, NoPiece)

	board.UpdateSquare(D8, BlackQueen, NoPiece)
	board.UpdateSquare(A8, BlackRook, NoPiece)
	board.UpdateSquare(C8, BlackBishop, NoPiece)
	board.UpdateSquare(B8, BlackKnight, NoPiece)

	if isEndgame(&board) {
		t.Error("Should not detect endgame with many pieces")
	}

	// Remove most pieces for endgame
	board.UpdateSquare(D1, NoPiece, WhiteQueen)
	board.UpdateSquare(A1, NoPiece, WhiteRook)
	board.UpdateSquare(C1, NoPiece, WhiteBishop)

	board.UpdateSquare(D8, NoPiece, BlackQueen)
	board.UpdateSquare(A8, NoPiece, BlackRook)
	board.UpdateSquare(C8, NoPiece, BlackBishop)

	if !isEndgame(&board) {
		t.Error("Should detect endgame with few pieces")
	}
}

func TestMobilityEvaluation(t *testing.T) {
	// Test mobility evaluation
	// Create position with different mobility
	pos := &Position{}
	pos.Board = Bitboard{}
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)

	// White queen with lots of mobility
	pos.Board.UpdateSquare(D4, WhiteQueen, NoPiece) // Central queen

	// Black queen trapped in corner
	pos.Board.UpdateSquare(A8, BlackQueen, NoPiece)
	pos.Board.UpdateSquare(A7, BlackPawn, NoPiece) // Blocking pawn
	pos.Board.UpdateSquare(B8, BlackRook, NoPiece) // Blocking rook

	pos.SetTag(WhiteToMove)
	pos.EnPassant = NoSquare

	eval := Evaluate(&pos.Board)

	// The evaluation should run without crashing and give some reasonable value
	// (The exact sign depends on many factors including PST values and material)
	if eval == 0 {
		t.Errorf("Evaluation should not be exactly zero with pieces on board, got %d", eval)
	}
}

func TestEvaluationSymmetry(t *testing.T) {
	// Test that evaluation is symmetric for mirrored positions
	board := Bitboard{}

	// Set up a position from white's perspective
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(D1, WhiteQueen, NoPiece)
	board.UpdateSquare(E2, WhitePawn, NoPiece)
	board.UpdateSquare(E8, BlackKing, NoPiece)
	board.UpdateSquare(D7, BlackPawn, NoPiece)

	evalWhite := Evaluate(&board)

	// Create mirrored position (swap colors and flip board)
	board2 := Bitboard{}
	board2.UpdateSquare(E8, BlackKing, NoPiece)
	board2.UpdateSquare(D8, BlackQueen, NoPiece)
	board2.UpdateSquare(E7, BlackPawn, NoPiece)
	board2.UpdateSquare(E1, WhiteKing, NoPiece)
	board2.UpdateSquare(D2, WhitePawn, NoPiece)

	evalBlack := Evaluate(&board2)

	// The evaluations should be roughly opposite
	if evalWhite*evalBlack >= 0 {
		t.Errorf("Symmetric positions should have opposite evaluations. White: %d, Black: %d", evalWhite, evalBlack)
	}
}

func BenchmarkEvaluation(b *testing.B) {
	board := StartingBoard()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(&board)
	}
}

func BenchmarkEvaluationComplex(b *testing.B) {
	// More complex position with many pieces
	board := Bitboard{}
	board.UpdateSquare(E1, WhiteKing, NoPiece)
	board.UpdateSquare(D1, WhiteQueen, NoPiece)
	board.UpdateSquare(A1, WhiteRook, NoPiece)
	board.UpdateSquare(H1, WhiteRook, NoPiece)
	board.UpdateSquare(C1, WhiteBishop, NoPiece)
	board.UpdateSquare(F1, WhiteBishop, NoPiece)
	board.UpdateSquare(B1, WhiteKnight, NoPiece)
	board.UpdateSquare(G1, WhiteKnight, NoPiece)

	for file := 0; file < 8; file++ {
		board.UpdateSquare(Square(8+file), WhitePawn, NoPiece)
		board.UpdateSquare(Square(48+file), BlackPawn, NoPiece)
	}

	board.UpdateSquare(E8, BlackKing, NoPiece)
	board.UpdateSquare(D8, BlackQueen, NoPiece)
	board.UpdateSquare(A8, BlackRook, NoPiece)
	board.UpdateSquare(H8, BlackRook, NoPiece)
	board.UpdateSquare(C8, BlackBishop, NoPiece)
	board.UpdateSquare(F8, BlackBishop, NoPiece)
	board.UpdateSquare(B8, BlackKnight, NoPiece)
	board.UpdateSquare(G8, BlackKnight, NoPiece)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(&board)
	}
}
