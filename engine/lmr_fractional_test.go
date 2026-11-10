package engine

import "testing"

func TestLMRFractionalArithmetic(t *testing.T) {
	old := lmrFractionalTable[12][20]
	defer func() { lmrFractionalTable[12][20] = old }()
	lmrFractionalTable[12][20] = 3*lmrPly + 600
	// Rounding the base first would give four, then the old history term
	// would be zero. Keeping both fractional contributions gives three.
	if got := quietLMRFractionalReduction(12, 20, true, true, 1000, false, false, false); got != 3 {
		t.Fatalf("round only after history: got %d want 3", got)
	}
	for _, tc := range []struct {
		name                           string
		pv, improving, tt, killer, cut bool
		score, want                    int
	}{
		{"base", true, true, false, false, false, 0, 4},
		{"non PV", false, true, false, false, false, 0, 5},
		{"not improving", true, false, false, false, false, 0, 5},
		{"cut", true, true, false, false, true, 0, 5},
		{"TT PV", true, true, true, false, false, 0, 3},
		{"killer", true, true, false, true, false, 0, 3},
		{"positive cap", true, true, false, false, false, 100000, 2},
		{"negative cap", true, true, false, false, false, -100000, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := quietLMRFractionalReduction(12, 20, tc.pv, tc.improving, tc.score, tc.tt, tc.killer, tc.cut); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestLMRFractionalCombinedHistory(t *testing.T) {
	var h workerHistory
	move := h1TestQuiet(E2, E4, WhitePawn)
	prev := h1TestQuiet(E7, E5, BlackPawn)
	prev2 := h1TestQuiet(G1, F3, WhiteKnight)
	h.historyTable[WhitePawn][E4] = 500
	h.continuationHistory[BlackPawn][E5][WhitePawn][E4] = 500
	h.followupHistory[WhiteKnight][F3][WhitePawn][E4] = 500
	for _, tc := range []struct {
		prev, prev2 Move
		want        int
	}{
		{EmptyMove, EmptyMove, 500}, {prev, EmptyMove, 1000},
		{EmptyMove, prev2, 1000}, {prev, prev2, 1500},
	} {
		score := h.GetHistoryScore(move, tc.prev, tc.prev2)
		if score != tc.want {
			t.Fatalf("combined score %d want %d", score, tc.want)
		}
		if r := quietLMRFractionalReduction(12, 20, true, true, score, false, false, false); r != 3 && tc.want >= 1000 {
			t.Fatalf("combined history not applied: score=%d reduction=%d", score, r)
		}
	}
	// Reversing only the follow-up entry crosses the final rounding boundary.
	before := quietLMRFractionalReduction(12, 20, true, true, h.GetHistoryScore(move, prev, prev2), false, false, false)
	h.followupHistory[WhiteKnight][F3][WhitePawn][E4] = -500
	after := quietLMRFractionalReduction(12, 20, true, true, h.GetHistoryScore(move, prev, prev2), false, false, false)
	if before != 3 || after != 4 {
		t.Fatalf("follow-up influence: before=%d after=%d", before, after)
	}
}

func TestLMRFractionalBoundsAndTunables(t *testing.T) {
	old := lmrFractionalTable[20][20]
	defer func() { setTunableParam("LMRBase", "0") }()
	if !setTunableParam("LMRBase", "1") || lmrFractionalTable[20][20] != old+lmrPly {
		t.Fatal("fractional table did not rebuild with the base option")
	}
	setTunableParam("LMRBase", "0")
	for depth := 3; depth <= 70; depth++ {
		for _, score := range []int{-100000, 0, 100000} {
			for _, cut := range []bool{false, true} {
				r := quietLMRFractionalReduction(depth, 70, false, false, score, true, true, cut)
				if r < 0 || r > depth-2 {
					t.Fatalf("depth=%d score=%d reduction=%d", depth, score, r)
				}
			}
		}
	}
}

func TestLMRFractionalSearch(t *testing.T) {
	old := T1104_LMR_FRACTIONAL
	defer func() { T1104_LMR_FRACTIONAL = old }()
	found := false
	for _, p := range TunableSearchParams {
		if p.Name == "T1104LMRFractional" {
			found = p.Ptr == &T1104_LMR_FRACTIONAL && p.Def == acceptedOptionDefaults["T1104LMRFractional"] && p.Min == 0 && p.Max == 1
		}
	}
	if !found || !setTunableParam("T1104LMRFractional", "1") || T1104_LMR_FRACTIONAL != 1 {
		t.Fatal("spin option with accepted default is not wired")
	}
	const fen = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	search := func() *SearchInfo {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		e, err := NewSearchEngineWithHash(4)
		if err != nil {
			t.Fatal(err)
		}
		return e.SearchIterativeDeepening(pos, 5, nil)
	}
	on := search()
	setTunableParam("T1104LMRFractional", "0")
	off := search()
	if on.LMRReductions == 0 || on.BestMove == EmptyMove || on.Nodes == off.Nodes {
		t.Fatalf("on path not exercised: on nodes=%d reductions=%d; off nodes=%d", on.Nodes, on.LMRReductions, off.Nodes)
	}
}
