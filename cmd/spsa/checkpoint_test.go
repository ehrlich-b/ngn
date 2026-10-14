package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ehrlich-b/ngn/engine"
)

func testParams(iters int) []*pvar {
	var params []*pvar
	for _, tp := range engine.TunableSearchParams {
		p := &pvar{name: tp.Name, theta: float64(tp.Def), min: float64(tp.Min), max: float64(tp.Max)}
		p.cEnd = math.Max(.5, (p.max-p.min)/20)
		p.cAmp = p.cEnd * math.Pow(float64(iters), spsaGamma)
		p.aGain = .05 * p.cEnd * p.cEnd * math.Pow(1.1*float64(iters), spsaAlpha)
		params = append(params, p)
	}
	return params
}

func testRecipe(params []*pvar, iters, pairs, openings, report int, seed int64) campaignRecipe {
	r := campaignRecipe{EngineSHA: strings.Repeat("1", 64), TunerSHA: strings.Repeat("2", 64),
		OpeningsSHA: strings.Repeat("3", 64), Openings: openings, GoVersion: "test", GOOS: "linux", GOARCH: "amd64",
		GOMAXPROCS: 1, RNG: "math/rand.NewSource/Int63-count-v1", Seed: seed, Iterations: iters,
		Pairs: pairs, TimeMs: 10000, IncMs: 100, MaxMoves: 200, REnd: .05, AFrac: .1, Report: report}
	for _, p := range params {
		r.Params = append(r.Params, parameterRecipe{p.name, defaultOf(p.name), p.min, p.max, p.cEnd, p.cAmp, p.aGain})
	}
	return r
}

// Compare every perturbation/opening and the exact float theta trajectory, not
// just the final rounded vector. Include Intn's rejection-sampling branch.
func TestCampaignResumeTrajectory(t *testing.T) {
	for _, pairs := range []int{1, 6} {
		for _, openings := range []int{0, 5000, 1<<30 + 1} {
			t.Run(fmt.Sprintf("pairs%d/openings%d", pairs, openings), func(t *testing.T) {
				const iters, split, report = 250, 75, 25
				continuous := testParams(iters)
				r := testRecipe(continuous, iters, pairs, openings, report, 20261008)
				objective := func(trace *[][]byte) func([][]int, [][]int, []int) []float64 {
					return func(plus, minus [][]int, op []int) []float64 {
						b, _ := json.Marshal([]any{plus, minus, op})
						*trace = append(*trace, b)
						scores := make([]float64, pairs)
						for j := range scores {
							v := float64(op[j]%7-3) / 10
							for i := range plus[j] {
								v += float64(minus[j][i]-plus[j][i]) / float64(i+10)
							}
							scores[j] = math.Tanh(v)
						}
						return scores
					}
				}
				var fullTrace, resumedTrace [][]byte
				var fullTheta, resumedTheta [][]float64
				snapshot := func(params []*pvar, dst *[][]float64) {
					var vals []float64
					for _, p := range params {
						vals = append(vals, p.theta)
					}
					*dst = append(*dst, vals)
				}
				fullRNG, fullSource := newCampaignRNG(r.Seed)
				if err := runCampaign(continuous, 1, iters, .1*iters, fullRNG, report, pairs, openings,
					objective(&fullTrace), func(int) error { snapshot(continuous, &fullTheta); return nil }); err != nil {
					t.Fatal(err)
				}
				params := testParams(iters)
				path := filepath.Join(t.TempDir(), "history.jsonl")
				if err := prepareHistory(path); err != nil {
					t.Fatal(err)
				}
				rng, source := newCampaignRNG(r.Seed)
				onReport := func(k int) error {
					snapshot(params, &resumedTheta)
					return writeCheckpoint(path, k, params, 50, r, source.draws)
				}
				// The schedule remains 250 iterations; only execution is interrupted.
				if err := runCampaign(params, 1, split, .1*iters, rng, report, pairs, openings, objective(&resumedTrace), onReport); err != nil {
					t.Fatal(err)
				}
				params = testParams(iters)
				saved, restoredRNG, restoredSource, err := loadCheckpoint(path, params, r)
				if err != nil || saved != split {
					t.Fatalf("load: iter=%d err=%v", saved, err)
				}
				rng, source = restoredRNG, restoredSource
				if err := runCampaign(params, saved+1, iters, .1*iters, rng, report, pairs, openings, objective(&resumedTrace), onReport); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(fullTrace, resumedTrace) || !reflect.DeepEqual(fullTheta, resumedTheta) || !reflect.DeepEqual(continuous, params) || fullSource.draws != source.draws {
					t.Fatal("interrupted campaign differs in draws, inputs or theta")
				}
				if _, _, _, err := loadCheckpoint(path, testParams(iters), r); err != nil {
					t.Fatalf("complete history: %v", err)
				}
				if openings == 1<<30+1 && source.draws <= uint64(iters*pairs*(len(params)+1)) {
					t.Fatal("rejection sampling was not exercised")
				}
			})
		}
	}
}

func TestCampaignRNGPreservesOriginalStream(t *testing.T) {
	for _, seed := range []int64{-1, 0, 20261008} {
		old := rand.New(rand.NewSource(seed))
		rng, _ := newCampaignRNG(seed)
		for i := 0; i < 10000; i++ {
			for _, bound := range []int{2, 5000, 1<<30 + 1} {
				if old.Intn(bound) != rng.Intn(bound) {
					t.Fatal("new RNG changed the fresh seeded stream")
				}
			}
		}
	}
}

func checkpointFixture(t *testing.T) (checkpoint, campaignRecipe) {
	t.Helper()
	params := testParams(100)
	r := testRecipe(params, 100, 6, 5000, 25, 20261008)
	rng, source := newCampaignRNG(r.Seed)
	for k := 0; k < 25; k++ {
		for j := 0; j < r.Pairs; j++ {
			for range params {
				rng.Intn(2)
			}
			rng.Intn(r.Openings)
		}
	}
	path := filepath.Join(t.TempDir(), "history.jsonl")
	if err := writeCheckpoint(path, 25, params, 50, r, source.draws); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cp checkpoint
	if err := json.Unmarshal(b, &cp); err != nil {
		t.Fatal(err)
	}
	return cp, r
}

func TestCheckpointRejectsChangedRecipe(t *testing.T) {
	changes := map[string]func(*campaignRecipe){
		"engine":         func(r *campaignRecipe) { r.EngineSHA = "changed" },
		"tuner":          func(r *campaignRecipe) { r.TunerSHA = "changed" },
		"openings-order": func(r *campaignRecipe) { r.OpeningsSHA = "changed" },
		"openings-count": func(r *campaignRecipe) { r.Openings++ },
		"Go":             func(r *campaignRecipe) { r.GoVersion = "changed" },
		"OS":             func(r *campaignRecipe) { r.GOOS = "changed" },
		"arch":           func(r *campaignRecipe) { r.GOARCH = "changed" },
		"GOMAXPROCS":     func(r *campaignRecipe) { r.GOMAXPROCS++ },
		"rng":            func(r *campaignRecipe) { r.RNG = "changed" },
		"seed":           func(r *campaignRecipe) { r.Seed++ },
		"iterations":     func(r *campaignRecipe) { r.Iterations++ },
		"pairs":          func(r *campaignRecipe) { r.Pairs++ },
		"single":         func(r *campaignRecipe) { r.Single = true },
		"tc":             func(r *campaignRecipe) { r.TimeMs++ },
		"inc":            func(r *campaignRecipe) { r.IncMs++ },
		"cap":            func(r *campaignRecipe) { r.MaxMoves++ },
		"rend":           func(r *campaignRecipe) { r.REnd += .01 },
		"afrac":          func(r *campaignRecipe) { r.AFrac += .01 },
		"report":         func(r *campaignRecipe) { r.Report++ },
		"power":          func(r *campaignRecipe) { r.LowPower = true },
		"name":           func(r *campaignRecipe) { r.Params[0].Name = "changed" },
		"default":        func(r *campaignRecipe) { r.Params[0].Default++ },
		"min":            func(r *campaignRecipe) { r.Params[0].Min-- },
		"max":            func(r *campaignRecipe) { r.Params[0].Max++ },
		"cEnd":           func(r *campaignRecipe) { r.Params[0].CEnd++ },
		"cAmp":           func(r *campaignRecipe) { r.Params[0].CAmp++ },
		"aGain":          func(r *campaignRecipe) { r.Params[0].AGain++ },
		"registry-order": func(r *campaignRecipe) { r.Params[0], r.Params[1] = r.Params[1], r.Params[0] },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			cp, recipe := checkpointFixture(t)
			b, _ := json.Marshal(cp)
			path := filepath.Join(t.TempDir(), "history.jsonl")
			if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}
			change(&recipe)
			params := testParams(100)
			before := testParams(100)
			if _, _, _, err := loadCheckpoint(path, params, recipe); err == nil {
				t.Fatal("changed recipe accepted")
			}
			if !reflect.DeepEqual(params, before) {
				t.Fatal("refused resume changed theta")
			}
		})
	}
}

func TestCheckpointRejectsCorruptHistory(t *testing.T) {
	mutations := map[string]func(*checkpoint){
		"legacy":            func(c *checkpoint) { c.Version = 0 },
		"future-schema":     func(c *checkpoint) { c.Version++ },
		"zero-iter":         func(c *checkpoint) { c.Iter = 0 },
		"negative-iter":     func(c *checkpoint) { c.Iter = -1 },
		"ahead-iter":        func(c *checkpoint) { c.Iter = 101 },
		"off-cadence":       func(c *checkpoint) { c.Iter = 24 },
		"time":              func(c *checkpoint) { c.Time = "invalid" },
		"health":            func(c *checkpoint) { c.PlusPct = 101 },
		"missing-theta":     func(c *checkpoint) { delete(c.Theta, "NullMoveR") },
		"unknown-theta":     func(c *checkpoint) { c.Theta["unknown"] = 1 },
		"substituted-theta": func(c *checkpoint) { delete(c.Theta, "NullMoveR"); c.Theta["unknown"] = 3 },
		"out-of-range":      func(c *checkpoint) { c.Theta["NullMoveR"] = 999 },
		"missing-rounded":   func(c *checkpoint) { delete(c.Rounded, "NullMoveR") },
		"wrong-rounded":     func(c *checkpoint) { c.Rounded["NullMoveR"]++ },
		"rng":               func(c *checkpoint) { c.RNGDraws++ },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			cp, recipe := checkpointFixture(t)
			mutate(&cp)
			b, _ := json.Marshal(cp)
			assertBadHistory(t, append(b, '\n'), recipe)
		})
	}
	cp, recipe := checkpointFixture(t)
	b, _ := json.Marshal(cp)
	for name, data := range map[string][]byte{
		"empty": nil, "blank": []byte("\n"), "partial": b, "invalid": []byte("{\n"),
		"trailing-json":  append(append(append([]byte{}, b...), []byte(" {}")...), '\n'),
		"corrupt-tail":   append(append(append([]byte{}, b...), '\n'), []byte("{\n")...),
		"corrupt-middle": append([]byte("{\n"), append(b, '\n')...),
		"duplicate-iter": append(append(append([]byte{}, b...), '\n'), append(b, '\n')...),
		"nonfinite":      []byte(strings.Replace(string(b), fmt.Sprintf(`"NullMoveR":%g`, cp.Theta["NullMoveR"]), `"NullMoveR":1e999`, 1) + "\n"),
	} {
		t.Run(name, func(t *testing.T) { assertBadHistory(t, data, recipe) })
	}
	if _, _, _, err := loadCheckpoint(filepath.Join(t.TempDir(), "missing"), testParams(100), recipe); err == nil {
		t.Fatal("missing history accepted")
	}
}

func assertBadHistory(t *testing.T, data []byte, recipe campaignRecipe) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.jsonl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	params, before := testParams(100), testParams(100)
	if _, _, _, err := loadCheckpoint(path, params, recipe); err == nil {
		t.Fatal("bad history accepted")
	}
	if !reflect.DeepEqual(params, before) {
		t.Fatal("failed load changed theta")
	}
}

func TestHistoryAndCheckpointWriteErrors(t *testing.T) {
	cp, recipe := checkpointFixture(t)
	path := filepath.Join(t.TempDir(), "history.jsonl")
	if err := prepareHistory(""); err == nil {
		t.Fatal("empty path accepted")
	}
	if err := prepareHistory(path); err != nil {
		t.Fatal(err)
	}
	if err := writeCheckpoint(path, cp.Iter, testParams(100), 50, recipe, cp.RNGDraws); err != nil {
		t.Fatal(err)
	}
	if err := prepareHistory(path); err == nil {
		t.Fatal("fresh campaign could mix with existing history")
	}
	if err := writeCheckpoint(t.TempDir(), cp.Iter, testParams(100), 50, recipe, cp.RNGDraws); err == nil {
		t.Fatal("write failure was swallowed")
	}
	if err := writeCheckpoint("", cp.Iter, testParams(100), 50, recipe, cp.RNGDraws); err == nil {
		t.Fatal("empty checkpoint path silently disabled persistence")
	}
	// On Linux /dev/full accepts open but rejects writes; not an open-only check.
	if _, err := os.Stat("/dev/full"); err == nil {
		if err := writeCheckpoint("/dev/full", cp.Iter, testParams(100), 50, recipe, cp.RNGDraws); err == nil {
			t.Fatal("write error was swallowed")
		}
	}
}

func TestCampaignStopsOnCheckpointFailure(t *testing.T) {
	params := testParams(100)
	rng, _ := newCampaignRNG(1)
	batches := 0
	err := runCampaign(params, 1, 100, 10, rng, 25, 1, 0,
		func(_, _ [][]int, _ []int) []float64 { batches++; return []float64{0} },
		func(int) error { return fmt.Errorf("disk failure") })
	if err == nil || batches != 25 {
		t.Fatalf("continued after checkpoint failure: batches=%d err=%v", batches, err)
	}
}

func TestRecipeBindsEffectiveOpeningOrderAndEngine(t *testing.T) {
	enginePath := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(enginePath, []byte("engine one"), 0o755); err != nil {
		t.Fatal(err)
	}
	params := testParams(100)
	makeR := func(openings [][]string) campaignRecipe {
		r, err := makeRecipe(enginePath, openings, params, 100, 6, false, 10000, 100, .05, .1, 25, 1, false)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	a := makeR([][]string{{"e2e4"}, {"d2d4"}})
	b := makeR([][]string{{"d2d4"}, {"e2e4"}})
	if a.OpeningsSHA == b.OpeningsSHA || a.TunerSHA == "" {
		t.Fatal("opening order or tuner bytes not bound")
	}
	if err := os.WriteFile(enginePath, []byte("engine two"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := makeR([][]string{{"e2e4"}, {"d2d4"}})
	if a.EngineSHA == c.EngineSHA || a.OpeningsSHA != c.OpeningsSHA {
		t.Fatal("engine bytes not bound independently")
	}
	options := []string{"setoption name EvalFile value new.nnue", "setoption name Threads value 1"}
	d, err := makeRecipe(enginePath, [][]string{{"e2e4"}}, params, 100, 6, false, 10000, 100, .05, .1, 25, 1, false, options...)
	if err != nil || !reflect.DeepEqual(d.EngineOptions, options) {
		t.Fatalf("fixed options missing from recipe: %v %v", d.EngineOptions, err)
	}
	options[0] = "changed"
	if d.EngineOptions[0] == "changed" {
		t.Fatal("recipe aliases mutable option input")
	}
}
