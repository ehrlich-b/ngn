package engine

import "testing"

func TestCuckooOwnKeys(t *testing.T) {
	count := 0
	for _, piece := range Pieces {
		if piece.Type() == Pawn {
			continue
		}
		for from := A1; from <= H8; from++ {
			pos := &Position{Side: piece.Color(), EnPassant: NoSquare}
			pos.Board.UpdateSquare(from, piece, NoPiece)
			for to := from + 1; to <= H8; to++ {
				move := NewMove(from, to, piece, NoPiece, NoType, 0)
				// Use NGN's independent move validator as the geometry oracle.
				if !moveIsPseudoLegal(pos, move) {
					continue
				}
				key := occupantKey(piece, from) ^ occupantKey(piece, to) ^ whiteSideKey
				if got := cuckooMove(key); got != move {
					t.Fatalf("missing reversible move %v (%v): got %v", move, piece, got)
				}
				count++
			}
		}
	}
	stored := 0
	for _, entry := range cuckooTable {
		if entry.move != EmptyMove {
			stored++
		}
	}
	if count != 3668 || stored != count {
		t.Fatalf("generated=%d stored=%d, want 3668", count, stored)
	}
}

func cuckooCycle(t *testing.T) (*Position, SearchInfo) {
	t.Helper()
	pos, err := ParseFEN("6rk/8/8/8/8/8/8/KR6 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	var info SearchInfo
	for _, text := range []string{"b1b2", "g8g7", "b2b1"} {
		info.RepStack[info.RepStackLen] = pos.Hash()
		info.RepStackLen++
		move := adapterLegalMove(t, pos, text)
		pos.MakeMove(move)
	}
	return pos, info
}

func TestCuckooSearchPath(t *testing.T) {
	pos, info := cuckooCycle(t)
	before := snapshotPVPosition(pos)
	if upcomingRepetition(pos, pos.Hash(), &info) == EmptyMove {
		t.Fatal("missed g7g8 closing the search-path cycle")
	}
	assertPVPositionRestored(t, pos, before)
	info.cuckooStart = 1
	if upcomingRepetition(pos, pos.Hash(), &info) != EmptyMove {
		t.Fatal("matched an ancestor before the null-move boundary")
	}
	info.cuckooStart = 0
	pos.HalfMoveClock = 2
	if upcomingRepetition(pos, pos.Hash(), &info) != EmptyMove {
		t.Fatal("matched an ancestor beyond the reversible segment")
	}
	pos.HalfMoveClock = 3
	info.RepStackLen = 2
	if upcomingRepetition(pos, pos.Hash(), &info) != EmptyMove {
		t.Fatal("claimed a cycle with fewer than three ancestor plies")
	}
}

func TestCuckooMoveValidation(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		from, to  Square
		piece     Piece
		want      bool
	}{
		{"forward", "7k/8/8/8/8/8/8/KR6 w - - 3 1", B1, B2, WhiteRook, true},
		{"reverse", "7k/8/8/8/8/8/1R6/K7 w - - 3 1", B2, B1, WhiteRook, true},
		{"black", "6rk/8/8/8/8/8/8/KR6 b - - 3 1", G8, G7, BlackRook, true},
		{"blocked", "7k/8/8/8/8/8/1P6/KR6 w - - 3 1", B1, B4, WhiteRook, false},
		{"occupied", "7k/8/8/8/8/1b6/8/KR6 w - - 3 1", B1, B3, WhiteRook, false},
		{"pinned", "4r2k/8/8/8/8/8/4N3/R3K3 w - - 3 1", E2, C3, WhiteKnight, false},
		{"king safety", "5r1k/8/8/8/8/8/8/R3K3 w - - 3 1", E1, F1, WhiteKing, false},
		{"rook rights", "7k/8/8/8/8/8/8/4K2R w K - 3 1", H1, H2, WhiteRook, false},
		{"king rights", "7k/8/8/8/8/8/8/4K2R w K - 3 1", E1, F1, WhiteKing, false},
		{"wrong side", "6rk/8/8/8/8/8/8/KR6 w - - 3 1", G8, G7, BlackRook, false},
		{"pawn", "7k/8/8/8/8/8/1P6/KR6 w - - 3 1", B2, B3, WhitePawn, false},
		{"legal ep", "7k/8/8/3pP3/8/8/8/KR6 w - d6 3 1", B1, B2, WhiteRook, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			before := snapshotPVPosition(pos)
			var info SearchInfo
			info.RepStackLen = 3
			info.RepStack[0] = pos.Hash() ^ occupantKey(tc.piece, tc.from) ^ occupantKey(tc.piece, tc.to) ^ whiteSideKey
			if got := upcomingRepetition(pos, pos.Hash(), &info) != EmptyMove; got != tc.want {
				t.Fatalf("upcoming repetition=%t, want %t", got, tc.want)
			}
			assertPVPositionRestored(t, pos, before)
		})
	}
}

func TestCuckooOptionAndAlpha(t *testing.T) {
	old := T1104Cuckoo
	t.Cleanup(func() { T1104Cuckoo = old })
	found := false
	for _, param := range TunableSearchParams {
		if param.Name == "T1104Cuckoo" {
			found = param.Ptr == &T1104Cuckoo && param.Def == acceptedOptionDefaults["T1104Cuckoo"] && param.Min == 0 && param.Max == 1
		}
	}
	if !found || !setTunableParam("T1104Cuckoo", "1") || T1104Cuckoo != 1 {
		t.Fatal("missing UCI spin option with accepted default")
	}
	for _, tc := range []struct {
		name             string
		option, ply      int
		alpha, beta      int
		excluded, cutoff bool
	}{
		{"enabled", 1, 3, -100, -1, false, true},
		{"disabled", 0, 3, -100, -1, false, false},
		{"root", 1, 0, -100, -1, false, false},
		{"nonnegative alpha", 1, 3, 0, 1, false, false},
		{"wide window continues", 1, 3, -100, 100, false, false},
		{"singular exclusion", 1, 3, -100, -1, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, path := cuckooCycle(t)
			_, info := newProbcutContractStateForFEN(t, "6rk/8/8/8/8/8/8/KR6 b - - 3 1", 2)
			if err := info.evaluator.Reset(pos); err != nil {
				t.Fatal(err)
			}
			info.RepStack, info.RepStackLen = path.RepStack, path.RepStackLen
			if tc.excluded {
				info.ExcludedMove = adapterLegalMove(t, pos, "g7g8")
				info.ExcludedPly = tc.ply
			}
			T1104Cuckoo = tc.option
			before := snapshotPVPosition(pos)
			got := alphaBetaPV(pos, 2, tc.ply, tc.alpha, tc.beta, true, false, false, info)
			if tc.cutoff {
				if got != 0 || info.TTProbes != 0 || info.Nodes != 0 {
					t.Fatalf("score=%d probes=%d nodes=%d, want draw before TT/search", got, info.TTProbes, info.Nodes)
				}
			} else if info.TTProbes == 0 {
				t.Fatal("search unexpectedly bypassed")
			}
			if info.RepStackLen != path.RepStackLen || info.cuckooStart != 0 {
				t.Fatal("search did not unwind repetition state")
			}
			assertPVPositionRestored(t, pos, before)
		})
	}
}

func TestCuckooRootHistory(t *testing.T) {
	old := T1104Cuckoo
	t.Cleanup(func() { T1104Cuckoo = old })
	for _, option := range []int{0, 1} {
		T1104Cuckoo = option
		for _, fixed := range []bool{false, true} {
			searcher, err := NewSearchEngineWithHash(1)
			if err != nil {
				t.Fatal(err)
			}
			pos, _ := cuckooCycle(t)
			var info *SearchInfo
			if fixed {
				info = searcher.SearchFixed(pos, 1, nil)
			} else {
				info = searcher.SearchIterativeDeepening(pos, 1, nil)
			}
			if info.RepStackLen != option || (option == 1 && info.RepStack[0] != pos.Hash()) || info.cuckooStart != 0 {
				t.Fatalf("option=%d fixed=%t: root history not preserved", option, fixed)
			}
		}
	}
}

func TestCuckooDrawFloor(t *testing.T) {
	old := T1104Cuckoo
	T1104Cuckoo = 1
	t.Cleanup(func() { T1104Cuckoo = old })
	for _, depth := range []int{0, 2} {
		pos, path := cuckooCycle(t)
		_, info := newProbcutContractStateForFEN(t, "6rk/8/8/8/8/8/8/KR6 b - - 3 1", depth)
		if err := info.evaluator.Reset(pos); err != nil {
			t.Fatal(err)
		}
		info.RepStack, info.RepStackLen = path.RepStack, path.RepStackLen
		info.evaluator.staticOracle = func(*Position) int { return -500 }
		// A position-only negative TT score cannot override a path-specific draw.
		info.tt.Set(pos.Hash(), EmptyMove, -500, 10, Exact, false)
		if got := alphaBetaPV(pos, depth, 3, -100, 100, false, false, false, info); got < 0 {
			t.Fatalf("depth=%d: available repetition returned losing score %d", depth, got)
		}
		if got := alphaBetaPV(pos, depth, 3, -100, -1, false, false, false, info); got != 0 {
			t.Fatalf("depth=%d: repetition did not cut at draw: %d", depth, got)
		}
	}
}

func TestCuckooNullSearchBoundary(t *testing.T) {
	old := T1104Cuckoo
	T1104Cuckoo = 1
	t.Cleanup(func() { T1104Cuckoo = old })
	pos, info := newProbcutContractStateForFEN(t, "6rk/8/8/8/8/8/8/KR6 b - - 3 1", 2)
	info.RepStackLen = 3
	before := snapshotPVPosition(pos)
	observed := false
	info.evaluator.staticOracle = func(p *Position) int {
		if info.evaluator.evaluationDepth() == 1 && p.Board == before.board && p.Turn() != before.side {
			observed = true
			if info.cuckooStart != 4 {
				t.Errorf("null child boundary=%d, want 4", info.cuckooStart)
			}
		}
		return 500
	}
	alphaBetaPV(pos, 2, 3, 0, 100, false, true, true, info)
	if !observed || info.NullMoveTries != 1 || info.cuckooStart != 0 || info.RepStackLen != 3 {
		t.Fatalf("null search not exercised/restored: observed=%t tries=%d boundary=%d length=%d", observed, info.NullMoveTries, info.cuckooStart, info.RepStackLen)
	}
	assertPVPositionRestored(t, pos, before)
}
