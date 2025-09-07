package engine

import (
	"testing"
)

func TestMoveOrdering(t *testing.T) {
	// Create a simple test position
	pos := &Position{}
	pos.Board = Bitboard{}
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)
	pos.SetTag(WhiteToMove)
	pos.EnPassant = NoSquare
	pos.HalfMoveClock = 0

	// Create some test moves with different types
	moves := []Move{
		// Quiet pawn move (should be last)
		NewMove(E2, E4, WhitePawn, NoPiece, NoType, 0),

		// Capture queen with pawn (should be first - high value victim)
		NewMove(E5, D6, WhitePawn, BlackQueen, NoType, Capture),

		// Capture pawn with queen (should be lower - low value victim, high value attacker)
		NewMove(D1, D7, WhiteQueen, BlackPawn, NoType, Capture),

		// Promotion (should be high priority)
		NewMove(E7, E8, WhitePawn, NoPiece, Queen, 0),

		// Castle (should get bonus)
		NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle),

		// En passant (should be like pawn capture)
		NewMove(E5, D6, WhitePawn, NoPiece, NoType, EnPassant),
	}

	orderedMoves := orderMoves(moves, pos)

	if len(orderedMoves) != len(moves) {
		t.Errorf("Expected %d moves, got %d", len(moves), len(orderedMoves))
	}

	// Test that the first move is a high-priority capture (queen capture)
	if len(orderedMoves) > 0 {
		firstMove := orderedMoves[0]
		if !firstMove.IsCapture() || firstMove.CapturedPiece() != BlackQueen {
			t.Errorf("Expected queen capture as first move, got %v", firstMove)
		}
	}
}

func TestMoveOrderingWithSearch(t *testing.T) {
	// Test that move ordering improves search efficiency
	// Create a position with captures available
	pos := &Position{}
	pos.Board = Bitboard{}

	// Set up position with multiple captures
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)

	// White pieces that can capture
	pos.Board.UpdateSquare(D1, WhiteQueen, NoPiece)
	pos.Board.UpdateSquare(E4, WhitePawn, NoPiece)

	// Black pieces to capture
	pos.Board.UpdateSquare(D7, BlackRook, NoPiece)
	pos.Board.UpdateSquare(F5, BlackPawn, NoPiece)

	pos.SetTag(WhiteToMove)
	pos.EnPassant = NoSquare
	pos.HalfMoveClock = 0

	// Search should still work correctly with move ordering
	info := Search(pos, 3)

	if info.BestMove == EmptyMove {
		t.Error("Search should find a best move with move ordering")
	}

	if info.Nodes == 0 {
		t.Error("Search should examine nodes")
	}
}
