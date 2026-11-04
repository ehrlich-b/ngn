package engine

import "testing"

func TestCorrMoreLearningAndStandPat(t *testing.T) {
	old := T1104CorrMore
	T1104CorrMore = 1
	defer func() { T1104CorrMore = old }()
	pos, err := ParseFEN("4k3/pp6/8/8/8/8/6PP/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	h := new(workerHistory)
	prev := NewMove(1, 18, BlackKnight, NoPiece, NoType, 0)
	h.maybeUpdateMore(pos, prev, 0, 200, 100, 8, NewMove(0, 1, WhiteRook, NoPiece, NoType, 0), false)
	if h.majorCorrectionHistory[White][majorCorrectionIndex(pos)] != 200 || h.continuationCorrectionHistory[White][BlackKnight][18] != 200 {
		t.Fatal("both contexts must learn the same signal")
	}
	if got := h.moreCorrection(pos, prev); got != 18 {
		t.Fatalf("correction = %d", got)
	}
	if got := h.moreCorrection(pos, EmptyMove); got != 9 {
		t.Fatalf("missing predecessor correction = %d", got)
	}
	h.maybeUpdateMore(pos, prev, 0, 200, 100, 8, EmptyMove, true)
	if h.majorCorrectionHistory[White][majorCorrectionIndex(pos)] != 200 {
		t.Fatal("in-check update")
	}
	T1104CorrMore = 0
	if h.moreCorrection(pos, prev) != 0 {
		t.Fatal("disabled correction applied")
	}
	h.ClearHistoryTable()
	T1104CorrMore = 1
	if h.moreCorrection(pos, prev) != 0 {
		t.Fatal("history was not cleared")
	}
}

func TestCorrMoreQsearch(t *testing.T) {
	old := T1104CorrMore
	T1104CorrMore = 1
	defer func() { T1104CorrMore = old }()
	pos, err := ParseFEN("4k3/pp6/8/8/8/8/6PP/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	ClearStop()
	defaultSearchEngine, _ = NewSearchEngineWithHash(DEFAULT_CACHE_SIZE)
	h := &defaultSearchEngine.worker.history
	prev := NewMove(1, 18, BlackKnight, NoPiece, NoType, 0)
	h.majorCorrectionHistory[White][majorCorrectionIndex(pos)] = 512
	h.continuationCorrectionHistory[White][BlackKnight][18] = 512
	info := newQSearchInfo()
	info.MoveStack[0] = prev
	raw := EvaluateForPlayer(&pos.Board, White)
	if got := quiescence(pos, -INFINITY, INFINITY, 1, info); got != raw+48 {
		t.Fatalf("qsearch = %d, want %d", got, raw+48)
	}
}
