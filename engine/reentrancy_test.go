package engine

import "testing"

// TestIIDIsWired guards the IID re-entrancy fix. IID re-enters alphaBetaPV on
// the SAME position while the node's own hash is on info.RepStack; before the
// fix the re-entry matched its own hash in the search-path repetition scan and
// returned 0 immediately, making IID a complete no-op (TT re-probe found
// nothing). A no-op feature produces IDENTICAL node counts whether toggled on or
// off; once wired, the counts differ. (See SearchToggles.IID.)
func TestIIDIsWired(t *testing.T) {
	// PV-heavy middlegame so IID (PV node, no TT move, depth>=4) fires.
	fen := "r1bqkb1r/pp1n1ppp/2p1pn2/3p4/2PP4/2N1PN2/PP3PPP/R1BQKB1R w KQkq - 0 6"

	nodes := func(iid bool) uint64 {
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
		return SearchIterativeDeepening(pos, 9, nil).Nodes
	}

	on := nodes(true)
	off := nodes(false)
	t.Logf("nodes: IID-on=%d  IID-off=%d", on, off)
	if on == off {
		t.Errorf("IID on vs off produced identical node counts (%d) — IID is a no-op (re-entrancy regressed)", on)
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
