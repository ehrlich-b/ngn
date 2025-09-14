package engine

import (
	"testing"
)

func TestHasLegalMoveDebug(t *testing.T) {
	// Start with position before Ra8
	startPos, _ := ParseFEN("7k/R7/6KP/8/8/8/8/8 w - - 0 1")
	ra8Move, _ := ParseAlgebraicMove("a7a8", startPos)
	startPos.MakeMove(ra8Move)

	// Now we have the checkmate position after Ra8
	pos := startPos

	t.Logf("Testing checkmate position: %s", GenerateFEN(pos))
	t.Logf("Black to move, in check: %v", pos.IsInCheck())

	// Test HasLegalMove
	hasLegal := pos.HasLegalMove()
	t.Logf("HasLegalMove returns: %v", hasLegal)

	// Manually test each move
	var moveBuffer [256]Move
	moveCount := GenerateMovesIntoBuffer(pos, moveBuffer[:])
	t.Logf("Generated %d pseudo-legal moves", moveCount)

	legalCount := 0
	for i := 0; i < moveCount; i++ {
		move := moveBuffer[i]
		oldEP, oldTag, oldHalfClock, legal := pos.MakeMove(move)
		if legal {
			moveStr := MoveToAlgebraic(move)
			t.Logf("  Move %s marked as legal", moveStr)

			// Check if king is actually safe after this move
			kingInCheck := pos.IsInCheck()
			pos.UnMakeMove(move, oldTag, oldEP, oldHalfClock)

			if kingInCheck {
				t.Errorf("    ERROR: Move %s leaves king in check!", moveStr)
			}
			legalCount++
		}
	}

	t.Logf("Total legal moves: %d", legalCount)

	if legalCount == 0 && hasLegal {
		t.Errorf("BUG: HasLegalMove returned true but there are no legal moves!")
	}
	if legalCount > 0 && !hasLegal {
		t.Errorf("BUG: HasLegalMove returned false but there are %d legal moves!", legalCount)
	}
}