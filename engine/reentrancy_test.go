package engine

import "testing"

// TestIIDIsWired guards the IID re-entrancy fix. IID re-enters alphaBetaPV on
// the SAME position while the node's own hash is on info.RepStack; before the
// fix the re-entry matched its own hash in the search-path repetition scan and
// returned 0 immediately, making IID a complete no-op (TT re-probe found
// nothing). A no-op feature produces IDENTICAL node counts whether toggled on or
// off; once wired, the counts differ. (See SearchToggles.IID.)
func TestIIDIsWired(t *testing.T) {
	// PV-heavy middlegame so IID (PV node, no TT move, depth>=4) fires. The FEN must
	// actually exercise IID: an eval-value change can shift a position out of the
	// IID-firing regime (the T4c minor correction-history term moved the prior
	// r1bqkb1r/... FEN there — it stopped firing IID, so on==off spuriously), so the
	// test asserts IID fires (on.IIDSearches>0) before reading the no-op signal. The
	// re-entrancy bug's signature is IID RUNS but changes nothing — identical node
	// counts despite IIDSearches>0 — so the node-count check stays the primary guard.
	fen := "r2q1rk1/1b1nbppp/p2ppn2/1p6/3NPP2/1BN1B3/PPPQ2PP/2KR3R w - - 0 12"

	search := func(iid bool) *SearchInfo {
		SearchToggles.IID = iid
		defer func() { SearchToggles.IID = true }()
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
		ClearHistoryTable()
		ClearKillerMoves()
		ClearCounterMoves()
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		return SearchIterativeDeepening(pos, 9, nil)
	}

	on := search(true)
	off := search(false)
	t.Logf("nodes: IID-on=%d(fires=%d)  IID-off=%d(fires=%d)", on.Nodes, on.IIDSearches, off.Nodes, off.IIDSearches)
	if on.IIDSearches == 0 {
		t.Fatalf("IID never fired on this FEN — it no longer exercises IID (pick a PV-heavy FEN with no early TT move); cannot guard the no-op regression")
	}
	if off.IIDSearches != 0 {
		t.Errorf("IID fired (%d) with the toggle OFF — SearchToggles.IID not wired", off.IIDSearches)
	}
	if on.Nodes == off.Nodes {
		t.Errorf("IID on vs off produced identical node counts (%d) — IID ran but changed nothing (re-entrancy regressed to a bogus-draw no-op)", on.Nodes)
	}
}

// TestSingularExtensionsFire guards the singular fix. Singular extensions never
// worked: the re-entrancy bug made the verification return a bogus draw, and
// even with that fixed the verification took an immediate TT cutoff from the
// same entry that triggered it (so singularScore == ttEval, never singular).
// With both fixed (ply-scoped TT-cutoff/store exclusion + repetition hide) the
// verification runs for real and extensions fire. A regression of either bug
// drops SingularExtensions back to 0.
func TestSingularExtensionsFire(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1" // Kiwipete

	search := func(singular bool) *SearchInfo {
		SearchToggles.Singular = singular
		defer func() { SearchToggles.Singular = true }()
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
		ClearHistoryTable()
		ClearKillerMoves()
		ClearCounterMoves()
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		return SearchIterativeDeepening(pos, 10, nil)
	}

	on := search(true)
	off := search(false)
	t.Logf("SingularExtensions: on=%d off=%d  nodes on=%d off=%d", on.SingularExtensions, off.SingularExtensions, on.Nodes, off.Nodes)
	if on.SingularExtensions == 0 {
		t.Errorf("no singular extensions fired with singular ON — verification is broken (returns bogus draw or self TT cutoff)")
	}
	if off.SingularExtensions != 0 {
		t.Errorf("singular extensions fired (%d) with singular OFF — toggle not wired", off.SingularExtensions)
	}
}
