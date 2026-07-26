package engine

import "testing"

// TestIIDIsWired guards the IID re-entrancy fix. IID re-enters alphaBetaPV on
// the SAME position while the node's own hash is on info.RepStack; before the
// fix the re-entry matched its own hash in the search-path repetition scan and
// returned 0 immediately, making IID a complete no-op (TT re-probe found
// nothing). A no-op feature produces IDENTICAL node counts whether toggled on or
// off; once wired, the counts differ. (See SearchToggles.IID.)
func TestIIDIsWired(t *testing.T) {
	// IID (PV node, no TT move, depth>=4) only fires in a particular regime, and a
	// search-behavior change can shift any single position out of it — the T4c minor
	// correction-history term did exactly that to the prior r1bqkb1r/... FEN, and the
	// T19 ttPv candidate did it again to the r2q1rk1/... FEN at depth 9. When that
	// happens IID-on == IID-off trivially, so the no-op guard would pass spuriously.
	//
	// Rather than pin one position and re-pick it every time the tree moves, probe a
	// SET of PV-heavy middlegames across two depths and guard on the first that
	// actually fires. This is strictly stronger than the single-FEN form: defeating
	// the guard now requires EVERY probe to drift out of regime at once, instead of
	// just one. The re-entrancy bug's signature is unchanged — IID RUNS but changes
	// nothing — so the node-count comparison stays the primary assertion.
	fens := []string{
		"r2q1rk1/1b1nbppp/p2ppn2/1p6/3NPP2/1BN1B3/PPPQ2PP/2KR3R w - - 0 12",
		"r1bq1rk1/pp2ppbp/2np1np1/8/2PNP3/2N1B3/PP2BPPP/R2QK2R w KQ - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"2rq1rk1/pp1bppbp/3p1np1/8/2PNP3/1PN1B3/P3BPPP/R2Q1RK1 w - - 0 1",
	}
	depths := []int{9, 10}

	search := func(fen string, depth int, iid bool) *SearchInfo {
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
		return SearchIterativeDeepening(pos, depth, nil)
	}

	var on, off *SearchInfo
	var usedFen string
	var usedDepth int
	for _, depth := range depths {
		for _, fen := range fens {
			if probe := search(fen, depth, true); probe.IIDSearches > 0 {
				on, usedFen, usedDepth = probe, fen, depth
				break
			}
		}
		if on != nil {
			break
		}
	}
	if on == nil {
		t.Fatalf("IID never fired on ANY of the %d probe positions at depths %v — either IID regressed to dead code, or every probe drifted out of the firing regime at once; cannot guard the no-op regression", len(fens), depths)
	}
	off = search(usedFen, usedDepth, false)
	t.Logf("fen=%q depth=%d  nodes: IID-on=%d(fires=%d)  IID-off=%d(fires=%d)", usedFen, usedDepth, on.Nodes, on.IIDSearches, off.Nodes, off.IIDSearches)
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
