package engine

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func lowPlyOption(t *testing.T, value int) {
	t.Helper()
	old := T1104LowPly
	T1104LowPly = value
	t.Cleanup(func() { T1104LowPly = old })
}

func TestLowPlyUCIOption(t *testing.T) {
	lowPlyOption(t, 0)
	u := NewUCIEngine()
	var out bytes.Buffer
	u.handleCommand("uci", &out)
	if !strings.Contains(out.String(), fmt.Sprintf("option name T1104LowPly type spin default %d min 0 max 1", acceptedOptionDefaults["T1104LowPly"])) {
		t.Fatal("missing spin option with accepted default")
	}
	u.handleCommand("setoption name T1104LowPly value 1", &out)
	if T1104LowPly != 1 {
		t.Fatal("UCI did not enable low-ply history")
	}
	u.handleCommand("setoption name T1104LowPly value 0", &out)
	if T1104LowPly != 0 {
		t.Fatal("UCI did not disable low-ply history")
	}
}

func TestLowPlyLearningBoundsAndOwnership(t *testing.T) {
	lowPlyOption(t, 1)
	var h, other workerHistory
	quiet := NewMove(C3, D5, WhiteKnight, NoPiece, NoType, 0)
	for _, ply := range []int{0, 4} {
		h.updateLowPly(quiet, ply, 3)
		if h.lowPlyHistory[ply][C3][D5] != 9 {
			t.Fatalf("ply %d did not receive depth-squared bonus", ply)
		}
	}
	before := h.lowPlyHistory
	for _, ply := range []int{-1, 5, MaximumDepth} {
		h.updateLowPly(quiet, ply, 4)
	}
	for _, move := range []Move{
		EmptyMove,
		NewMove(C3, D5, WhiteKnight, BlackPawn, NoType, Capture),
		NewMove(E5, D6, WhitePawn, BlackPawn, NoType, EnPassant),
		NewMove(A7, A8, WhitePawn, NoPiece, Queen, 0),
		NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle),
	} {
		h.updateLowPly(move, 0, 4)
	}
	h.updateLowPly(quiet, 0, 0)
	T1104LowPly = 0
	h.updateLowPly(quiet, 0, 4)
	if h.lowPlyHistory != before || other.lowPlyHistory != ([lowPlyCount][64][64]int{}) {
		t.Fatal("ineligible update or cross-worker learning")
	}
	T1104LowPly = 1
	for i := 0; i < 1000; i++ {
		h.updateLowPly(quiet, 0, MaximumDepth)
	}
	if got := h.lowPlyHistory[0][C3][D5]; got <= 9 || got > historyMax {
		t.Fatalf("unbounded learning: %d", got)
	}
	h.ClearHistoryTable()
	if h.lowPlyHistory != ([lowPlyCount][64][64]int{}) {
		t.Fatal("history reset retained low-ply entries")
	}
}

func TestLowPlyQuietOrderingOnly(t *testing.T) {
	lowPlyOption(t, 1)
	var h workerHistory
	q1 := NewMove(C3, D5, WhiteKnight, NoPiece, NoType, 0)
	q2 := NewMove(E3, D5, WhiteKnight, NoPiece, NoType, 0)
	moves := []Move{q1, q2,
		NewMove(A1, A7, WhiteRook, BlackPawn, NoType, Capture),
		NewMove(A7, A8, WhitePawn, NoPiece, Queen, 0),
		NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle),
		NewMove(B1, C3, WhiteKnight, NoPiece, NoType, 0),
	}
	score := func(ply int) ([]Move, []int) {
		out, scores := make([]Move, len(moves)), make([]int, len(moves))
		h.scoreMovesIntoBuffer(moves, ply, EmptyMove, EmptyMove, moves[5], true, out, scores, nil, true)
		return out, scores
	}
	_, before := score(4)
	h.updateLowPly(q2, 4, 4)
	// Poison non-quiet and TT slots too: consumers must leave their scores alone.
	for _, move := range moves[2:] {
		h.lowPlyHistory[4][move.Source()][move.Destination()] = 1000
	}
	out, after := score(4)
	for i := range moves {
		want := before[i]
		if i == 1 {
			want += 16
		}
		if after[i] != want {
			t.Fatalf("move %s score=%d want=%d", moves[i].ToString(), after[i], want)
		}
	}
	// Equal piece/to, different from: only the trained quiet should move ahead.
	selectNextMove(out, after, 2, 0)
	if out[0] != q2 {
		t.Fatal("trained quiet did not gain ordering priority")
	}
	for _, ply := range []int{-1, 0, 3, 5} {
		_, got := score(ply)
		for i := range got {
			if got[i] != before[i] {
				t.Fatalf("low-ply history leaked into ply %d", ply)
			}
		}
	}
	T1104LowPly = 0
	_, off := score(4)
	for i := range off {
		if off[i] != before[i] {
			t.Fatal("disabled table changed ordering")
		}
	}
	if h.GetHistoryScore(q2, EmptyMove, EmptyMove) != 0 {
		t.Fatal("low-ply score leaked into pruning/reduction history")
	}
}

func TestLowPlySearchCutoff(t *testing.T) {
	const fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	for _, tc := range []struct {
		name           string
		on, ply, alpha int
		learn          bool
	}{
		{"on cutoff", 1, 4, -10, true},
		{"off cutoff", 0, 4, -10, false},
		{"deep cutoff", 1, 5, -10, false},
		{"fail low", 1, 4, 10, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lowPlyOption(t, tc.on)
			pos, info := newProbcutContractStateForFEN(t, fen, 1)
			alphaBetaPV(pos, 1, tc.ply, tc.alpha, tc.alpha+1, true, false, false, info)
			if info.Stopped {
				t.Fatal("search stopped")
			}
			if tc.alpha < 0 && info.BetaCutoffs != 1 {
				t.Fatalf("wanted one real quiet cutoff, got %d", info.BetaCutoffs)
			}
			var sum int
			for _, table := range info.history.lowPlyHistory {
				for _, row := range table {
					for _, value := range row {
						sum += value
					}
				}
			}
			want := 0
			if tc.learn {
				want = 1
				move, _, _, _, hit, _ := info.tt.Get(pos.Hash())
				if !hit || info.history.lowPlyHistory[tc.ply][move.Source()][move.Destination()] != 1 {
					t.Fatal("cutoff did not train its actual ply/from/to")
				}
			}
			if sum != want {
				t.Fatalf("learned bonus sum=%d want=%d", sum, want)
			}
		})
	}
}
