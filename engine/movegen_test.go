package engine

import (
	"testing"
)

func TestBasicMoveGeneration(t *testing.T) {
	// Test move generation from starting position
	pos := &Position{
		Board: StartingBoard(),
		Tag:   WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant: NoSquare, // Make sure en passant is properly initialized
	}
	
	moves := GenerateMoves(pos)
	
	// Starting position should have exactly 20 legal moves
	if len(moves) != 20 {
		t.Errorf("Starting position should have 20 moves, got %d", len(moves))
	}
	
	// Count move types
	pawnMoves := 0
	knightMoves := 0
	for _, move := range moves {
		switch move.MovingPiece() {
		case WhitePawn:
			pawnMoves++
		case WhiteKnight:
			knightMoves++
		}
	}
	
	// Should have 16 pawn moves (8 single pushes, 8 double pushes)
	if pawnMoves != 16 {
		t.Errorf("Expected 16 pawn moves, got %d", pawnMoves)
	}
	
	// Should have 4 knight moves (2 knights × 2 moves each)
	if knightMoves != 4 {
		t.Errorf("Expected 4 knight moves, got %d", knightMoves)
	}
}

func TestPawnMoves(t *testing.T) {
	tests := []struct {
		name        string
		setupBoard  func() *Position
		expectedMin int
		expectedMax int
	}{
		{
			name: "Starting position white pawns",
			setupBoard: func() *Position {
				return &Position{
					Board:     StartingBoard(),
					Tag:       WhiteToMove,
					EnPassant: NoSquare,
				}
			},
			expectedMin: 16,
			expectedMax: 16,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := tt.setupBoard()
			moves := generateWhitePawnMoves(pos)
			
			if len(moves) < tt.expectedMin || len(moves) > tt.expectedMax {
				t.Errorf("%s: expected %d-%d pawn moves, got %d", 
					tt.name, tt.expectedMin, tt.expectedMax, len(moves))
			}
		})
	}
}

func TestKnightMoves(t *testing.T) {
	// Create position with single knight on E4
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhiteKnight, NoPiece)
	
	moves := generateKnightMoves(pos, E4)
	
	// Knight on E4 should have 8 possible moves
	expected := 8
	if len(moves) != expected {
		t.Errorf("Knight on E4 should have %d moves, got %d", expected, len(moves))
	}
	
	// Test moves are correct
	expectedSquares := []Square{D2, F2, C3, G3, C5, G5, D6, F6}
	foundSquares := make([]Square, len(moves))
	for i, move := range moves {
		foundSquares[i] = move.Destination()
	}
	
	for _, expectedSq := range expectedSquares {
		found := false
		for _, foundSq := range foundSquares {
			if foundSq == expectedSq {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected knight move to %s not found", expectedSq.Name())
		}
	}
}

func TestBishopMoves(t *testing.T) {
	// Create position with single bishop on E4
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhiteBishop, NoPiece)
	
	moves := generateBishopMoves(pos, E4)
	
	// Bishop on E4 should have 13 possible moves (4 directions, varying lengths)  
	expected := 13
	if len(moves) != expected {
		t.Errorf("Bishop on E4 should have %d moves, got %d", expected, len(moves))
	}
}

func TestRookMoves(t *testing.T) {
	// Create position with single rook on E4
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhiteRook, NoPiece)
	
	moves := generateRookMoves(pos, E4)
	
	// Rook on E4 should have 14 possible moves (4 directions, varying lengths)
	expected := 14
	if len(moves) != expected {
		t.Errorf("Rook on E4 should have %d moves, got %d", expected, len(moves))
	}
}

func TestQueenMoves(t *testing.T) {
	// Create position with single queen on E4
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhiteQueen, NoPiece)
	
	moves := generateQueenMoves(pos, E4)
	
	// Queen on E4 should have 27 possible moves (combines bishop + rook moves)
	expected := 27
	if len(moves) != expected {
		t.Errorf("Queen on E4 should have %d moves, got %d", expected, len(moves))
	}
}

func TestKingMoves(t *testing.T) {
	// Create position with single king on E4
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhiteKing, NoPiece)
	
	moves := generateKingMoves(pos, WhiteKing)
	
	// King on E4 should have 8 possible moves
	expected := 8
	if len(moves) != expected {
		t.Errorf("King on E4 should have %d moves, got %d", expected, len(moves))
	}
}

func TestCastlingMoves(t *testing.T) {
	// Test white castling
	pos := &Position{
		Board: Bitboard{},
		Tag:   WhiteToMove | WhiteCanCastleKingSide | WhiteCanCastleQueenSide,
	}
	// Set up for castling: King on E1, Rooks on A1 and H1
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(A1, WhiteRook, NoPiece)
	pos.Board.UpdateSquare(H1, WhiteRook, NoPiece)
	
	moves := generateKingMoves(pos, WhiteKing)
	
	// Should include castling moves
	castlingMoves := 0
	for _, move := range moves {
		if move.IsCastle() {
			castlingMoves++
		}
	}
	
	if castlingMoves != 2 {
		t.Errorf("Expected 2 castling moves, got %d", castlingMoves)
	}
}

func TestMoveWithCaptures(t *testing.T) {
	// Create position with white pawn that can capture
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E4, WhitePawn, NoPiece)
	pos.Board.UpdateSquare(D5, BlackPawn, NoPiece)
	pos.Board.UpdateSquare(F5, BlackKnight, NoPiece)
	
	moves := generateWhitePawnMoves(pos)
	
	// Should have: 1 push forward + 2 captures
	expected := 3
	if len(moves) != expected {
		t.Errorf("Expected %d pawn moves, got %d", expected, len(moves))
	}
	
	// Count captures
	captures := 0
	for _, move := range moves {
		if move.IsCapture() {
			captures++
		}
	}
	
	if captures != 2 {
		t.Errorf("Expected 2 captures, got %d", captures)
	}
}

func TestEnPassant(t *testing.T) {
	// Create position with en passant possibility
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: D6, // Black pawn just moved D7-D5
	}
	pos.Board.UpdateSquare(C5, WhitePawn, NoPiece)
	pos.Board.UpdateSquare(E5, WhitePawn, NoPiece)
	pos.Board.UpdateSquare(D5, BlackPawn, NoPiece)
	
	moves := generateWhitePawnMoves(pos)
	
	// Count en passant moves
	enPassantMoves := 0
	for _, move := range moves {
		if move.IsEnPassant() {
			enPassantMoves++
		}
	}
	
	// Should have 2 en passant captures (from C5 and E5)
	if enPassantMoves != 2 {
		t.Errorf("Expected 2 en passant moves, got %d", enPassantMoves)
	}
}

func TestPromotionMoves(t *testing.T) {
	// Create position with pawn about to promote
	pos := &Position{
		Board:     Bitboard{},
		Tag:       WhiteToMove,
		EnPassant: NoSquare,
	}
	pos.Board.UpdateSquare(E7, WhitePawn, NoPiece)
	
	moves := generateWhitePawnMoves(pos)
	
	// Should generate 4 promotion moves (Q, R, B, N)
	expected := 4
	if len(moves) != expected {
		t.Errorf("Expected %d promotion moves, got %d", expected, len(moves))
	}
	
	// Check all promotion types are present
	promotionTypes := make(map[PieceType]bool)
	for _, move := range moves {
		promotionTypes[move.PromoType()] = true
	}
	
	expectedTypes := []PieceType{Queen, Rook, Bishop, Knight}
	for _, pieceType := range expectedTypes {
		if !promotionTypes[pieceType] {
			t.Errorf("Missing promotion to %v", pieceType)
		}
	}
}

func TestPerftStartingPosition(t *testing.T) {
	// Test perft for starting position
	pos := &Position{
		Board:     StartingBoard(),
		Tag:       WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant: NoSquare,
	}
	
	// Test depth 1 - should be 20 moves
	result := RunPerftTest(pos, 1)
	expected := uint64(20)
	if result.Nodes != expected {
		t.Errorf("Perft depth 1: expected %d nodes, got %d", expected, result.Nodes)
	}
	
	// Test depth 2 - should be 400 moves  
	result = RunPerftTest(pos, 2)
	expected = uint64(400)
	if result.Nodes != expected {
		t.Errorf("Perft depth 2: expected %d nodes, got %d", expected, result.Nodes)
	}
}

func BenchmarkMoveGeneration(b *testing.B) {
	pos := &Position{
		Board:     StartingBoard(),
		Tag:       WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant: NoSquare,
	}
	
	BenchmarkFunction(b, func() {
		GenerateMoves(pos)
	})
}

func BenchmarkPerftDepth3(b *testing.B) {
	pos := &Position{
		Board:     StartingBoard(),
		Tag:       WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant: NoSquare,
	}
	
	BenchmarkFunction(b, func() {
		RunPerftTest(pos, 3)
	})
}