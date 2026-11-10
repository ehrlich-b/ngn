package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func wave2MarginState(t *testing.T) (*Position, *SearchInfo) {
	t.Helper()
	old := W2NullMargin
	t.Cleanup(func() { W2NullMargin = old })
	W2NullMargin = 1
	return wave2NullState(t)
}

func TestWave2NullMarginReduction(t *testing.T) {
	for _, options := range [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}} {
		for _, tc := range []struct {
			name                           string
			eval, correction, extra, depth int
		}{
			{"negative", 99, 0, 0, 12},
			{"zero", 100, 0, 0, 12},
			{"199", 299, 0, 0, 12},
			{"200", 300, 0, 1, 12},
			{"399", 499, 0, 1, 12},
			{"400", 500, 0, 2, 12},
			{"bounded", 2000, 0, 2, 12},
			{"corrected crosses 200", 299, 1, 1, 12},
			{"corrected below 200", 300, -1, 0, 12},
			{"positive mate inside", MATE_IN_MAX - 1, 0, 2, 12},
			{"positive mate boundary", MATE_IN_MAX, 0, 0, 12},
			{"positive mate outside", MATE_IN_MAX + 1, 0, 0, 12},
			{"negative mate inside", -MATE_IN_MAX + 1, 0, 0, 12},
			{"negative mate boundary", -MATE_IN_MAX, 0, 0, 12},
			{"negative mate outside", -MATE_IN_MAX - 1, 0, 0, 12},
			{"unknown", unknownStaticEval, 0, 0, 12},
			{"depth floor", 500, 0, 2, 2},
		} {
			t.Run(fmt.Sprintf("verify%d_margin%d/%s", options[0], options[1], tc.name), func(t *testing.T) {
				p, info := wave2MarginState(t)
				W2NullVerify, W2NullMargin = options[0], options[1]
				info.evaluator.staticOracle = func(*Position) int { return tc.eval }
				// Set a real correction-history contribution of exactly +/-1 cp.
				idx := pawnCorrectionIndex(p.Board.pieces[WhitePawn], p.Board.pieces[BlackPawn])
				info.history.pawnCorrectionHistory[p.Turn()][idx] = tc.correction * 22
				if info.history.correctionValue(p.Turn(), idx) != tc.correction {
					t.Fatal("correction fixture does not produce the declared centipawns")
				}
				// A shallow exact TT hit refines staticEval to 900 but cannot cut off.
				// In particular, a negative corrected surplus remains eligible via TT;
				// the reduction must still use the corrected pre-TT value.
				info.tt.Set(p.Hash(), EmptyMove, 900, 1, Exact, false)
				before := snapshotPVPosition(p)
				prior := info.history.GetLastMovePlayed()
				moves, evals := info.MoveStack, info.StaticEvalStack
				extra := 0
				if options == [2]int{1, 1} {
					extra = tc.extra
				}
				wantDepth := max(0, tc.depth-(NULL_MOVE_R+tuneNullMoveOffset+tc.depth/tuneNullMoveDepthDivisor)-extra)
				calls := 0
				info.nullSearch = func(pos *Position, d, ply, a, b int, pv, canNull, cut bool, s *SearchInfo) int {
					calls++
					if calls == 1 {
						if d != wantDepth || ply != 3 || a != -100 || b != -99 || pv || canNull || cut || s.nullVerification {
							t.Fatalf("actual null child: depth=%d want=%d ply=%d window=[%d,%d] flags=%v/%v/%v", d, wantDepth, ply, a, b, pv, canNull, cut)
						}
						if pos.Hash() == before.hash || s.evaluator.evaluationDepth() != 1 {
							t.Fatal("missing speculative null transition")
						}
						return -100
					}
					if calls != 2 || W2NullVerify == 0 || tc.depth < 8 || d != max(1, tc.depth-NULL_MOVE_R) || ply != 2 || a != 99 || b != 100 || pv || canNull || !cut || !s.nullVerification {
						t.Fatal("margin changed parent verification contract")
					}
					if !reflect.DeepEqual(snapshotPVPosition(pos), before) || s.evaluator.evaluationDepth() != 0 || s.history.GetLastMovePlayed() != prior || s.MoveStack != moves {
						t.Fatal("verification did not receive restored parent state")
					}
					s.MoveStack[2], s.StaticEvalStack[2] = EmptyMove, -999
					s.history.SetLastMovePlayed(EmptyMove)
					return 100
				}
				got := alphaBetaPV(p, tc.depth, 2, 99, 100, false, true, true, info)
				requireWave2NullRestored(t, p, info, before, prior)
				evals[2] = 900
				wantCalls := 1
				if options[0] == 1 && tc.depth >= 8 {
					wantCalls++
				}
				if got != 100 || calls != wantCalls || info.NullMoveTries != 1 || info.NullMoveCutoffs != 1 || info.Stopped || info.MoveStack != moves || info.StaticEvalStack != evals {
					t.Fatalf("cutoff/state mismatch: score=%d calls=%d", got, calls)
				}
				t.Logf("verify=%d margin=%d corrected=%d refined=900 beta=100 extra=%d actual_child=%d calls=%d", W2NullVerify, W2NullMargin, tc.eval+tc.correction, extra, wantDepth, calls)
			})
		}
	}
}

func TestWave2NullMarginRejectedVerification(t *testing.T) {
	p, info := wave2MarginState(t)
	info.evaluator.staticOracle = func(*Position) int { return 500 }
	before := snapshotPVPosition(p)
	prior := info.history.GetLastMovePlayed()
	calls, ordinary := 0, false
	info.nullSearch = func(pos *Position, d, ply, a, b int, pv, canNull, cut bool, s *SearchInfo) int {
		calls++
		if calls == 1 {
			if d != 0 || ply != 3 { // 8 - (4+1+8/6) - 2, floored at zero.
				t.Fatalf("actual margin null depth=%d ply=%d", d, ply)
			}
			return -100
		}
		if calls != 2 || !s.nullVerification || d != 4 || ply != 2 {
			t.Fatal("missing parent verification")
		}
		return 99
	}
	info.evaluator.transitionObserver = func(observed *Position, e *workerEvaluator) {
		if e.evaluationDepth() == 1 && observed.Board != before.board {
			ordinary = true
			info.control.RequestStop()
		}
	}
	got := alphaBetaPV(p, 8, 2, 99, 100, false, true, true, info)
	requireWave2NullRestored(t, p, info, before, prior)
	if calls != 2 || !ordinary || !info.Stopped || got != stoppedSearchScore || info.NullVerifyFailLows != 1 || info.NullMoveCutoffs != 0 {
		t.Fatal("margin cutoff rejection did not continue ordinary search")
	}
	t.Logf("margin=1 actual_child=0 verify_fail_low=%d ordinary=%v restored=true", info.NullVerifyFailLows, ordinary)
}

func TestWave2NullMarginMateWindows(t *testing.T) {
	_, _ = wave2MarginState(t)
	for _, beta := range []int{unknownStaticEval, -MATE_IN_MAX - 1, -MATE_IN_MAX, MATE_IN_MAX, MATE_IN_MAX + 1} {
		if got := wave2NullMargin(500, beta); got != 0 {
			t.Fatalf("invalid beta=%d extra=%d", beta, got)
		}
	}
	if wave2NullMargin(-MATE_IN_MAX+201, -MATE_IN_MAX+1) != 1 || wave2NullMargin(MATE_IN_MAX-1, MATE_IN_MAX-201) != 1 {
		t.Fatal("finite values just inside mate boundaries should remain usable")
	}
}

func TestWave2NullMarginOffParentTrace(t *testing.T) {
	originalSearchOptions(t)
	oldVerify, oldMargin := W2NullVerify, W2NullMargin
	t.Cleanup(func() { W2NullVerify, W2NullMargin = oldVerify, oldMargin })
	W2NullVerify, W2NullMargin = 1, 0
	data, err := os.ReadFile("../experiments/wave2-1104/null-margin/parent-trace.json")
	if err != nil {
		t.Fatal(err)
	}
	var baseline []map[string]interface{}
	if err := json.Unmarshal(data, &baseline); err != nil || len(baseline) != 4 {
		t.Fatalf("expected four parent fixtures: %v", err)
	}
	net, err := LoadDefaultNet()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range baseline {
		e, err := NewSearchEngineWithHash(1)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.SelectNGNN1Evaluator(net); err != nil {
			t.Fatal(err)
		}
		p, err := ParseFEN(want["fen"].(string))
		if err != nil {
			t.Fatal(err)
		}
		info := e.SearchIterativeDeepening(p, 4, nil)
		pv := []string{}
		for _, m := range e.ExtractPVFromTT(p, info.BestMove, 4) {
			pv = append(pv, m.ToString())
		}
		actual := map[string]interface{}{"fen": want["fen"], "depth": info.Depth, "nodes": info.Nodes, "qnodes": info.QNodes, "score": info.BestScore, "best": info.BestMove.ToString(), "pv": pv, "null_tries": info.NullMoveTries, "null_cutoffs": info.NullMoveCutoffs, "stopped": info.Stopped, "net_sha256": DefaultNetSHA256}
		encoded, err := json.Marshal(actual)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]interface{}
		if err := json.Unmarshal(encoded, &got); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("margin-off differs from passed parent: got=%s want=%v err=%v", encoded, want, err)
		}
		t.Logf("parent trace %s", encoded)
	}
	for _, p := range TunableSearchParams {
		if p.Name == "W2NullMargin" {
			if p.Ptr != &W2NullMargin || p.Def != acceptedOptionDefaults["W2NullMargin"] || p.Min != 0 || p.Max != 1 {
				t.Fatal("bad margin option metadata")
			}
			return
		}
	}
	t.Fatal("W2NullMargin UCI option missing")
}
