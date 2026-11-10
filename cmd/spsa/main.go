// Command spsa runs Simultaneous Perturbation Stochastic Approximation over
// NGN's tunable search parameters (engine.TunableSearchParams), tuning the whole
// vector JOINTLY — the optimum the one-knob-at-a-time SPRT is structurally unable
// to find because search params interact (LMR base couples with NMP R couples
// with the futility/RFP margins). Each iteration perturbs theta by +/-c*delta,
// sets theta+ on one engine process and theta- on the other via UCI setoption,
// plays a color-balanced game pair at a real clock, and nudges theta toward
// whichever side scored (fishtest-style update).
//
// Real clock, not fixed nodes: most of these params (LMR/NMP/futility/singular)
// are node-SPENDING techniques, so a fixed-node budget would charge their nodes
// without crediting the depth they buy — the wrong ruler. Real clock is correct.
//
// The gate for the RESULT vector is the GAUNTLET (self-play tuning can drift
// toward self-play-specific quirks), never this driver's own score. Validate the
// update math instantly with -selftest (synthetic quadratic objective, no games)
// before committing days of compute.
//
//	PARAMS=LMRBase,LMRDivisor,NullMoveDepthDivisor
//	./build/spsa -params $PARAMS -selftest             # synthetic math check, no games
//	./build/spsa -params $PARAMS -iters 20 -tc 10+0.1  # wiring sanity
//	./build/spsa -params $PARAMS -iters 12000          # full campaign
//	./build/spsa -params $PARAMS -resume               # resume the same recipe

package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/internal/uci"
)

// Standard Spall SPSA exponents (the values fishtest uses).
const (
	spsaAlpha = 0.602
	spsaGamma = 0.101
)

// pvar is one tunable parameter's running float theta plus its precomputed
// fishtest schedule constants. theta is kept as a float and rounded only when
// handed to the engine via setoption, so sub-integer gradient steps accumulate.
type pvar struct {
	name     string
	theta    float64
	min, max float64
	cEnd     float64 // perturbation magnitude at the final iteration
	cAmp     float64 // c_i: c_k = cAmp / k^gamma
	aGain    float64 // a_i: a_k = aGain / (A+k)^alpha
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var engineOptions optionFlags
	flag.Var(&engineOptions, "engineoption", "UCI NAME=VALUE on both engines before readiness (repeatable, ordered)")
	enginePath := flag.String("engine", "./build/ngn", "engine binary to tune (one process plays theta+, one plays theta-)")
	tc := flag.String("tc", "10+0.1", "real clock SECONDS[+INC] (e.g. 10+0.1); search params are node-spending so real clock is the correct ruler")
	iters := flag.Int("iters", 12000, "SPSA iterations (each = one color-balanced game pair unless -single)")
	chunkIters := flag.Int("chunk-iters", 0, "maximum new iterations this invocation; 0 runs the full schedule (unchanged on resume)")
	single := flag.Bool("single", false, "one game per iteration instead of a color-balanced pair (2x faster, noisier gradient)")
	pairs := flag.Int("pairs", 1, "concurrent perturbations averaged per iteration — parallelizes wall-clock K× and cuts gradient variance ~1/K (T8 campaign uses 4); 1 = the original sequential SPSA (identical rng stream)")
	openingsFile := flag.String("openings", "", "opening book (one move-seq per line); default = built-in set")
	outFile := flag.String("out", "output/spsa.jsonl", "checkpoint/history JSONL (also the -resume source)")
	resume := flag.Bool("resume", false, "resume the exact recipe, theta and RNG from a versioned checkpoint in -out")
	rEnd := flag.Float64("rend", 0.05, "SPSA end learning-rate scale (bigger = faster but noisier param movement)")
	aFrac := flag.Float64("afrac", 0.1, "SPSA stability constant A as a fraction of -iters")
	report := flag.Int("report", 25, "log + checkpoint theta every N iterations")
	seed := flag.Int64("seed", 1, "RNG seed (perturbation signs + opening selection)")
	lowPower := flag.Bool("lowpower", true, "route engines to macOS E-cores (taskpolicy -b)")
	selftest := flag.Bool("selftest", false, "validate the SPSA math against a synthetic quadratic objective (no games), then exit")
	paramNames := flag.String("params", "", "comma-separated search parameters to tune; each must have at least 20 steps (empty selects all)")
	flag.Parse()

	if *iters < 1 || *pairs < 1 || *report < 1 || !finite(*rEnd) || *rEnd <= 0 || !finite(*aFrac) || *aFrac < 0 {
		return fmt.Errorf("iters, pairs, report and rend must be positive; afrac must be finite and nonnegative")
	}
	if _, err := campaignEnd(1, *iters, *chunkIters, *report); err != nil {
		return err
	}
	if *selftest && *resume {
		return fmt.Errorf("selftest cannot resume a real-game campaign")
	}
	K := *pairs
	rng, source := newCampaignRNG(*seed)

	// Build the parameter vector from the engine registry (single source of truth).
	params, err := tuningParams(engine.TunableSearchParams, *paramNames)
	if err != nil {
		return err
	}

	// Precompute the fishtest schedule: c_k = cEnd*(iters/k)^gamma shrinks to cEnd
	// at the end; a_k holds the per-iter step ~ rEnd*cEnd near convergence. cEnd
	// auto-scales to each param's range so wide and narrow params move comparably.
	A := *aFrac * float64(*iters)
	for _, p := range params {
		p.cEnd = math.Max(0.5, (p.max-p.min)/20.0)
		p.cAmp = p.cEnd * math.Pow(float64(*iters), spsaGamma)
		p.aGain = *rEnd * p.cEnd * p.cEnd * math.Pow(A+float64(*iters), spsaAlpha)
	}

	if *selftest {
		runSelfTest(params, *iters, K, A, rng)
		return nil
	}

	// --- real-game objective ---
	tcTimeMs, tcIncMs := parseTC(*tc)
	openings, err := uci.LoadOpenings(*openingsFile)
	if err != nil || len(openings) == 0 {
		return fmt.Errorf("loading openings: %v (got %d)", err, len(openings))
	}
	recipe, err := makeRecipe(*enginePath, openings, params, *iters, K, *single,
		tcTimeMs, tcIncMs, *rEnd, *aFrac, *report, *seed, *lowPower, engineOptions...)
	if err != nil {
		return err
	}
	startIter := 1
	if *resume {
		var saved int
		saved, rng, source, err = loadCheckpoint(*outFile, params, recipe)
		if err != nil {
			return fmt.Errorf("resume refused: %w", err)
		}
		startIter = saved + 1
		fmt.Printf("resumed exact recipe from iter %d (%s), RNG draws=%d\n", saved, *outFile, source.draws)
	} else if err := prepareHistory(*outFile); err != nil {
		return err
	}
	if startIter > *iters {
		fmt.Println("campaign already complete; gate the saved vector before accepting it")
		printSetoptions(params)
		return nil
	}
	endIter, err := campaignEnd(startIter, *iters, *chunkIters, *report)
	if err != nil {
		return err
	}

	type worker struct{ plus, minus *uci.Engine }
	workers := make([]worker, K)
	defer func() {
		for _, w := range workers {
			if w.plus != nil {
				uci.Stop(w.plus)
			}
			if w.minus != nil {
				uci.Stop(w.minus)
			}
		}
	}()
	for j := range workers {
		workers[j].plus = uci.StartWithOptions(*enginePath, fmt.Sprintf("plus%d", j), *lowPower, engineOptions)
		workers[j].minus = uci.StartWithOptions(*enginePath, fmt.Sprintf("minus%d", j), *lowPower, engineOptions)
		if workers[j].plus == nil || workers[j].minus == nil {
			return fmt.Errorf("engine failed UCI handshake after retries (worker %d)", j)
		}
	}

	cfg := uci.GameConfig{TCTimeMs: tcTimeMs, TCIncMs: tcIncMs, MaxMoves: 200}

	fmt.Printf("NGN joint search-param SPSA\n===========================\n")
	fmt.Printf("engine=%s  params=%d  iters=%d (%s, %d/iter)  tc=%ds+%.1fs%s\n",
		*enginePath, len(params), *iters, pairMode(*single), K, tcTimeMs/1000, float64(tcIncMs)/1000, uci.PowerNote(*lowPower))
	fmt.Printf("openings=%d  out=%s  rend=%.3f  afrac=%.2f  seed=%d\n\n", len(openings), *outFile, *rEnd, *aFrac, *seed)
	printTheta(params, "start")

	plusPts, games, forfeits := 0.0, 0, 0 // health since last report: theta+ ~50%, forfeits ~0
	// playBatch plays one iteration's K perturbations concurrently — worker j runs
	// plusV[j] vs minusV[j] on its own process pair over opening opIdx[j] — and
	// returns each pair's R in [-1,1] from theta+'s POV. The sign/opening draws
	// happen in runCampaign under the single-threaded rng; only the games run in
	// parallel, so the run stays deterministic in -seed regardless of scheduling.
	playBatch := func(plusV, minusV [][]int, opIdx []int) []float64 {
		R := make([]float64, len(plusV))
		var mu sync.Mutex
		var wg sync.WaitGroup
		for j := range plusV {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				w := workers[j]
				setParams(w.plus, params, plusV[j])
				setParams(w.minus, params, minusV[j])
				op := openings[opIdx[j]]
				g1 := uci.PlayGame(w.plus, w.minus, op, true, cfg) // theta+ as White
				s, err := validGamePoints(g1)
				if err != nil {
					log.Printf("worker %d: %v", j, err)
					R[j] = math.NaN()
					return
				}
				n, ff := 1, 0
				if g1.Reason == "time-forfeit" {
					ff++
				}
				if !*single {
					g2 := uci.PlayGame(w.plus, w.minus, op, false, cfg) // theta+ as Black
					second, err := validGamePoints(g2)
					if err != nil {
						log.Printf("worker %d: %v", j, err)
						R[j] = math.NaN()
						return
					}
					if g2.Reason == "time-forfeit" {
						ff++
					}
					s += second
					n = 2
				}
				// R in [-1,1]: pair s in [0,2] -> s-1; single s in {0,.5,1} -> 2s-1.
				if *single {
					R[j] = 2*s - 1
				} else {
					R[j] = s - 1
				}
				mu.Lock()
				plusPts += s
				games += n
				forfeits += ff
				mu.Unlock()
			}(j)
		}
		wg.Wait()
		return R
	}

	onReport := func(k int) error {
		pct := 50.0
		if games > 0 {
			pct = 100 * plusPts / float64(games)
		}
		fmt.Printf("[iter %d] theta+ %.1f%% over last %d games (%d forfeits)  ", k, pct, games, forfeits)
		printTheta(params, "now")
		if err := writeCheckpoint(*outFile, k, params, pct, recipe, source.draws); err != nil {
			return err
		}
		plusPts, games, forfeits = 0, 0, 0
		return nil
	}

	if err := runCampaign(params, startIter, endIter, A, rng, *report, K, len(openings), playBatch, onReport); err != nil {
		return err
	}

	if endIter < *iters {
		fmt.Printf("\nchunk complete at iter %d; resume the unchanged %d-iteration schedule with -resume\n", endIter, *iters)
		return nil
	}
	fmt.Printf("\n=== FINAL VECTOR (gate on the GAUNTLET, not this run's score) ===\n")
	printSetoptions(params)
	return nil
}

// Chunk size controls execution only. Schedule constants and the checkpoint
// recipe always use the total iteration count. End on a normal report boundary
// so loadCheckpoint can validate the uninterrupted checkpoint cadence.
func campaignEnd(start, total, chunk, report int) (int, error) {
	if chunk < 0 {
		return 0, fmt.Errorf("chunk-iters must be nonnegative")
	}
	end := total
	if chunk > 0 && chunk < total-start+1 {
		end = start + chunk - 1
	}
	if end != total && end%report != 0 {
		return 0, fmt.Errorf("chunk ends at iter %d outside report cadence %d", end, report)
	}
	return end, nil
}

// runCampaign is the SPSA loop shared by the real and selftest objectives.
// score(plus,minus) returns R in [-1,1] (>0 means theta+ was better) plus a
// label; onReport(k) is called every `report` iters and at the end.
func runCampaign(params []*pvar, startIter, iters int, A float64, rng *rand.Rand,
	report, K, nOpenings int,
	playBatch func(plusV, minusV [][]int, opIdx []int) []float64, onReport func(k int) error) error {

	for k := startIter; k <= iters; k++ {
		kf := float64(k)
		// Draw K independent perturbations for this step. All rng use is here in the
		// single main goroutine, so the sequence is deterministic in -seed no matter
		// how the K games race in playBatch.
		deltas := make([][]float64, K)
		plusV := make([][]int, K)
		minusV := make([][]int, K)
		opIdx := make([]int, K)
		for j := 0; j < K; j++ {
			deltas[j] = make([]float64, len(params))
			plusV[j] = make([]int, len(params))
			minusV[j] = make([]int, len(params))
			for i, p := range params {
				d := 1.0
				if rng.Intn(2) == 0 {
					d = -1.0
				}
				deltas[j][i] = d
				c := p.cAmp / math.Pow(kf, spsaGamma)
				plusV[j][i] = clampRound(p.theta+c*d, p.min, p.max)
				minusV[j][i] = clampRound(p.theta-c*d, p.min, p.max)
			}
			if nOpenings > 0 {
				opIdx[j] = rng.Intn(nOpenings)
			}
		}

		R := playBatch(plusV, minusV, opIdx)
		if len(R) != K {
			return fmt.Errorf("iteration %d: incomplete game batch", k)
		}
		for j, score := range R {
			if !finite(score) || score < -1 || score > 1 {
				return fmt.Errorf("iteration %d worker %d: faulted game batch; theta and checkpoint unchanged", k, j)
			}
		}

		// Averaged fishtest update: mean over the K perturbations' R*delta gradient
		// estimates (variance ~1/K, expected direction unchanged). c is common to the
		// step so (a/c) factors out of the mean.
		for i, p := range params {
			c := p.cAmp / math.Pow(kf, spsaGamma)
			a := p.aGain / math.Pow(A+kf, spsaAlpha)
			grad := 0.0
			for j := 0; j < K; j++ {
				grad += R[j] * deltas[j][i]
			}
			grad /= float64(K)
			p.theta += (a / c) * grad // maximize theta+'s score
			if p.theta < p.min {
				p.theta = p.min
			}
			if p.theta > p.max {
				p.theta = p.max
			}
		}

		if k%report == 0 || k == iters {
			if err := onReport(k); err != nil {
				return err
			}
		}
	}
	return nil
}

func validGamePoints(game uci.GameResult) (float64, error) {
	switch game.Reason {
	case "checkmate", "stalemate", "draw-rule", "max-moves", "insufficient-material":
		return points(game.Res), nil
	default:
		return 0, fmt.Errorf("game fault: %s", game.Reason)
	}
}

// runSelfTest drives the exact SPSA loop against a synthetic quadratic bowl with
// a known optimum, so the update math is validated deterministically in seconds
// without playing a single game. theta should converge to `target`.
func runSelfTest(params []*pvar, iters, K int, A float64, rng *rand.Rand) {
	target := make([]float64, len(params))
	for i, p := range params {
		target[i] = p.min + 0.4*(p.max-p.min) // distinct from the def so movement is visible
	}
	loss := func(vals []int) float64 {
		s := 0.0
		for i, p := range params {
			d := (float64(vals[i]) - target[i]) / (p.max - p.min)
			s += d * d
		}
		return s
	}
	playBatch := func(plusV, minusV [][]int, _ []int) []float64 {
		R := make([]float64, len(plusV))
		for j := range plusV {
			R[j] = math.Tanh(4.0 * (loss(minusV[j]) - loss(plusV[j]))) // >0 when plus is closer
		}
		return R
	}
	fmt.Printf("SPSA selftest: %d params, %d iters, %d/iter, synthetic quadratic\n", len(params), iters, K)
	runCampaign(params, 1, iters, A, rng, max(iters/10, 1), K, 0, playBatch, func(k int) error {
		fmt.Printf("[iter %d] ", k)
		printTheta(params, "now")
		return nil
	})
	fmt.Println("\n=== SELFTEST: theta vs hidden target ===")
	maxErr := 0.0
	for i, p := range params {
		e := math.Abs(p.theta-target[i]) / (p.max - p.min)
		if e > maxErr {
			maxErr = e
		}
		fmt.Printf("  %-22s theta %6.2f  target %6.2f  err %4.1f%%  (range %.0f-%.0f)\n",
			p.name, p.theta, target[i], 100*e, p.min, p.max)
	}
	verdict := "PASS (SPSA converges — update math is correct)"
	if maxErr >= 0.15 {
		verdict = "CHECK (raise -iters or -rend; some param did not converge)"
	}
	fmt.Printf("max normalized error: %.1f%%  ->  %s\n", 100*maxErr, verdict)
}

// --- helpers ---

func setParams(e *uci.Engine, params []*pvar, vals []int) {
	for i, p := range params {
		uci.Send(e, fmt.Sprintf("setoption name %s value %d", p.name, vals[i]))
	}
}

func points(o uci.Outcome) float64 {
	switch o {
	case uci.Win:
		return 1
	case uci.Draw:
		return 0.5
	default:
		return 0
	}
}

func clampRound(x, lo, hi float64) int {
	return int(min(max(math.Round(x), lo), hi))
}

func parseTC(tc string) (timeMs, incMs int) {
	parts := strings.SplitN(tc, "+", 2)
	base, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || base <= 0 {
		log.Fatalf("bad -tc %q (want SECONDS or SECONDS+INC, e.g. 10+0.1)", tc)
	}
	timeMs = int(base * 1000)
	if len(parts) == 2 {
		inc, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil || inc < 0 {
			log.Fatalf("bad -tc increment in %q", tc)
		}
		incMs = int(inc * 1000)
	}
	return timeMs, incMs
}

func pairMode(single bool) string {
	if single {
		return "single game"
	}
	return "game pair"
}

func printTheta(params []*pvar, tag string) {
	var b strings.Builder
	b.WriteString(tag)
	b.WriteString(": ")
	for _, p := range params {
		fmt.Fprintf(&b, "%s=%d ", p.name, clampRound(p.theta, p.min, p.max))
	}
	fmt.Println(b.String())
}

func printSetoptions(params []*pvar) {
	for _, p := range params {
		v := clampRound(p.theta, p.min, p.max)
		def := defaultOf(p.name)
		flag := ""
		if v != def {
			flag = fmt.Sprintf("   (was %d)", def)
		}
		fmt.Printf("setoption name %s value %d%s\n", p.name, v, flag)
	}
}

func defaultOf(name string) int {
	for _, tp := range engine.TunableSearchParams {
		if tp.Name == name {
			return tp.Def
		}
	}
	return 0
}
