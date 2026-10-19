package engine

import (
	"reflect"
	"testing"
)

func qstaticFixture(t *testing.T, fen string) (*Position, *SearchInfo) {
	t.Helper()
	position, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	info := newQSearchInfo()
	info.control = &SearchControl{}
	info.history = &workerHistory{}
	info.tt = NewCache(1)
	return position, info
}

func TestQStaticCompatibleBoundChangesQuietHorizon(t *testing.T) {
	for _, fen := range []string{
		"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
		"4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
	} {
		for _, bound := range []NodeType{LowerBound, UpperBound} {
			position, info := qstaticFixture(t, fen)
			raw := info.history.correctedStandPat(position, info.evaluator)
			want := raw + 80
			if bound == UpperBound {
				want = raw - 80
			}
			info.tt.Set(position.Hash(), EmptyMove, scoreToTT(want, 3), 8, bound, false)
			before := snapshotPVPosition(position)
			got := quiescence(position, -INFINITY, INFINITY, 3, info)
			if got != want || info.QTTCutoffs != 0 {
				t.Fatalf("quiet horizon ignored compatible bound: turn=%v bound=%v got=%d want=%d direct cutoffs=%d", position.Turn(), bound, got, want, info.QTTCutoffs)
			}
			if !reflect.DeepEqual(snapshotPVPosition(position), before) {
				t.Fatal("qsearch changed root")
			}
		}
	}
}

func TestQStaticIgnoresOppositeDirectionAndMateBounds(t *testing.T) {
	for _, bound := range []NodeType{LowerBound, UpperBound} {
		for _, mate := range []bool{false, true} {
			position, info := qstaticFixture(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
			raw := info.history.correctedStandPat(position, info.evaluator)
			value := raw - 80
			if bound == UpperBound {
				value = raw + 80
			}
			if mate {
				value = MATE_VALUE - 20
				if bound == UpperBound {
					value = -value
				}
			}
			info.tt.Set(position.Hash(), EmptyMove, scoreToTT(value, 3), 8, bound, false)
			got := quiescence(position, -INFINITY, INFINITY, 3, info)
			if got != raw {
				t.Fatalf("bound=%v mate=%v changed static: got=%d want=%d", bound, mate, got, raw)
			}
		}
	}
}

func TestQStaticUsesCorrectedStandPatAsStartingEstimate(t *testing.T) {
	position, info := qstaticFixture(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
	idx := pawnCorrectionIndex(position.Board.pieces[WhitePawn], position.Board.pieces[BlackPawn])
	info.history.pawnCorrectionHistory[White][idx] = corrHistLimit
	corrected := info.history.correctedStandPat(position, info.evaluator)
	if corrected <= info.evaluator.SearchSTM(position) {
		t.Fatal("fixture lacks positive correction")
	}
	// A lower bound between the raw and corrected eval cannot lower the estimate.
	value := corrected - 1
	info.tt.Set(position.Hash(), EmptyMove, scoreToTT(value, 0), 8, LowerBound, false)
	if got := quiescence(position, -INFINITY, INFINITY, 0, info); got != corrected {
		t.Fatalf("lower bound erased correction: got=%d want=%d", got, corrected)
	}
}

func TestQStaticCheckDrawAndStopPrecedeRefinement(t *testing.T) {
	const checked = "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1"
	base, baseInfo := qstaticFixture(t, checked)
	want := quiescence(base, -INFINITY, INFINITY, 0, baseInfo)
	position, info := qstaticFixture(t, checked)
	info.tt.Set(position.Hash(), EmptyMove, -200, 8, UpperBound, false)
	if got := quiescence(position, -INFINITY, INFINITY, 0, info); got != want {
		t.Fatalf("check node acquired stand-pat: got=%d want=%d", got, want)
	}
	position, info = qstaticFixture(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 100 1")
	info.tt.Set(position.Hash(), EmptyMove, 200, 8, LowerBound, false)
	if got := quiescence(position, -INFINITY, INFINITY, 0, info); got != 0 || info.QTTProbes != 0 {
		t.Fatalf("draw did not precede TT refinement: got=%d probes=%d", got, info.QTTProbes)
	}
	position, info = qstaticFixture(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
	info.tt.Set(position.Hash(), EmptyMove, 200, 8, LowerBound, false)
	before := append([]ttRecord(nil), info.tt.items...)
	info.control.RequestStop()
	if got := quiescence(position, -INFINITY, INFINITY, 0, info); got != stoppedSearchScore || !info.Stopped || info.QTTProbes != 0 {
		t.Fatalf("stop did not precede TT refinement: got=%d stopped=%v probes=%d", got, info.Stopped, info.QTTProbes)
	}
	if !reflect.DeepEqual(before, info.tt.items) {
		t.Fatal("stopped search changed TT")
	}
}
