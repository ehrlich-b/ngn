package engine

import "testing"

func newQSearchInfo() *SearchInfo {
	var moveBuffer [256]Move
	var captureBuffer [64]Move
	var orderedBuffer [256]Move
	var seeGains [32]int
	return &SearchInfo{
		MoveBuffer:    &moveBuffer,
		CaptureBuffer: &captureBuffer,
		OrderedBuffer: &orderedBuffer,
		SEEGains:      &seeGains,
	}
}

func qsearchOf(t *testing.T, fen string) (got, standPat int) {
	t.Helper()
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("bad FEN %q: %v", fen, err)
	}
	ClearStop()
	TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
	standPat = EvaluateForPlayer(&pos.Board, pos.Turn())
	got = quiescence(pos, -INFINITY, INFINITY, 0, newQSearchInfo())
	return
}

// TestQuiescenceInvariants checks core qsearch correctness on crafted positions.
// qsearch produces every leaf evaluation in the search, so a bug here corrupts
// everything above it.
func TestQuiescenceInvariants(t *testing.T) {
	// A) Quiet positions (no captures, not in check): qsearch must return EXACTLY
	// the stand-pat eval.
	quiet := []string{
		"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
		"4k3/pp6/8/8/8/8/6PP/4K3 w - - 0 1",
		"r3k3/8/8/8/8/8/8/4K2R w Kq - 0 1",
	}
	for _, fen := range quiet {
		got, standPat := qsearchOf(t, fen)
		if got != standPat {
			t.Errorf("quiet qsearch(%s)=%d, want stand-pat %d", fen, got, standPat)
		}
	}

	// B) A free hanging queen (pawn attacks an undefended queen): qsearch must
	// realize the win, i.e. score far above stand-pat. If delta/SEE pruning wrongly
	// skipped the capture, qsearch would collapse to stand-pat and fail here.
	got, standPat := qsearchOf(t, "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1")
	if got-standPat < 800 {
		t.Errorf("hanging-queen qsearch=%d, stand-pat=%d (gain %d); expected to win ~a queen",
			got, standPat, got-standPat)
	}

	// C) Only a losing capture available (QxP defended by a pawn, SEE -900):
	// qsearch must DECLINE it and return stand-pat, not blunder the queen.
	got, standPat = qsearchOf(t, "4k3/8/2p5/3p4/8/8/8/3QK3 w - - 0 1")
	if got != standPat {
		t.Errorf("losing-capture qsearch=%d, want stand-pat %d (should decline QxP)", got, standPat)
	}
}

func TestQuiescenceDepthCapStillHandlesCheckmate(t *testing.T) {
	pos, err := ParseFEN("7k/6Q1/6K1/8/8/8/8/8 b - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	if !pos.IsInCheck() {
		t.Fatal("test position must have side to move in check")
	}
	ClearStop()
	TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)

	const ply = 6
	got := quiescenceWithDepth(pos, -INFINITY, INFINITY, ply, newQSearchInfo(), 6)
	want := -MATE_VALUE + ply
	if got != want {
		t.Fatalf("qsearch at cap while in check returned %d, want mate score %d", got, want)
	}
}
