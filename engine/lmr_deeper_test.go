package engine

import "testing"

func TestLMRDeeperDepth(t *testing.T) {
	old := T1104_LMR_DEEPER
	t.Cleanup(func() { T1104_LMR_DEEPER = old })
	tests := []struct {
		name                                                      string
		enabled, next, depth, ply, root, score, best, alpha, want int
	}{
		{"off", 0, 7, 8, 3, 8, 100, 0, 0, 7},
		{"strong", 1, 7, 8, 3, 8, 100, 0, 0, 8},
		{"threshold", 1, 7, 8, 3, 8, 36, 0, 0, 7},
		{"marginal", 1, 7, 8, 3, 8, 11, 0, 10, 6},
		{"budget", 1, 7, 8, 29, 8, 100, 0, 0, 7},
		{"node cap", 1, 9, 8, 3, 8, 100, 0, 0, 9},
		{"floor", 1, 0, 8, 3, 8, 11, 0, 10, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			T1104_LMR_DEEPER = tc.enabled
			if got := lmrVerificationDepth(tc.next, tc.depth, tc.ply, tc.root, tc.score, tc.best, tc.alpha); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// These fixed searches pin the actual verification call, not just the depth
// selection helper. Each vector changes if its corresponding decision is
// removed: marginal, strong, or strong with a spent extension budget.
func TestLMRDeeperSearch(t *testing.T) {
	oldOption, oldBudget := T1104_LMR_DEEPER, EXTENSION_BUDGET
	t.Cleanup(func() { T1104_LMR_DEEPER, EXTENSION_BUDGET = oldOption, oldBudget })
	cases := []struct {
		name, fen     string
		depth, budget int
		nodes         uint64
	}{
		{"marginal", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 8, 29, 4739},
		{"strong", "r1bqr1k1/ppp2ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQ1RK1 w - - 0 8", 8, 29, 21387},
		{"spent budget", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 10, 0, 101447},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			T1104_LMR_DEEPER, EXTENSION_BUDGET = 1, tc.budget
			defaultSearchEngine, _ = NewSearchEngineWithHash(DEFAULT_CACHE_SIZE)
			pos, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			info := SearchIterativeDeepening(pos, tc.depth, nil)
			if info.LMRReSearches == 0 || info.Nodes != tc.nodes {
				t.Fatalf("nodes=%d want=%d LMR researches=%d", info.Nodes, tc.nodes, info.LMRReSearches)
			}
		})
	}
}
