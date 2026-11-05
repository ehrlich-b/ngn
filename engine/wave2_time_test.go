package engine

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

type scoreTimeTrace struct {
	Name       string
	Scores     []*int
	Reset      bool
	PostReset  []*int `json:"post_reset"`
	Researched bool
	Factor     float64
	Share      float64
	HardMS     int               `json:"hard_ms"`
	RawProbes  []json.RawMessage `json:"probes"`
}

func scoreTimeManager(clock *terminationObserverClock, enabled bool) *TimeManager {
	tm := newTimeManager(clock.Now)
	tm.scoreTime.enabled = enabled
	tm.SetTimeControl(SearchParams{WhiteTime: 10150, BlackTime: 10150, MovesToGo: 10}, true)
	return tm // soft=1s, hard=4s, with the ordinary 100ms reserve and 50ms overhead.
}

func TestWave2ScoreTimeTraces(t *testing.T) {
	data, err := os.ReadFile("../experiments/wave2-1104/score-time/traces.json")
	if err != nil {
		t.Fatal(err)
	}
	var traces []scoreTimeTrace
	if err := json.Unmarshal(data, &traces); err != nil {
		t.Fatal(err)
	}
	for _, trace := range traces {
		t.Run(trace.Name, func(t *testing.T) {
			clock := &terminationObserverClock{now: time.Unix(123, 0)}
			off, on, baseline := scoreTimeManager(clock, false), scoreTimeManager(clock, true), scoreTimeManager(clock, false)
			managers := []*TimeManager{off, on, baseline}
			feed := func(scores []*int) {
				for i, score := range scores {
					for _, tm := range managers {
						tm.NewIteration()
					}
					clock.Advance(100 * time.Millisecond)
					for _, tm := range managers {
						held := !trace.Researched || i != len(scores)-1
						if score == nil || tm == baseline {
							tm.ReportCompletedIteration(i == 0, held, trace.Share)
						} else {
							tm.ReportCompletedIteration(i == 0, held, trace.Share, *score)
						}
					}
				}
			}
			feed(trace.Scores)
			if trace.Reset {
				for _, tm := range managers {
					tm.SetTimeControlAt(SearchParams{WhiteTime: 10150, BlackTime: 10150, MovesToGo: 10}, true, clock.Now())
					if tm.scoreTime.previousKnown || tm.scoreTime.loss != 0 || tm.scoreTime.factor() != 1 {
						t.Fatal("new search reused a score")
					}
				}
				feed(trace.PostReset)
			}
			if math.Abs(on.scoreTime.factor()-trace.Factor) > 1e-12 || off.scoreTime.factor() != 1 {
				t.Fatalf("factors off=%g on=%g want on=%g", off.scoreTime.factor(), on.scoreTime.factor(), trace.Factor)
			}
			for _, tm := range managers {
				tm.NewIteration() // Real iteration duration feeds the existing projection.
				if tm.lastIterationTime != 100*time.Millisecond {
					t.Fatalf("last iteration=%s", tm.lastIterationTime)
				}
				if trace.HardMS != 0 {
					tm.hardTime = time.Duration(trace.HardMS) * time.Millisecond
				}
			}
			hard, soft := on.hardTime, on.softTime
			for _, raw := range trace.RawProbes {
				var tuple [3]json.RawMessage
				if err := json.Unmarshal(raw, &tuple); err != nil {
					t.Fatal(err)
				}
				var elapsedMS int
				var wantOff, wantOn bool
				for i, dest := range []any{&elapsedMS, &wantOff, &wantOn} {
					if err := json.Unmarshal(tuple[i], dest); err != nil {
						t.Fatal(err)
					}
				}
				clock.now = on.startTime.Add(time.Duration(elapsedMS) * time.Millisecond)
				stops := [3]bool{}
				for i, tm := range managers {
					// Each probe is an independent admission decision, bypassing only
					// the existing 1024-call rate limiter and cached previous stop.
					tm.shouldStop, tm.checkCounter = false, 1023
					stops[i] = tm.shouldStopSearchAt(5, timeCheckIterationAdmission)
				}
				if stops != [3]bool{wantOff, wantOn, wantOff} {
					t.Fatalf("at %dms off/on/legacy=%v want=%v", elapsedMS, stops, [3]bool{wantOff, wantOn, wantOff})
				}
				if on.hardTime != hard || on.softTime != soft || off.hardTime != hard {
					t.Fatal("score trend changed base budgets")
				}
				if time.Duration(elapsedMS)*time.Millisecond >= hard && on.tournamentStopReason(0, time.Duration(elapsedMS)*time.Millisecond) != searchTerminationHardDeadline {
					t.Fatal("hard deadline was extended")
				}
				t.Logf("TRACE %s elapsed_ms=%d off_stop=%t on_stop=%t legacy_stop=%t factor=%.6f hard_ms=%d", trace.Name, elapsedMS, stops[0], stops[1], stops[2], on.scoreTime.factor(), hard.Milliseconds())
			}
		})
	}
}

func TestWave2ScoreTimeFormula(t *testing.T) {
	for _, tc := range []struct {
		loss int
		want float64
	}{{-100, 1}, {0, 1}, {1, 1.005}, {25, 1.125}, {49, 1.245}, {50, 1.25}, {100, 1.25}, {57998, 1.25}} {
		trend := wave2ScoreTime{enabled: true}
		trend.report([]int{28999})
		trend.report([]int{28999 - tc.loss})
		// The improving case crosses into mate range and is also neutral.
		if math.Abs(trend.factor()-tc.want) > 1e-12 {
			t.Errorf("loss=%d factor=%g want=%g", tc.loss, trend.factor(), tc.want)
		}
	}
}

func TestWave2ScoreTimeCompletedCallbackAndSearchReset(t *testing.T) {
	saveInit, saveMult := ASP_INIT, ASP_MULT
	ASP_INIT, ASP_MULT = 1, 110
	defer func() { ASP_INIT, ASP_MULT = saveInit, saveMult }()
	clock := &terminationObserverClock{now: time.Unix(123, 0)}
	tm := scoreTimeManager(clock, true)
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	observedResearch := false
	for run := 0; run < 2; run++ {
		// Seed a stale decline; the real search entry must clear it even when
		// the API caller reuses this manager without SetTimeControl.
		tm.ReportCompletedIteration(false, true, 0.625, 500)
		tm.ReportCompletedIteration(false, true, 0.625, -500)
		var observer searchTerminationObserver
		tm.setSearchTerminationObserver(&observer)
		pos, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
		if err != nil {
			t.Fatal(err)
		}
		var previous, count int
		info := searcher.SearchIterativeDeepeningWithCallback(pos, 5, tm, func(info *SearchInfo) {
			count++
			want := 1.0
			if count > 1 && info.BestScore < previous {
				want += math.Min(float64(previous-info.BestScore)/200, 0.25)
			}
			if !tm.scoreTime.previousKnown || tm.scoreTime.previousScore != info.BestScore || math.Abs(tm.scoreTime.factor()-want) > 1e-12 {
				t.Fatalf("completed callback depth=%d score=%d state=%+v want factor=%g", info.Depth, info.BestScore, tm.scoreTime, want)
			}
			attempts := observer.Iterations[count-1].AspirationAttempts
			observedResearch = observedResearch || attempts > 1
			t.Logf("CALLBACK run=%d depth=%d attempts=%d score=%d previous=%d factor=%.6f", run, info.Depth, attempts, info.BestScore, previous, tm.scoreTime.factor())
			previous = info.BestScore
			clock.Advance(10 * time.Millisecond)
		})
		if count != 5 || info.Depth != 5 || info.Stopped {
			t.Fatalf("completed iterations=%d depth=%d stopped=%t", count, info.Depth, info.Stopped)
		}
	}
	if !observedResearch {
		t.Fatal("fixture did not complete an aspiration re-search")
	}
}

func TestWave2ScoreTimeAspirationFailureBeforeCompletion(t *testing.T) {
	saveInit, saveMult := ASP_INIT, ASP_MULT
	ASP_INIT, ASP_MULT = 1, 110
	defer func() { ASP_INIT, ASP_MULT = saveInit, saveMult }()
	clock := &terminationObserverClock{now: time.Unix(123, 0)}
	tm := scoreTimeManager(clock, true)
	var observer searchTerminationObserver
	tm.setSearchTerminationObserver(&observer)
	var live *SearchInfo
	var completed wave2ScoreTime
	completedDepth := 0
	var completedFailures uint64
	interrupted := false
	tm.now = func() time.Time {
		// Observe every existing time-check site so even a tiny failed attempt
		// cannot finish its re-search between rate-limited clock samples.
		tm.checkCounter = 1023
		if live != nil && live.RootDepth > completedDepth && live.AspFailLows+live.AspFailHighs > completedFailures {
			if tm.scoreTime != completed {
				t.Fatalf("aspiration failure fed an incomplete score: got=%+v want=%+v", tm.scoreTime, completed)
			}
			interrupted = true
			return tm.startTime.Add(tm.hardTime)
		}
		return clock.Now()
	}
	pos, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	info := searcher.SearchIterativeDeepeningWithCallback(pos, 6, tm, func(info *SearchInfo) {
		count++
		live, completed = info, tm.scoreTime
		completedDepth = info.Depth
		completedFailures = info.AspFailLows + info.AspFailHighs
		clock.Advance(10 * time.Millisecond)
	})
	if !interrupted || !info.Stopped || info.Depth < 3 || info.Depth >= 6 || count != info.Depth || observer.IterationCount != count || tm.scoreTime != completed {
		t.Fatalf("incomplete aspiration: interrupted=%t stopped=%t depth=%d callbacks=%d observations=%d", interrupted, info.Stopped, info.Depth, count, observer.IterationCount)
	}
	if observer.FirstReason != searchTerminationHardDeadline {
		t.Fatalf("stop reason=%s", observer.FirstReason)
	}
	t.Logf("ASPIRATION completed_depth=%d failed_attempts=%d retained_score=%d reason=%s", info.Depth, info.AspFailLows+info.AspFailHighs, completed.previousScore, observer.FirstReason)
}

func TestWave2ScoreTimeFixedNodes(t *testing.T) {
	var results [2]*SearchInfo
	for i, enabled := range []bool{false, true} {
		clock := &terminationObserverClock{now: time.Unix(123, 0)}
		tm := scoreTimeManager(clock, enabled)
		params := parseSearchParams([]string{"nodes", "1000"})
		tm.SetTimeControl(params, true)
		searcher, err := NewSearchEngineWithHash(1)
		if err != nil {
			t.Fatal(err)
		}
		searcher.SetMaxNodes(params.Nodes)
		results[i] = searcher.SearchIterativeDeepeningWithCallback(controlTestPosition(t), MaximumDepth, tm, func(*SearchInfo) {
			clock.Advance(10 * time.Second) // Far beyond any tournament hard budget.
		})
		if tm.timeControl != Infinite || !results[i].Stopped || results[i].Nodes != 1000 {
			t.Fatalf("fixed-node result enabled=%t control=%v nodes=%d stopped=%t", enabled, tm.timeControl, results[i].Nodes, results[i].Stopped)
		}
	}
	a, b := results[0], results[1]
	if a.Nodes != b.Nodes || a.Depth != b.Depth || a.BestMove != b.BestMove || a.BestScore != b.BestScore {
		t.Fatal("score timing affected fixed-node search")
	}
	t.Logf("FIXED_NODES off/on nodes=%d depth=%d move=%v score=%d", a.Nodes, a.Depth, a.BestMove, a.BestScore)
}

func TestWave2ScoreTimeUCIOption(t *testing.T) {
	uci := NewUCIEngine()
	var output bytes.Buffer
	uci.handleUCI(&output)
	if uci.timeManager.scoreTime.enabled || !strings.Contains(output.String(), "option name W2ScoreTime type spin default 0 min 0 max 1") {
		t.Fatal("W2ScoreTime must be advertised default off")
	}
	if isTunableParam("W2ScoreTime") || isTunableParam("ScoreTimeScale") || isTunableParam("ScoreTimeMax") {
		t.Fatal("score timing entered the SPSA registry")
	}
	uci.handleSetOption([]string{"name", "w2scoretime", "value", "1"}, io.Discard)
	uci.handleNewGame(io.Discard)
	if !uci.timeManager.scoreTime.enabled || uci.timeManager.scoreTime.previousKnown {
		t.Fatal("ucinewgame must preserve the option and reset the scores")
	}
	for _, invalid := range []string{"-1", "2", "true", "oops"} {
		uci.handleSetOption([]string{"name", "W2ScoreTime", "value", invalid}, io.Discard)
		if !uci.timeManager.scoreTime.enabled {
			t.Fatalf("invalid value %s changed option", invalid)
		}
	}
	uci.handleSetOption([]string{"name", "W2ScoreTime", "value", "0"}, io.Discard)
	if uci.timeManager.scoreTime.enabled {
		t.Fatal("option could not be turned off")
	}
}
