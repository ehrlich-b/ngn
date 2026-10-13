package engine

import (
	"reflect"
	"testing"
)

func consensusInfo(t *testing.T, p *Position, text string, depth, score int) *SearchInfo {
	t.Helper()
	m := EmptyMove
	if text != "" {
		var err error
		if text == "e2e5" {
			m = NewMove(E2, E5, WhitePawn, NoPiece, NoType, 0)
		} else {
			m, err = ParseAlgebraicMove(text, p)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return &SearchInfo{BestMove: m, BestScore: score, Depth: depth, SelDepth: depth + 2}
}
func TestSMPConsensusCompletedGroups(t *testing.T) {
	p := controlTestPosition(t)
	type vote struct {
		move         string
		depth, score int
	}
	for _, tc := range []struct {
		name    string
		primary vote
		helpers []vote
		want    int
	}{
		{"two-deeper-votes", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, 30}, {"d2d4", 8, 25}, {"g1f3", 7, 40}}, 2},
		{"same-depth-lowest-index", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}, {"g1f3", 7, 40}}, 1},
		{"primary-group-tie", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}, {"e2e4", 7, 40}}, 0},
		{"helper-groups-tie", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}, {"g1f3", 7, 40}, {"g1f3", 7, 35}}, 0},
		{"shallower-vote-excluded", vote{"e2e4", 6, 20}, []vote{{"d2d4", 5, 30}, {"d2d4", 7, 25}}, 0},
		{"partial-empty-result-excluded", vote{"e2e4", 6, 20}, []vote{{"", MaximumDepth, -INFINITY}, {"d2d4", 7, 25}}, 0},
		{"no-primary-completion", vote{"", MaximumDepth, -INFINITY}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}}, 0},
		{"illegal-votes-excluded", vote{"e2e4", 6, 20}, []vote{{"e2e5", 7, 30}, {"e2e5", 7, 25}}, 0},
		{"keep-primary-winning-mate", vote{"e2e4", 6, MATE_VALUE - 3}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}}, 0},
		{"do-not-select-losing-mate", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, -MATE_VALUE + 7}, {"d2d4", 7, -MATE_VALUE + 9}}, 0},
		{"escape-primary-losing-mate", vote{"e2e4", 6, -MATE_VALUE + 9}, []vote{{"d2d4", 7, 30}, {"d2d4", 7, 25}}, 1},
		{"unanimous-winning-mate", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, MATE_VALUE - 7}, {"d2d4", 8, MATE_VALUE - 5}}, 2},
		{"mixed-mate-category-holds", vote{"e2e4", 6, 20}, []vote{{"d2d4", 7, MATE_VALUE - 7}, {"d2d4", 8, 25}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			primary := consensusInfo(t, p, tc.primary.move, tc.primary.depth, tc.primary.score)
			before := *primary
			helpers := make([]*SearchInfo, len(tc.helpers))
			for i, v := range tc.helpers {
				helpers[i] = consensusInfo(t, p, v.move, v.depth, v.score)
			}
			want := primary
			if tc.want > 0 {
				want = helpers[tc.want-1]
			}
			if got := selectSMPConsensus(p, primary, helpers); got != want {
				t.Fatalf("got%p want%p", got, want)
			}
			if !reflect.DeepEqual(before, *primary) {
				t.Fatal("selection mutated primary")
			}
		})
	}
}
func TestSMPConsensusRebuildsJoinedReport(t *testing.T) {
	e := newSMPTestEngine(t, 3)
	p := controlTestPosition(t)
	before := snapshotPVPosition(p)
	voters := []*SearchInfo{consensusInfo(t, p, "e2e4", 4, 20), consensusInfo(t, p, "d2d4", 5, 32), consensusInfo(t, p, "d2d4", 6, 33)}
	e.smpHooks = &smpSearchHooks{workerFinished: func(i int, info *SearchInfo) {
		v := voters[i]
		info.BestMove = v.BestMove
		info.BestScore = v.BestScore
		info.Depth = v.Depth
		info.SelDepth = v.SelDepth
	}}
	var callbacks []SearchInfo
	result := e.SearchIterativeDeepeningWithCallback(p, 2, nil, func(info *SearchInfo) { callbacks = append(callbacks, *info) })
	if result.BestMove.ToString() != "d2d4" || result.BestScore != 33 || result.Depth != 6 || result.RootDepth != 6 {
		t.Fatalf("joined decision move=%s score=%d depth=%d", result.BestMove.ToString(), result.BestScore, result.Depth)
	}
	if result.EffectiveThreads != 3 || result.PVLength < 1 || result.PV[0] != result.BestMove {
		t.Fatal("joined width/PV mismatch")
	}
	if len(callbacks) == 0 || callbacks[len(callbacks)-1].BestMove != result.BestMove || callbacks[len(callbacks)-1].Nodes != result.Nodes {
		t.Fatal("final callback did not report joined result/counters")
	}
	if !reflect.DeepEqual(before, snapshotPVPosition(p)) {
		t.Fatal("consensus changed root history")
	}
}
