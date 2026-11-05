package engine

import "testing"

func TestPriorBonusOption(t *testing.T) {
	for _, param := range TunableSearchParams {
		if param.Name == "T1104PriorBonus" {
			if param.Ptr != &T1104PriorBonus || param.Def != 0 || param.Min != 0 || param.Max != 1 || T1104PriorBonus != 0 {
				t.Fatal("prior bonus must ship as a default-off boolean spin")
			}
			return
		}
	}
	t.Fatal("missing prior bonus option")
}

func TestPriorBonusEligibilityAndDepth(t *testing.T) {
	old := T1104PriorBonus
	defer func() { T1104PriorBonus = old }()
	quiet := NewMove(G8, F6, BlackKnight, NoPiece, NoType, 0)
	context := NewMove(E2, E4, WhitePawn, NoPiece, NoType, 0)
	for _, tc := range []struct {
		name                  string
		on, depth, ply, score int
		previous              Move
		tt, stopped           bool
		want                  int
	}{
		{"enabled equality", 1, 2, 2, 100, quiet, false, false, 64},
		{"deeper fail low", 1, 4, 2, 99, quiet, false, false, 256},
		{"rate capped", 1, 100, 2, 99, quiet, false, false, 6400},
		{"off", 0, 4, 2, 99, quiet, false, false, 0},
		{"improved alpha", 1, 4, 2, 101, quiet, false, false, 0},
		{"TT predecessor", 1, 4, 2, 99, quiet, true, false, 0},
		{"null predecessor", 1, 4, 2, 99, EmptyMove, false, false, 0},
		{"capture", 1, 4, 2, 99, NewMove(F6, E4, BlackKnight, WhitePawn, NoType, Capture), false, false, 0},
		{"promotion", 1, 4, 2, 99, NewMove(A2, A1, BlackPawn, NoPiece, Queen, 0), false, false, 0},
		{"root", 1, 4, 0, 99, quiet, false, false, 0},
		{"stopped", 1, 4, 2, 99, quiet, false, true, 0},
		{"no continuation", 1, 4, 1, 99, quiet, false, false, 256},
	} {
		t.Run(tc.name, func(t *testing.T) {
			T1104PriorBonus = tc.on
			info := &SearchInfo{history: new(workerHistory), Stopped: tc.stopped}
			if tc.ply > 0 {
				info.MoveStack[tc.ply-1] = tc.previous
				if tc.tt {
					info.priorBonusTTStack[tc.ply-1] = tc.previous
				}
			}
			if tc.ply >= 2 {
				info.MoveStack[tc.ply-2] = context
			}
			info.rewardPriorQuiet(tc.depth, tc.ply, 100, tc.score)
			h := info.history
			if got := h.historyTable[BlackKnight][F6]; got != tc.want {
				t.Fatalf("main=%d want=%d", got, tc.want)
			}
			wantCont := 0
			if tc.ply >= 2 {
				wantCont = tc.want
			}
			if got := h.continuationHistory[WhitePawn][E4][BlackKnight][F6]; got != wantCont {
				t.Fatalf("continuation=%d want=%d", got, wantCont)
			}
			if h.historyTable[WhiteKnight][F6] != 0 || h.followupHistory[WhitePawn][E4][BlackKnight][F6] != 0 {
				t.Fatal("updated wrong side or follow-up")
			}
			if tc.want == 0 && (h.historyTable != [13][64]int{} || h.continuationHistory != [13][64][13][64]int{}) {
				t.Fatal("rejected signal changed history")
			}
		})
	}
}

func TestPriorBonusAlphaBetaCompletedFailLow(t *testing.T) {
	old := T1104PriorBonus
	defer func() { T1104PriorBonus = old }()
	for _, black := range []bool{false, true} {
		for _, on := range []int{0, 1} {
			for _, ttPrior := range []bool{false, true} {
				T1104PriorBonus = on
				fen := "4k3/pp6/8/8/8/5N2/6PP/4K3 b - - 0 1"
				previous := NewMove(G1, F3, WhiteKnight, NoPiece, NoType, 0)
				context := NewMove(E7, E5, BlackPawn, NoPiece, NoType, 0)
				if black {
					fen = "4k3/pp6/5n2/8/8/8/6PP/4K3 w - - 0 1"
					previous = NewMove(G8, F6, BlackKnight, NoPiece, NoType, 0)
					context = NewMove(E2, E4, WhitePawn, NoPiece, NoType, 0)
				}
				pos, info := qstaticFixture(t, fen)
				info.MoveStack[0], info.MoveStack[1] = context, previous
				info.history.SetLastMovePlayed(previous)
				if ttPrior {
					info.priorBonusTTStack[1] = previous
				}
				score := alphaBetaPV(pos, 1, 2, 2000, 2100, true, false, false, info)
				if info.Stopped || score > 2000 || info.AllNodes == 0 {
					t.Fatalf("fixture did not complete fail low: score=%d all=%d", score, info.AllNodes)
				}
				want := 0
				if on == 1 && !ttPrior {
					want = 16
				}
				if got := info.history.historyTable[previous.MovingPiece()][previous.Destination()]; got != want {
					t.Fatalf("black=%v on=%d tt=%v: main=%d want=%d", black, on, ttPrior, got, want)
				}
				if got := info.history.continuationHistory[context.MovingPiece()][context.Destination()][previous.MovingPiece()][previous.Destination()]; got != want {
					t.Fatalf("continuation=%d want=%d", got, want)
				}
			}
		}
	}
}

// This forced reply receives the normal parent cutoff reward in both cases.
// Only the non-TT case may also receive the child's fail-low prior reward.
func TestPriorBonusUsesParentsTTMove(t *testing.T) {
	old := T1104PriorBonus
	T1104PriorBonus = 1
	defer func() { T1104PriorBonus = old }()
	move := NewMove(A8, B8, BlackKing, NoPiece, NoType, 0)
	for _, ttParent := range []bool{false, true} {
		pos, info := qstaticFixture(t, "k7/8/2K5/8/8/8/8/R7 b - - 0 1")
		if ttParent {
			info.tt.Set(pos.Hash(), move, 0, 0, Exact, false)
		}
		score := alphaBetaPV(pos, 2, 1, -2100, -2000, true, false, false, info)
		if info.Stopped || score < -2000 {
			t.Fatalf("forced reply did not cut: %d", score)
		}
		want := 64
		if !ttParent {
			want = h1HistoryResponse(16, h1HistoryTarget, h1HistoryRate(2))
		}
		if got := info.history.historyTable[BlackKing][B8]; got != want {
			t.Fatalf("ttParent=%v: main=%d want=%d", ttParent, got, want)
		}
	}
}

func TestPriorBonusSkipsRestrictedSearch(t *testing.T) {
	old := T1104PriorBonus
	T1104PriorBonus = 1
	defer func() { T1104PriorBonus = old }()
	pos, info := qstaticFixture(t, "4k3/pp6/8/8/8/5N2/6PP/4K3 b - - 0 1")
	previous := NewMove(G1, F3, WhiteKnight, NoPiece, NoType, 0)
	info.MoveStack[0] = previous
	info.history.SetLastMovePlayed(previous)
	info.ExcludedMove = NewMove(A7, A6, BlackPawn, NoPiece, NoType, 0)
	info.ExcludedPly = 1
	score := alphaBetaPV(pos, 1, 1, 2000, 2100, true, false, false, info)
	if info.Stopped || score > 2000 || info.AllNodes == 0 {
		t.Fatal("restricted fixture did not fail low")
	}
	if info.history.historyTable[WhiteKnight][F3] != 0 {
		t.Fatal("restricted upper bound rewarded predecessor")
	}
}
