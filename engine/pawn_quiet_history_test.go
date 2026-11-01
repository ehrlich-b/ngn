package engine

import "testing"

func pawnQuietFixture(t *testing.T) (*Position, []Move) {
	t.Helper()
	pos, err := ParseFEN("4k3/4p3/8/8/8/8/4P3/1N2K1N1 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	return pos, []Move{
		NewMove(B1, C3, WhiteKnight, NoPiece, NoType, 0),
		NewMove(B1, A3, WhiteKnight, NoPiece, NoType, 0),
		NewMove(G1, F3, WhiteKnight, NoPiece, NoType, 0),
	}
}

func TestPawnQuietCompletedExamples(t *testing.T) {
	pos, moves := pawnQuietFixture(t)
	h := &workerHistory{}
	h.updatePawnQuiets(pos, moves, moves[1], 8, 0, 10)
	row := &h.pawnQuietHistory[pawnQuietIndex(pos)][WhiteKnight]
	if row[C3] >= 0 || row[A3] <= 0 || row[F3] != 0 {
		t.Fatalf("only winner and prior completed quiet should learn: %v", row)
	}
	if h.GetHistoryScore(moves[1], EmptyMove, EmptyMove) != 0 {
		t.Fatal("pawn ordering history leaked into pruning/LMR history")
	}
	for i := 0; i < 1000; i++ {
		h.updatePawnQuiets(pos, moves, moves[1], 100, 0, 10)
	}
	if row[A3] > historyMax || row[C3] < -historyMax {
		t.Fatal("history escaped its bounds")
	}
}

func TestPawnQuietStoppedRootDoesNotLearn(t *testing.T) {
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	searcher.NewGame()
	searcher.SetMaxNodes(1)
	info := searcher.Search(controlTestPosition(t), 1)
	if !info.Stopped {
		t.Fatal("node-limited root did not stop")
	}
	if searcher.worker.history.pawnQuietHistory != [pawnQuietBuckets][13][64]int16{} {
		t.Fatal("interrupted root trained pawn history")
	}
}

func TestPawnQuietNoUnprovedSignal(t *testing.T) {
	pos, moves := pawnQuietFixture(t)
	h := &workerHistory{}
	h.updatePawnQuiets(pos, moves, moves[1], 8, 10, 10)
	h.updatePawnQuiets(pos, moves[:1], moves[1], 8, 0, 10)
	for _, special := range []Move{
		NewMove(E2, D3, WhitePawn, BlackKnight, NoType, Capture),
		NewMove(E5, D6, WhitePawn, NoPiece, NoType, EnPassant),
		NewMove(E7, E8, WhitePawn, NoPiece, Queen, 0),
		NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle),
	} {
		h.updatePawnQuiets(pos, append(moves[:1:1], special), special, 8, 0, 10)
	}
	if h.pawnQuietHistory != [pawnQuietBuckets][13][64]int16{} {
		t.Fatal("fail-low, unsearched winner, or special move trained pawn history")
	}
}

func TestPawnQuietContextOwnershipAndReset(t *testing.T) {
	pos, moves := pawnQuietFixture(t)
	other, err := ParseFEN("4k3/4p3/8/8/8/4P3/8/1N2K1N1 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	if pawnQuietIndex(pos) == pawnQuietIndex(other) {
		t.Fatal("fixture contexts collide")
	}
	first, second := &workerHistory{}, &workerHistory{}
	first.updatePawnQuiets(pos, moves, moves[1], 8, 0, 10)
	if first.pawnQuietHistory[pawnQuietIndex(other)][WhiteKnight][A3] != 0 || second.pawnQuietHistory != [pawnQuietBuckets][13][64]int16{} {
		t.Fatal("context or worker ownership leaked")
	}
	first.ClearHistoryTable()
	if first.pawnQuietHistory != [pawnQuietBuckets][13][64]int16{} {
		t.Fatal("new game did not clear pawn history")
	}
}
