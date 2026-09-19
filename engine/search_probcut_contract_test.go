package engine

import (
	"reflect"
	"testing"

	"github.com/ehrlich-b/ngn/nnue"
)

const probcutContractFEN = "4k3/p7/8/8/8/8/8/R3K3 w - - 0 1"

func newProbcutContractState(t *testing.T, depth int) (*Position, *SearchInfo) {
	t.Helper()
	model := loadEvaluatorTensors(t, new(nnue.Tensors))
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := searcher.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	pos := n3cPosition(t, probcutContractFEN)
	searcher.sessionMu.Lock()
	generation := mustAcquireHCEModelUse()
	t.Cleanup(func() {
		searcher.worker.control.ClearStop()
		releaseHCEModelUse()
		searcher.sessionMu.Unlock()
	})
	tt := searcher.prepareTTGeneration(generation)
	evaluator := searcher.mustPreparePrimaryEvaluator(pos, generation)
	return pos, newN3CDirectSearchInfo(searcher, evaluator, tt, depth)
}

func probcutChild(t *testing.T, pos *Position, move Move) *Position {
	t.Helper()
	child := pos.Copy()
	child.MakeMove(move)
	return child
}

func requireProbcutSearchUnwound(t *testing.T, pos *Position, before pvPositionSnapshot, info *SearchInfo, wantLast Move, wantRepLen, wantFrameDepth int) {
	t.Helper()
	if after := snapshotPVPosition(pos); !reflect.DeepEqual(after, before) {
		t.Fatalf("search changed root position\nbefore: %+v\nafter:  %+v", before, after)
	}
	if got := info.history.GetLastMovePlayed(); got != wantLast {
		t.Fatalf("last move after search = %s, want %s", got.ToString(), wantLast.ToString())
	}
	if info.RepStackLen != wantRepLen || info.FrameDepth != wantFrameDepth {
		t.Fatalf("search stack lengths after search = repetition %d, frames %d; want %d, %d", info.RepStackLen, info.FrameDepth, wantRepLen, wantFrameDepth)
	}
	if got := info.evaluator.nnueDepth(); got != 0 {
		t.Fatalf("search left evaluator depth %d, want 0", got)
	}
}

func TestSingularVerificationSkipsProbcutAtExcludedPly(t *testing.T) {
	pos, info := newProbcutContractState(t, 5)
	before := snapshotPVPosition(pos)
	excluded := adapterLegalMove(t, pos, "a1a7")
	predecessor := adapterLegalMove(t, pos, "e1d1")
	info.ExcludedMove, info.ExcludedPly = excluded, 0
	info.history.SetLastMovePlayed(predecessor)

	// Preserve a normal-position entry while the restricted search runs. The
	// exact child entry makes the old ProbCut path deterministic.
	const rootEval int16 = 37
	info.tt.Set(pos.Hash(), excluded, rootEval, 5, Exact, true)
	child := probcutChild(t, pos, excluded)
	info.tt.Set(child.Hash(), EmptyMove, scoreToTT(0, 1), 1, Exact, false)
	visitedExcluded := false
	info.evaluator.transitionObserver = func(observed *Position, evaluator *workerEvaluator) {
		if evaluator.nnueDepth() == 1 && observed.Hash() == child.Hash() {
			visitedExcluded = true
		}
	}

	got := alphaBetaPV(pos, 5, 0, -201, -200, false, false, false, info)
	if got == stoppedSearchScore || info.Stopped {
		t.Fatalf("singular verification stopped: score=%d stopped=%v", got, info.Stopped)
	}
	if visitedExcluded {
		t.Fatalf("singular verification traversed excluded move %s through ProbCut", excluded.ToString())
	}
	if info.ProbcutPrunes != 0 {
		t.Fatalf("singular root credited %d ProbCut prunes, want 0", info.ProbcutPrunes)
	}
	move, eval, depth, nodeType, hit, ttPV := info.tt.Get(pos.Hash())
	if !hit || move != excluded || eval != rootEval || depth != 5 || nodeType != Exact || !ttPV {
		t.Fatalf("restricted search changed ordinary TT entry: hit=%v move=%s eval=%d depth=%d type=%v pv=%v", hit, move.ToString(), eval, depth, nodeType, ttPV)
	}
	requireProbcutSearchUnwound(t, pos, before, info, predecessor, 0, 0)
}

func TestSingularVerificationDisablesWholeRootProbcut(t *testing.T) {
	pos, info := newProbcutContractState(t, 5)
	before := snapshotPVPosition(pos)
	excludedQuiet := adapterLegalMove(t, pos, "e1d1")
	capture := adapterLegalMove(t, pos, "a1a7")
	info.ExcludedMove, info.ExcludedPly = excludedQuiet, 0
	child := probcutChild(t, pos, capture)
	info.tt.Set(child.Hash(), EmptyMove, scoreToTT(0, 1), 1, Exact, false)

	got := alphaBetaPV(pos, 5, 0, -201, -200, false, false, false, info)
	if got == stoppedSearchScore || info.Stopped {
		t.Fatalf("quiet-exclusion verification stopped: score=%d stopped=%v", got, info.Stopped)
	}
	if info.ProbcutPrunes != 0 {
		t.Fatalf("quiet-exclusion root credited %d ProbCut prunes, want 0", info.ProbcutPrunes)
	}
	requireProbcutSearchUnwound(t, pos, before, info, EmptyMove, 0, 0)
}

func TestProbcutRemainsEnabledOutsideExcludedPly(t *testing.T) {
	for _, test := range []struct {
		name        string
		ply         int
		excludedPly int
	}{
		{name: "ordinary search", ply: 0, excludedPly: -1},
		{name: "descendant of excluded ply", ply: 1, excludedPly: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			pos, info := newProbcutContractState(t, 5)
			before := snapshotPVPosition(pos)
			capture := adapterLegalMove(t, pos, "a1a7")
			if test.excludedPly >= 0 {
				info.ExcludedMove = adapterLegalMove(t, pos, "e1d1")
				info.ExcludedPly = test.excludedPly
			}
			child := probcutChild(t, pos, capture)
			info.tt.Set(child.Hash(), EmptyMove, scoreToTT(0, test.ply+1), 1, Exact, false)
			visitedCapture := false
			info.evaluator.transitionObserver = func(observed *Position, evaluator *workerEvaluator) {
				if evaluator.nnueDepth() == 1 && observed.Hash() == child.Hash() {
					visitedCapture = true
				}
			}

			got := alphaBetaPV(pos, 5, test.ply, -201, -200, false, false, false, info)
			if got != -200 || info.Stopped || !visitedCapture || info.ProbcutPrunes != 1 {
				t.Fatalf("ProbCut control score=%d stopped=%v capture=%v prunes=%d", got, info.Stopped, visitedCapture, info.ProbcutPrunes)
			}
			requireProbcutSearchUnwound(t, pos, before, info, EmptyMove, 0, 0)
		})
	}
}

func TestStoppedSingularVerificationSkipsProbcutAndUnwinds(t *testing.T) {
	pos, info := newProbcutContractState(t, 5)
	before := snapshotPVPosition(pos)
	excluded := adapterLegalMove(t, pos, "a1a7")
	ancestor := adapterLegalMove(t, pos, "e1d1")
	info.ExcludedMove, info.ExcludedPly = excluded, 2
	info.MoveStack[0] = ancestor
	info.history.SetLastMovePlayed(ancestor)
	info.RepStack[0], info.RepStack[1], info.RepStackLen = 0x1234, 0x5678, 2
	info.FrameDepth = 2
	child := probcutChild(t, pos, excluded)
	info.tt.Set(child.Hash(), EmptyMove, scoreToTT(0, 3), 1, Exact, false)
	visitedExcluded := false
	requestedStop := false
	info.evaluator.transitionObserver = func(observed *Position, evaluator *workerEvaluator) {
		if evaluator.nnueDepth() != 1 {
			return
		}
		if observed.Hash() == child.Hash() {
			visitedExcluded = true
		}
		if !requestedStop {
			requestedStop = true
			info.control.RequestStop()
		}
	}

	got := alphaBetaPV(pos, 5, 2, -201, -200, false, false, false, info)
	if !requestedStop || !info.Stopped || got != stoppedSearchScore {
		t.Fatalf("forced stop result score=%d stopped=%v requested=%v", got, info.Stopped, requestedStop)
	}
	if visitedExcluded {
		t.Fatalf("stopped singular verification traversed excluded move %s through ProbCut", excluded.ToString())
	}
	if info.MoveStack[0] != ancestor || info.RepStack[0] != 0x1234 || info.RepStack[1] != 0x5678 {
		t.Fatal("forced stop changed live ancestor search state")
	}
	requireProbcutSearchUnwound(t, pos, before, info, ancestor, 2, 2)
}
