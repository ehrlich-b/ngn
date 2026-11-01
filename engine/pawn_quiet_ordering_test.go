package engine

import "testing"

// Uses only the baseline API: without a pawn-context producer/consumer, both
// positions necessarily receive the same quiet ordering score after learning.
func TestPawnQuietOrderingConsumesCompletedRoot(t *testing.T) {
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	pos, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	other, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/P7/1PPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	searcher.NewGame()
	info := searcher.Search(pos, 1)
	if info.Stopped || info.BestMove == EmptyMove {
		t.Fatal("root did not complete")
	}
	moves := []Move{info.BestMove}
	var out [1]Move
	var trained, different [1]int
	h := searcher.worker.history
	h.scoreMovesIntoBuffer(moves, 0, EmptyMove, EmptyMove, EmptyMove, false, out[:], trained[:], pos, false)
	h.scoreMovesIntoBuffer(moves, 0, EmptyMove, EmptyMove, EmptyMove, false, out[:], different[:], other, false)
	if trained[0] <= different[0] {
		t.Fatalf("completed root did not train its pawn context: trained=%d other=%d", trained[0], different[0])
	}
}
