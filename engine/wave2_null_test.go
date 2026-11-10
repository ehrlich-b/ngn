package engine

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const wave2NullFEN = "7k/8/8/3pP3/8/8/8/KR6 w - d6 17 1"

func wave2NullState(t *testing.T) (*Position, *SearchInfo) {
	t.Helper()
	oldOption, oldToggles := W2NullVerify, SearchToggles
	t.Cleanup(func() { W2NullVerify, SearchToggles = oldOption, oldToggles })
	W2NullVerify = 1
	// Keep the injected contract focused on NMP, rather than earlier cutoffs.
	SearchToggles.RFP, SearchToggles.Probcut, SearchToggles.IID = false, false, false
	p, info := newProbcutContractStateForFEN(t, wave2NullFEN, 8)
	info.evaluator.staticOracle = func(*Position) int { return 200 }
	prior := NewMove(A1, A2, WhiteKing, NoPiece, NoType, 0)
	info.history.SetLastMovePlayed(prior)
	for i := range info.MoveStack {
		info.MoveStack[i] = prior
		info.StaticEvalStack[i] = i + 50
	}
	return p, info
}

func requireWave2NullRestored(t *testing.T, p *Position, info *SearchInfo, before pvPositionSnapshot, prior Move) {
	t.Helper()
	if after := snapshotPVPosition(p); !reflect.DeepEqual(after, before) {
		t.Fatalf("position not restored: before=%+v after=%+v", before, after)
	}
	if info.evaluator.evaluationDepth() != 0 || info.history.GetLastMovePlayed() != prior || info.MoveStack[1] != prior || info.nullVerification || info.RepStackLen != 0 || info.FrameDepth != 0 {
		t.Fatalf("unbalanced worker state: eval=%d prior=%v suppression=%v rep=%d frame=%d", info.evaluator.evaluationDepth(), info.history.GetLastMovePlayed(), info.nullVerification, info.RepStackLen, info.FrameDepth)
	}
}

func TestWave2NullVerifyContract(t *testing.T) {
	originalSearchOptions(t)
	for _, tc := range []struct {
		name                             string
		depth, option, verificationScore int
		stop                             string
		wantVerify                       bool
	}{
		{"verified cutoff", 8, 1, 100, "", true},
		{"verification depth floor", 8, 1, 100, "", true},
		{"rejected cutoff resumes moves", 8, 1, 99, "ordinary", true},
		{"rejected cutoff completes fail low", 8, 1, 99, "complete", true},
		{"shallow cutoff", 7, 1, 0, "", false},
		{"off deep cutoff", 8, 0, 0, "", false},
		{"cancel null child", 8, 1, 100, "null", false},
		{"cancel verification", 8, 1, 100, "verification", true},
		{"null fail low", 8, 1, 100, "null fail low", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, info := wave2NullState(t)
			W2NullVerify = tc.option
			info.maxNodes = 4096
			if tc.name == "verification depth floor" {
				oldR := NULL_MOVE_R
				NULL_MOVE_R = 8
				t.Cleanup(func() { NULL_MOVE_R = oldR })
			}
			before := snapshotPVPosition(p)
			prior := info.history.GetLastMovePlayed()
			moves, evals := info.MoveStack, info.StaticEvalStack
			calls, ordinary := 0, false
			info.evaluator.transitionObserver = func(observed *Position, e *workerEvaluator) {
				// A null transition clears EP and leaves the board unchanged.
				if e.evaluationDepth() == 1 && observed.Board != before.board {
					ordinary = true
					if tc.stop == "ordinary" || tc.stop == "null fail low" {
						info.control.RequestStop()
					}
				}
			}
			info.nullSearch = func(pos *Position, d, ply, a, b int, pv, canNull, cut bool, s *SearchInfo) int {
				calls++
				if calls == 1 {
					wantDepth := tc.depth - (NULL_MOVE_R + tuneNullMoveOffset + tc.depth/tuneNullMoveDepthDivisor)
					if wantDepth < 0 {
						wantDepth = 0
					}
					if d != wantDepth || ply != 3 || a != -100 || b != -99 || pv || canNull || cut || s.nullVerification {
						t.Fatalf("null call d=%d ply=%d [%d,%d] flags=%v/%v/%v", d, ply, a, b, pv, canNull, cut)
					}
					if pos.Hash() == before.hash || pos.EnPassant != NoSquare || pos.HalfMoveClock != before.halfMoveClock+1 || s.evaluator.evaluationDepth() != 1 || s.history.GetLastMovePlayed() != EmptyMove || s.MoveStack[2] != EmptyMove {
						t.Fatal("null transition missing")
					}
					if tc.stop == "null" {
						s.control.RequestStop()
						return alphaBetaPV(pos, 0, ply, a, b, false, false, false, s)
					}
					if tc.stop == "null fail low" {
						return -99
					}
					return -100
				}
				if calls != 2 || !tc.wantVerify {
					t.Fatalf("unexpected recursive seam call %d", calls)
				}
				if d != max(1, tc.depth-NULL_MOVE_R) || ply != 2 || a != 99 || b != 100 || pv || canNull || !cut || !s.nullVerification || s.RepStackLen != 0 {
					t.Fatalf("verification call d=%d ply=%d [%d,%d] suppressed=%v rep=%d", d, ply, a, b, s.nullVerification, s.RepStackLen)
				}
				if !reflect.DeepEqual(snapshotPVPosition(pos), before) || s.evaluator.evaluationDepth() != 0 || s.history.GetLastMovePlayed() != prior || s.MoveStack != moves {
					t.Fatal("verification entered before restoring real state")
				}
				// Deliberately clobber same-ply path context to prove save/restore.
				s.MoveStack[2], s.StaticEvalStack[2] = EmptyMove, -999
				s.history.SetLastMovePlayed(EmptyMove)
				if tc.stop == "verification" {
					s.control.RequestStop()
					return alphaBetaPV(pos, d, ply, a, b, pv, true, cut, s)
				}
				if tc.stop == "complete" {
					s.evaluator.staticOracle = func(*Position) int { return 0 }
				}
				return tc.verificationScore
			}
			got := alphaBetaPV(p, tc.depth, 2, 99, 100, false, true, true, info)
			requireWave2NullRestored(t, p, info, before, prior)
			if tc.wantVerify != (info.NullVerifySearches == 1) || info.NullMoveTries != 1 {
				t.Fatalf("counters tries=%d verify=%d", info.NullMoveTries, info.NullVerifySearches)
			}
			switch tc.stop {
			case "complete":
				if !ordinary || info.Stopped || got >= 100 || info.NullMoveCutoffs != 0 || info.NullVerifyFailLows != 1 {
					t.Fatalf("ordinary fail-low incomplete: score=%d nodes=%d stopped=%v", got, info.Nodes, info.Stopped)
				}
			case "ordinary", "null fail low":
				if !ordinary || !info.Stopped || got != stoppedSearchScore || info.NullMoveCutoffs != 0 {
					t.Fatal("rejected null cutoff did not resume ordinary move search")
				}
			case "null", "verification":
				if !info.Stopped || got != stoppedSearchScore || info.NullMoveCutoffs != 0 || info.NullVerifyCutoffs != 0 {
					t.Fatal("cancelled result trusted")
				}
			default:
				if info.Stopped || got != 100 || info.NullMoveCutoffs != 1 || ordinary {
					t.Fatal("expected null cutoff")
				}
				if info.MoveStack != moves {
					t.Fatal("move history not restored")
				}
			}
			if tc.wantVerify {
				// alphaBetaPV owns this node's static slot; verification must preserve it.
				evals[2] = 200
				if !ordinary && info.StaticEvalStack != evals {
					t.Fatal("verification leaked static-eval context")
				}
				if tc.stop == "ordinary" && info.NullVerifyFailLows != 1 {
					t.Fatal("missing fail-low witness")
				}
				if tc.stop == "" && info.NullVerifyCutoffs != 1 {
					t.Fatal("missing verified-cutoff witness")
				}
			}
			t.Logf("null=%d verify=%d verified=%d rejected=%d ordinary=%v stopped=%v", info.NullMoveTries, info.NullVerifySearches, info.NullVerifyCutoffs, info.NullVerifyFailLows, ordinary, info.Stopped)
		})
	}
}

func TestWave2NullVerifySuppression(t *testing.T) {
	p, info := wave2NullState(t)
	before := snapshotPVPosition(p)
	prior := info.history.GetLastMovePlayed()
	calls := 0
	info.nullSearch = func(pos *Position, d, ply, a, b int, pv, canNull, cut bool, s *SearchInfo) int {
		calls++
		if calls == 1 {
			return -100
		}
		if calls != 2 || !s.nullVerification {
			t.Fatal("null pruning entered verification subtree")
		}
		attempts := s.NullMoveTries
		// Enter the real reduced search with canNull=true, and observe real children.
		transitions := 0
		s.evaluator.transitionObserver = func(observed *Position, e *workerEvaluator) {
			transitions++
			if !s.nullVerification || s.allowNullMove(99, 100) {
				t.Fatal("descendant lost suppression")
			}
		}
		score := alphaBetaPV(pos, d, ply, a, b, false, true, true, s)
		if s.NullMoveTries != attempts || transitions == 0 || s.Stopped {
			t.Fatalf("nested suppression not exercised: tries=%d transitions=%d stopped=%v", s.NullMoveTries, transitions, s.Stopped)
		}
		t.Logf("real verification transitions=%d score=%d null attempts=%d", transitions, score, s.NullMoveTries-attempts)
		return 100
	}
	if got := alphaBetaPV(p, 8, 2, 99, 100, false, true, true, info); got != 100 || info.NullVerifyCutoffs != 1 {
		t.Fatal("verification failed")
	}
	requireWave2NullRestored(t, p, info, before, prior)
	// Same worker, fresh sibling search: pruning must be available again.
	info.evaluator.transitionObserver = nil
	info.tt.Clear()
	info.nullSearch = func(*Position, int, int, int, int, bool, bool, bool, *SearchInfo) int { return -100 }
	got := alphaBetaPV(p, 7, 2, 99, 100, false, true, true, info)
	if got != 100 || info.NullMoveTries != 2 || info.NullVerifySearches != 1 {
		t.Fatal("suppression leaked to sibling")
	}
	other := &SearchInfo{}
	if !other.allowNullMove(99, 100) {
		t.Fatal("suppression leaked to another worker")
	}
}

func TestWave2NullVerifyGuards(t *testing.T) {
	for _, kind := range []string{"excluded root", "excluded ancestor", "mate beta", "mate alpha", "pawn only"} {
		t.Run(kind, func(t *testing.T) {
			p, info := wave2NullState(t)
			a, b := 99, 100
			switch kind {
			case "excluded root":
				info.ExcludedMove = adapterLegalMove(t, p, "b1b2")
				info.ExcludedPly = 2
			case "excluded ancestor":
				info.ExcludedMove = adapterLegalMove(t, p, "b1b2")
				info.ExcludedPly = 0
			case "mate beta":
				a, b = MATE_IN_MAX-1, MATE_IN_MAX
				info.evaluator.staticOracle = func(*Position) int { return MATE_IN_MAX + 10 }
			case "mate alpha":
				a, b = -MATE_IN_MAX, -MATE_IN_MAX+1
			case "pawn only":
				p = n3cPosition(t, "7k/8/8/3pP3/8/8/8/K7 w - d6 17 1")
				if err := info.evaluator.Reset(p); err != nil {
					t.Fatal(err)
				}
			}
			info.nullSearch = func(*Position, int, int, int, int, bool, bool, bool, *SearchInfo) int {
				t.Fatal("guard allowed null search")
				return 0
			}
			info.evaluator.transitionObserver = func(*Position, *workerEvaluator) { info.control.RequestStop() }
			info.maxNodes = 32
			alphaBetaPV(p, 8, 2, a, b, false, true, true, info)
			if info.NullVerifySearches != 0 {
				t.Fatal("guard allowed verification")
			}
		})
	}
}

func TestWave2NullVerifySparseActual(t *testing.T) {
	old := W2NullVerify
	W2NullVerify = 1
	t.Cleanup(func() { W2NullVerify = old })
	for _, backend := range []string{"HCE", "H768"} {
		t.Run(backend, func(t *testing.T) {
			p, info := newProbcutContractStateForFEN(t, "7k/8/8/8/8/8/8/KR6 w - - 17 1", 8)
			info.evaluator.staticOracle = nil // Real evaluation, production pruning and recursive search.
			if backend == "H768" {
				net, err := LoadDefaultNet()
				if err != nil {
					t.Fatal(err)
				}
				info.evaluator = ngnn1NewWorker(t, net, p)
			}
			info.maxNodes = 2000
			before := snapshotPVPosition(p)
			prior := info.history.GetLastMovePlayed()
			beta := info.evaluator.SearchSTM(p) - 100
			var accumulator []int32
			if info.evaluator.nnue != nil {
				accumulator = append([]int32(nil), info.evaluator.nnue.at(0)...)
			}
			score := alphaBetaPV(p, 8, 0, beta-1, beta, false, true, true, info)
			requireWave2NullRestored(t, p, info, before, prior)
			if info.evaluator.nnue != nil && !reflect.DeepEqual(accumulator, info.evaluator.nnue.at(0)) {
				t.Fatal("active evaluator accumulator changed")
			}
			if info.Stopped || info.NullVerifySearches == 0 || info.NullVerifyCutoffs == 0 || info.Nodes < 2 {
				t.Fatalf("sparse witness missing: score=%d nodes=%d verify=%d cutoff=%d stopped=%v", score, info.Nodes, info.NullVerifySearches, info.NullVerifyCutoffs, info.Stopped)
			}
			t.Logf("actual sparse %s score=%d beta=%d nodes=%d null=%d verification=%d verified=%d", backend, score, beta, info.Nodes, info.NullMoveTries, info.NullVerifySearches, info.NullVerifyCutoffs)
		})
	}
}

func TestWave2NullVerifyCancelDescendant(t *testing.T) {
	p, info := wave2NullState(t)
	net, err := LoadDefaultNet()
	if err != nil {
		t.Fatal(err)
	}
	info.evaluator = ngnn1NewWorker(t, net, p)
	info.evaluator.staticOracle = func(*Position) int { return 200 }
	before := snapshotPVPosition(p)
	prior := info.history.GetLastMovePlayed()
	accumulator := append([]int32(nil), info.evaluator.nnue.at(0)...)
	calls, transitions := 0, 0
	info.nullSearch = func(pos *Position, d, ply, a, b int, pv, canNull, cut bool, s *SearchInfo) int {
		calls++
		if calls == 1 {
			return -100
		}
		if calls != 2 || !s.nullVerification {
			t.Fatal("nested null attempt")
		}
		s.evaluator.transitionObserver = func(*Position, *workerEvaluator) {
			transitions++
			if transitions == 2 {
				s.control.RequestStop()
			}
		}
		return alphaBetaPV(pos, d, ply, a, b, pv, true, cut, s)
	}
	score := alphaBetaPV(p, 8, 2, 99, 100, false, true, true, info)
	requireWave2NullRestored(t, p, info, before, prior)
	if !info.Stopped || transitions != 2 || score != stoppedSearchScore || info.NullVerifySearches != 1 || info.NullVerifyCutoffs != 0 || info.NullMoveCutoffs != 0 || !reflect.DeepEqual(accumulator, info.evaluator.nnue.at(0)) {
		t.Fatal("descendant cancellation did not unwind")
	}
	t.Logf("H768 verification descendant cancellation: transitions=%d verify=%d cutoffs=%d", transitions, info.NullVerifySearches, info.NullMoveCutoffs)
}

func TestWave2NullVerifyOffIdentity(t *testing.T) {
	originalSearchOptions(t)
	old := W2NullVerify
	W2NullVerify = 0
	t.Cleanup(func() { W2NullVerify = old })
	data, err := os.ReadFile("../experiments/wave2-1104/null-verify/baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		FEN           string
		Depth         int
		Nodes, QNodes uint64
		Score         int
		Best          string
		PV            []string
		NullTries     uint64 `json:"null_tries"`
		NullCutoffs   uint64 `json:"null_cutoffs"`
		Net           string `json:"net_sha256"`
		Stopped       bool
	}
	var baseline []result
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline) != 4 {
		t.Fatal("expected four baseline fixtures")
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
		p, err := ParseFEN(want.FEN)
		if err != nil {
			t.Fatal(err)
		}
		info := e.SearchIterativeDeepening(p, 4, nil)
		pv := []string{}
		for _, m := range e.ExtractPVFromTT(p, info.BestMove, 4) {
			pv = append(pv, m.ToString())
		}
		got := result{want.FEN, info.Depth, info.Nodes, info.QNodes, info.BestScore, info.BestMove.ToString(), pv, info.NullMoveTries, info.NullMoveCutoffs, DefaultNetSHA256, info.Stopped}
		if !reflect.DeepEqual(got, want) || info.NullVerifySearches != 0 {
			t.Fatalf("off differs from starting head: got=%+v want=%+v", got, want)
		}
		t.Logf("H768 d4 best=%s score=%d nodes=%d PV=%v", got.Best, got.Score, got.Nodes, got.PV)
	}
	for _, p := range TunableSearchParams {
		if p.Name == "W2NullVerify" {
			if p.Ptr != &W2NullVerify || p.Def != acceptedOptionDefaults["W2NullVerify"] || p.Min != 0 || p.Max != 1 {
				t.Fatal("bad option metadata")
			}
			return
		}
	}
	t.Fatal("W2NullVerify UCI option missing")
}
