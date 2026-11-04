package engine

import "testing"

func TestSingularDouble(t *testing.T) {
	old := T1104_SINGULAR_DOUBLE
	defer func() { T1104_SINGULAR_DOUBLE = old }()
	for _, tc := range []struct {
		option, score int
		pv            bool
		want          int
	}{
		{0, -1000, false, 1}, {1, 0, false, 0}, {1, -1, false, 1},
		{1, -32, false, 1}, {1, -33, false, 2}, {1, -96, false, 2},
		{1, -97, false, 3}, {1, -97, true, 2},
	} {
		T1104_SINGULAR_DOUBLE = tc.option
		if got := singularExtension(tc.score, 0, tc.pv); got != tc.want {
			t.Errorf("option=%d score=%d pv=%v: got %d, want %d", tc.option, tc.score, tc.pv, got, tc.want)
		}
	}
}

func TestSingularDoubleSearch(t *testing.T) {
	old := T1104_SINGULAR_DOUBLE
	defer func() { T1104_SINGULAR_DOUBLE = old }()
	search := func(option int) *SearchInfo {
		T1104_SINGULAR_DOUBLE = option
		defaultSearchEngine, _ = NewSearchEngineWithHash(DEFAULT_CACHE_SIZE)
		ClearHistoryTable()
		ClearKillerMoves()
		ClearCounterMoves()
		pos, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
		if err != nil {
			t.Fatal(err)
		}
		return SearchIterativeDeepening(pos, 10, nil)
	}
	off, on := search(0), search(1)
	if on.SingularExtensions == 0 || on.Nodes == off.Nodes {
		t.Fatalf("option did not change singular search: off nodes=%d on nodes=%d singular=%d", off.Nodes, on.Nodes, on.SingularExtensions)
	}
}
