package engine

import "testing"

func enableContPlies(t *testing.T, value int) {
	t.Helper()
	old := T1104ContPlies
	t.Cleanup(func() { T1104ContPlies = old })
	T1104ContPlies = value
}

func TestContPliesOptionAndAncestors(t *testing.T) {
	enableContPlies(t, 0)
	found := false
	for _, param := range TunableSearchParams {
		if param.Name == "T1104ContPlies" {
			found = true
			if param.Ptr != &T1104ContPlies || param.Def != acceptedOptionDefaults["T1104ContPlies"] || param.Min != 0 || param.Max != 1 {
				t.Fatalf("invalid spin option: %+v", param)
			}
		}
	}
	if !found || !setTunableParam("T1104ContPlies", "1") || T1104ContPlies != 1 {
		t.Fatal("option was not registered/applied")
	}
	var info SearchInfo
	for i := range info.MoveStack {
		info.MoveStack[i] = NewMove(A2, Square(i%64), WhitePawn, NoPiece, NoType, 0)
	}
	for _, ply := range []int{0, 1, 2, 3, 4, 5, 6, 7, MaximumDepth - 1} {
		four, six := info.contPlies(ply)
		want4, want6 := EmptyMove, EmptyMove
		if ply >= 4 {
			want4 = info.MoveStack[ply-4]
		}
		if ply >= 6 {
			want6 = info.MoveStack[ply-6]
		}
		if four != want4 || six != want6 {
			t.Fatalf("ply=%d: got %v/%v want %v/%v", ply, four, six, want4, want6)
		}
	}
	info.MoveStack[2] = EmptyMove // a null move has no continuation context
	if four, _ := info.contPlies(6); four != EmptyMove {
		t.Fatal("null move acquired a context")
	}
	setTunableParam("T1104ContPlies", "0")
	if four, six := info.contPlies(6); four != EmptyMove || six != EmptyMove {
		t.Fatal("disabled option read ancestors")
	}
}

func TestContPliesHalfWeightOrderingAndConsumers(t *testing.T) {
	enableContPlies(t, 1)
	var h workerHistory
	move := h1TestQuiet(G1, F3, WhiteKnight)
	other := h1TestQuiet(B1, C3, WhiteKnight)
	prev := h1TestQuiet(E7, E5, BlackPawn)
	prev2 := h1TestQuiet(E2, E4, WhitePawn)
	prev4 := h1TestQuiet(D2, D4, WhitePawn)
	prev6 := h1TestQuiet(C2, C4, WhitePawn)
	piece, to := move.MovingPiece(), move.Destination()
	h.historyTable[piece][to] = 100
	h.continuationHistory[prev.MovingPiece()][prev.Destination()][piece][to] = 200
	h.followupHistory[prev2.MovingPiece()][prev2.Destination()][piece][to] = 300
	h.followupHistory[prev4.MovingPiece()][prev4.Destination()][piece][to] = -7001
	h.followupHistory[prev6.MovingPiece()][prev6.Destination()][piece][to] = -3001
	const want = -4400 // 100+200+300-3500-1500, truncating each half toward zero
	if got := h.GetHistoryScore(move, prev, prev2, prev4, prev6); got != want {
		t.Fatalf("history=%d want=%d", got, want)
	}
	var out [2]Move
	var scores [2]int
	n, _ := h.scoreMovesIntoBuffer([]Move{move, other}, 6, prev, prev2, EmptyMove, false, out[:], scores[:], nil, false, prev4, prev6)
	if n != 2 || scores[0] != want || scores[1] != 0 {
		t.Fatalf("ordering scores=%v count=%d", scores, n)
	}
	selectNextMove(out[:], scores[:], n, 0)
	if out[0] != other {
		t.Fatal("distant history did not change quiet ordering")
	}
	if want >= -tuneHistoryPruneThreshold || h1NormalizedLMRTerm(want) != 1 {
		t.Fatal("negative distant history did not reach pruning/LMR scale")
	}
	T1104ContPlies = 0
	if got := h.GetHistoryScore(move, prev, prev2, prev4, prev6); got != 600 {
		t.Fatalf("disabled history=%d want=600", got)
	}
	h.scoreMovesIntoBuffer([]Move{move, other}, 6, prev, prev2, EmptyMove, false, out[:], scores[:], nil, false, prev4, prev6)
	if scores[0] != 600 {
		t.Fatalf("disabled ordering=%d want=600", scores[0])
	}
	T1104ContPlies = 1
	for _, noisy := range []Move{
		NewMove(E4, D5, WhitePawn, BlackPawn, NoType, Capture),
		NewMove(A7, A8, WhitePawn, NoPiece, Queen, 0),
	} {
		if h.GetHistoryScore(noisy, prev, prev2, prev4, prev6) != 0 {
			t.Fatal("noisy move got quiet history")
		}
	}
}

func TestContPliesCompletedNodeHalfRate(t *testing.T) {
	enableContPlies(t, 1)
	loser := h1TestQuiet(E2, E4, WhitePawn)
	winner := h1TestQuiet(D2, D4, WhitePawn)
	later := h1TestQuiet(G1, F3, WhiteKnight)
	prev2 := h1TestQuiet(B1, C3, WhiteKnight)
	prev4 := h1TestQuiet(C1, F4, WhiteBishop)
	prev6 := h1TestQuiet(A2, A4, WhitePawn)
	for _, option := range []int{0, 1} {
		T1104ContPlies = option
		var h workerHistory
		h.h1UpdateCompletedNode([]Move{loser, winner, later}, winner, EmptyMove, prev2, 10, 0, 20, prev4, EmptyMove, prev6)
		for _, test := range []struct {
			move Move
			sign int
		}{{loser, -1}, {winner, 1}, {later, 0}} {
			if got := h.historyTable[test.move.MovingPiece()][test.move.Destination()]; got != test.sign*1600 {
				t.Fatalf("main update=%d", got)
			}
			for _, previous := range []Move{prev2, prev4, prev6} {
				want := test.sign * 1600
				if previous != prev2 {
					want = test.sign * 800 * option
				}
				if got := h.followupHistory[previous.MovingPiece()][previous.Destination()][test.move.MovingPiece()][test.move.Destination()]; got != want {
					t.Fatalf("option=%d previous=%s update=%d want=%d", option, previous.ToString(), got, want)
				}
			}
		}
		if h.followupHistory[NoPiece][A1][winner.MovingPiece()][winner.Destination()] != 0 {
			t.Fatal("empty context updated a table slot")
		}
		before := h.followupHistory[prev4.MovingPiece()][prev4.Destination()][winner.MovingPiece()][winner.Destination()]
		h.h1UpdateCompletedNode([]Move{winner}, winner, EmptyMove, prev2, 10, 20, 20, prev4, prev6)
		if got := h.followupHistory[prev4.MovingPiece()][prev4.Destination()][winner.MovingPiece()][winner.Destination()]; got != before {
			t.Fatal("fail-low trained distant history")
		}
	}
}

func TestContPliesLegacyHalfBonus(t *testing.T) {
	enableContPlies(t, 1)
	move := h1TestQuiet(G1, F3, WhiteKnight)
	prev4 := h1TestQuiet(D2, D4, WhitePawn)
	prev6 := h1TestQuiet(C2, C4, WhitePawn)
	for _, depth := range []int{5, 64} {
		var h workerHistory
		h.UpdateHistoryTable(move, EmptyMove, EmptyMove, depth, prev4, prev6)
		want := min(depth*depth, tuneHistoryBonusCap) / 2
		for _, previous := range []Move{prev4, prev6} {
			entry := &h.followupHistory[previous.MovingPiece()][previous.Destination()][move.MovingPiece()][move.Destination()]
			if *entry != want {
				t.Fatalf("depth=%d bonus=%d want=%d", depth, *entry, want)
			}
		}
		h.PenalizeHistoryTable(move, EmptyMove, EmptyMove, depth, prev4, prev6)
		if got := h.followupHistory[prev4.MovingPiece()][prev4.Destination()][move.MovingPiece()][move.Destination()]; got != -want*want/historyMax {
			t.Fatalf("penalty=%d", got)
		}
	}
}

func TestContPliesSearchTrainsDistantAncestors(t *testing.T) {
	enableContPlies(t, 1)
	pos, info := newProbcutContractStateForFEN(t, "7k/8/8/8/8/8/8/KR6 w - - 0 1", 2)
	prev4 := h1TestQuiet(E2, E4, WhitePawn)
	prev6 := h1TestQuiet(C1, F4, WhiteBishop)
	info.MoveStack[2], info.MoveStack[0] = prev4, prev6
	alphaBetaPV(pos, 2, 6, -INFINITY, INFINITY, true, true, false, info)
	if info.Stopped {
		t.Fatal("small search stopped")
	}
	trained := false
	for piece := range info.history.historyTable {
		for to, main := range info.history.historyTable[piece] {
			four := info.history.followupHistory[prev4.MovingPiece()][prev4.Destination()][piece][to]
			six := info.history.followupHistory[prev6.MovingPiece()][prev6.Destination()][piece][to]
			if four != 0 || six != 0 {
				trained = true
				if four != 32 || six != 32 || main != 64 {
					t.Fatalf("search updates main/four/six=%d/%d/%d want=64/32/32", main, four, six)
				}
			}
		}
	}
	if !trained {
		t.Fatal("completed search did not train distant ancestors")
	}
}
