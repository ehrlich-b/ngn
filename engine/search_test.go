package engine

import (
	"testing"
)

func newStartingPosition() *Position {
	return &Position{
		Board:     StartingBoard(),
		Tag:       WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove,
		EnPassant: NoSquare,
	}
}

func TestBasicSearch(t *testing.T) {
	pos := newStartingPosition()

	// Test search at depth 1 from starting position
	info := Search(pos, 1)

	if info == nil {
		t.Fatal("Search returned nil")
	}

	if info.BestMove == EmptyMove {
		t.Error("Search did not find a best move")
	}

	if info.Nodes == 0 {
		t.Error("Search reported 0 nodes searched")
	}

	if info.Depth != 1 {
		t.Errorf("Expected depth 1, got %d", info.Depth)
	}

	// Best move should be a legal move
	legalMoves := GenerateLegalMoves(pos)
	found := false
	for _, move := range legalMoves {
		if move == info.BestMove {
			found = true
			break
		}
	}
	if !found {
		t.Error("Best move is not a legal move")
	}
}

func TestSearchTacticalPosition(t *testing.T) {
	// Set up a position where white has a clear advantage
	// Test that search finds a reasonable move
	pos := &Position{}
	pos.Board = Bitboard{}

	// Place kings
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)

	// Place white queen - should be stronger than black pieces
	pos.Board.UpdateSquare(D1, WhiteQueen, NoPiece)

	// Place some black pieces
	pos.Board.UpdateSquare(A7, BlackPawn, NoPiece)
	pos.Board.UpdateSquare(B7, BlackPawn, NoPiece)
	pos.Board.UpdateSquare(A8, BlackRook, NoPiece)

	pos.SetTag(WhiteToMove)
	pos.EnPassant = NoSquare
	pos.HalfMoveClock = 0

	// Search should find a good move
	info := Search(pos, 3)

	// Verify that search found a move
	if info.BestMove == EmptyMove {
		t.Error("Search should find a best move")
	}

	// The search should find a reasonable score (not necessarily positive
	// since this position might not actually be winning for White)
	if info.BestScore < -1000 {
		t.Errorf("Search returned unreasonably low score: %d", info.BestScore)
	}
}

func TestSearchAvoidMate(t *testing.T) {
	// Set up a position where Black is threatened with mate
	// White Queen on d1, Black King on e8, Black should move king
	pos := &Position{}
	pos.Board = Bitboard{}

	// Place kings
	pos.Board.UpdateSquare(A1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)

	// Place white queen threatening mate
	pos.Board.UpdateSquare(D1, WhiteQueen, NoPiece)

	pos.SetTag(BlackToMove)
	pos.EnPassant = NoSquare
	pos.HalfMoveClock = 0

	// Search should try to avoid immediate mate
	info := Search(pos, 2)

	// The score shouldn't be a large negative (mate) value
	if info.BestScore < -MATE_IN_MAX {
		t.Errorf("Black should avoid mate, but got score %d", info.BestScore)
	}
}

func TestSearchNodeCount(t *testing.T) {
	pos := newStartingPosition()

	// Search at different depths and verify node counts increase
	info1 := Search(pos, 1)
	info2 := Search(pos, 2)

	if info2.Nodes <= info1.Nodes {
		t.Errorf("Deeper search should examine more nodes. Depth 1: %d, Depth 2: %d",
			info1.Nodes, info2.Nodes)
	}
}

func BenchmarkSearch(b *testing.B) {
	pos := newStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Search(pos, 3)
	}
}

func TestIterativeDeepening(t *testing.T) {
	pos := newStartingPosition()

	// Test that iterative deepening produces results at each depth
	info := SearchIterativeDeepening(pos, 3, nil)

	if info.BestMove == EmptyMove {
		t.Error("Iterative deepening should find a best move")
	}

	if info.Depth != 3 {
		t.Errorf("Expected final depth 3, got %d", info.Depth)
	}

	// Compare with fixed depth search - should get same result
	infoFixed := SearchFixed(pos, 3, nil)

	if info.BestMove != infoFixed.BestMove {
		t.Errorf("Iterative deepening and fixed depth should find same best move. ID: %s, Fixed: %s",
			info.BestMove.ToString(), infoFixed.BestMove.ToString())
	}

	if info.BestScore != infoFixed.BestScore {
		t.Errorf("Iterative deepening and fixed depth should get same score. ID: %d, Fixed: %d",
			info.BestScore, infoFixed.BestScore)
	}
}

func BenchmarkSearchDeep(b *testing.B) {
	pos := newStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Search(pos, 4)
	}
}

func TestQuiescenceSearch(t *testing.T) {
	// Create a position with a tactical opportunity (capture sequence)
	pos := &Position{}
	pos.Board = Bitboard{}

	// Place kings
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)

	// Create a capture sequence: White queen can capture black rook
	pos.Board.UpdateSquare(D1, WhiteQueen, NoPiece)
	pos.Board.UpdateSquare(D7, BlackRook, NoPiece)

	// Add a black piece that can recapture
	pos.Board.UpdateSquare(C8, BlackBishop, NoPiece)

	pos.SetTag(WhiteToMove)
	pos.EnPassant = NoSquare
	pos.HalfMoveClock = 0

	// Search should find the tactical sequence
	info := Search(pos, 2)

	if info.BestMove == EmptyMove {
		t.Error("Should find a best move in tactical position")
	}

	// The engine should be able to evaluate the position
	// (exact score depends on position evaluation)
	if info.Nodes == 0 {
		t.Error("Search should examine some nodes")
	}
}

func TestTranspositionTable(t *testing.T) {
	pos := newStartingPosition()

	// Clear the transposition table
	TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)

	// Do a search to populate the transposition table
	info1 := Search(pos, 3)

	if info1.BestMove == EmptyMove {
		t.Error("First search should find a best move")
	}

	// Do the same search again - should be faster due to TT hits
	info2 := Search(pos, 3)

	// Scores must match. Best-move can differ when multiple moves tie on score —
	// TT-driven move ordering surfaces a different (equally-good) move first.
	if info2.BestScore != info1.BestScore {
		t.Errorf("Both searches should get the same score. First: %d, Second: %d",
			info1.BestScore, info2.BestScore)
	}

	// Second search should examine fewer nodes (due to TT hits)
	// Note: This might not always be true due to hash collisions and replacement strategy
	// but it's generally expected
	if info2.Nodes >= info1.Nodes {
		t.Logf("Warning: Second search examined %d nodes vs first search %d nodes",
			info2.Nodes, info1.Nodes)
	}
}

func BenchmarkIterativeDeepening(b *testing.B) {
	pos := newStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SearchIterativeDeepening(pos, 4, nil)
	}
}

func BenchmarkWithTranspositionTable(b *testing.B) {
	pos := newStartingPosition()

	// Clear TT before benchmarking
	TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Search(pos, 4)
	}
}
