// Command sprt runs an NGN-vs-NGN self-play SPRT — the fine-grained measurement
// instrument the 5-game vs-Stockfish smoke cannot be (an identical binary swung
// 80%->20% there). It plays a new binary against a base binary over a set of
// balanced openings (each played twice, colors reversed, for variance
// reduction), detects game ends authoritatively via the engine package
// (checkmate/stalemate/threefold/50-move), and runs a Generalized SPRT on the
// trinomial score, stopping as soon as the log-likelihood ratio crosses a bound.
//
// It defaults to FIXED-DEPTH games (go depth N), not time-based. That removes
// time noise entirely and asks the question we actually care about for tuning:
// "at an equal depth budget, does the new eval/ordering pick better moves?" Use
// -movetime for a final time-control validation once a change looks good.
//
// The UCI subprocess harness and Elo math live in internal/uci and
// internal/rating (shared with cmd/smoke, cmd/elo-assess, and cmd/gauntlet).
//
//	# build a baseline to test against, then a candidate
//	cp build/ngn build/ngn_base   # baseline = current HEAD
//	# ...make a change, rebuild build/ngn...
//	./build/sprt -depth 9 -elo1 10            # is new stronger by >=10 ELO?
//	./build/sprt -genbook 500 -out book.txt   # generate openings for finer runs
//	./build/sprt -openings book.txt -depth 9 -elo1 5
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/internal/rating"
	"github.com/ehrlich-b/ngn/internal/uci"
)

type spec struct {
	opening    []string
	newIsWhite bool
}

func main() {
	newPath := flag.String("new", "./build/ngn", "candidate engine binary")
	basePath := flag.String("base", "./build/ngn_base", "baseline engine binary")
	depth := flag.Int("depth", 9, "fixed search depth per move (0 = use -movetime)")
	movetime := flag.Int("movetime", 0, "ms per move (overrides -depth when > 0)")
	nodes := flag.Int("nodes", 0, "fixed node budget per move (overrides -depth; deterministic, credits ordering+eval, cool at any concurrency)")
	baseDepth := flag.Int("basedepth", 0, "base engine depth (0 = same as -depth; set lower as a positive control / depth ladder)")
	elo0 := flag.Float64("elo0", 0, "SPRT H0 elo bound (null: new is not better)")
	elo1 := flag.Float64("elo1", 10, "SPRT H1 elo bound (alt: new is better by this)")
	alpha := flag.Float64("alpha", 0.05, "SPRT type-I error")
	beta := flag.Float64("beta", 0.05, "SPRT type-II error")
	maxGames := flag.Int("maxgames", 1000, "hard cap on games")
	minGames := flag.Int("mingames", 24, "minimum games before the SPRT stop bound can fire")
	maxMoves := flag.Int("maxmoves", 200, "ply cap per game (then adjudicated draw)")
	openingsFile := flag.String("openings", "", "file of opening move-sequences (one per line); default = built-in set")
	concurrency := flag.Int("concurrency", 2, "parallel games")
	lowPower := flag.Bool("lowpower", true, "route engines to macOS E-cores (taskpolicy -b)")
	genBook := flag.Int("genbook", 0, "generate N balanced openings and exit (writes to -out)")
	genPlies := flag.Int("genplies", 6, "random plies per generated opening")
	balanceCp := flag.Int("balancecp", 100, "|eval| ceiling for a generated opening to count as balanced")
	balanceDepth := flag.Int("balancedepth", 8, "search depth used to score opening balance")
	outFile := flag.String("out", "", "output file for -genbook")
	seed := flag.Int64("seed", 1, "RNG seed for -genbook")
	flag.Parse()

	if *genBook > 0 {
		runGenBook(*genBook, *genPlies, *balanceCp, *balanceDepth, *seed, *outFile)
		return
	}

	openings, err := uci.LoadOpenings(*openingsFile)
	if err != nil {
		log.Fatalf("loading openings: %v", err)
	}
	if len(openings) == 0 {
		log.Fatal("no usable openings")
	}

	for _, p := range []string{*newPath, *basePath} {
		if _, err := os.Stat(p); err != nil {
			log.Fatalf("engine binary not found: %s (build it first)", p)
		}
	}

	newDepth := *depth
	baseD := *depth
	if *baseDepth > 0 {
		baseD = *baseDepth
	}
	mode := fmt.Sprintf("depth %d", *depth)
	if baseD != *depth {
		mode = fmt.Sprintf("new depth %d vs base depth %d", *depth, baseD)
	}
	if *movetime > 0 {
		mode = fmt.Sprintf("movetime %dms", *movetime)
		if *concurrency > 1 {
			fmt.Printf("WARNING: movetime mode with concurrency %d — core contention makes timing (and thus results) load-dependent. Use -concurrency 1 for clean measurement.\n", *concurrency)
		}
	}
	if *nodes > 0 {
		mode = fmt.Sprintf("nodes %d/move", *nodes)
	}

	lower := math.Log(*beta / (1 - *alpha))
	upper := math.Log((1 - *beta) / *alpha)

	fmt.Printf("NGN self-play SPRT\n")
	fmt.Printf("==================\n")
	fmt.Printf("new=%s  base=%s\n", *newPath, *basePath)
	fmt.Printf("mode: %s | openings: %d (x2 colors) | concurrency: %d%s\n",
		mode, len(openings), *concurrency, uci.PowerNote(*lowPower))
	fmt.Printf("H0: elo<=%.1f   H1: elo>=%.1f   (alpha=%.2f beta=%.2f -> LLR bounds [%.2f, %.2f])\n\n",
		*elo0, *elo1, *alpha, *beta, lower, upper)

	runSPRT(runConfig{
		newPath: *newPath, basePath: *basePath,
		newDepth: newDepth, baseDepth: baseD, movetime: *movetime, nodes: *nodes, maxMoves: *maxMoves,
		elo0: *elo0, elo1: *elo1, lower: lower, upper: upper,
		maxGames: *maxGames, minGames: *minGames, concurrency: *concurrency, lowPower: *lowPower,
	}, openings)
}

type runConfig struct {
	newPath, basePath               string
	newDepth, baseDepth, movetime   int
	nodes                           int
	maxMoves                        int
	elo0, elo1, lower, upper        float64
	maxGames, minGames, concurrency int
	lowPower                        bool
}

func runSPRT(cfg runConfig, openings [][]string) {
	// Build the work list: each opening twice with reversed colors. At fixed
	// depth OR fixed nodes a repeated (opening,color) is a deterministic replay
	// carrying no new information, so cap distinct work at 2*len(openings); only
	// movetime mode (timing variance) justifies cycling beyond that.
	var specs []spec
	for _, op := range openings {
		specs = append(specs, spec{op, true}, spec{op, false})
	}
	total := len(specs)
	if cfg.movetime > 0 {
		// movetime: cycle openings up to maxGames
		for len(specs) < cfg.maxGames {
			specs = append(specs, specs[len(specs)%total])
		}
		total = cfg.maxGames
	} else if total > cfg.maxGames {
		total = cfg.maxGames
	}
	specs = specs[:total]

	specCh := make(chan spec, total)
	for _, s := range specs {
		specCh <- s
	}
	close(specCh)

	gameCfg := uci.GameConfig{
		NewDepth: cfg.newDepth, BaseDepth: cfg.baseDepth,
		Movetime: cfg.movetime, Nodes: cfg.nodes, MaxMoves: cfg.maxMoves,
	}

	results := make(chan uci.GameResult, cfg.concurrency)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < cfg.concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newEng := uci.Start(cfg.newPath, "new", cfg.lowPower)
			baseEng := uci.Start(cfg.basePath, "base", cfg.lowPower)
			if newEng == nil || baseEng == nil {
				log.Fatalf("engine failed UCI handshake after retries (new ok=%v, base ok=%v); aborting run instead of forfeiting every game as no-move", newEng != nil, baseEng != nil)
			}
			defer uci.Stop(newEng)
			defer uci.Stop(baseEng)
			for s := range specCh {
				select {
				case <-stop:
					return
				default:
				}
				results <- uci.PlayGame(newEng, baseEng, s.opening, s.newIsWhite, gameCfg)
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()

	var w, d, l int
	start := time.Now()
	stopped := false
	verdict := "INCONCLUSIVE (ran out of games — add openings or raise -maxgames)"
	for r := range results {
		switch r.Res {
		case uci.Win:
			w++
		case uci.Draw:
			d++
		case uci.Loss:
			l++
		}
		n := w + d + l
		llr := sprtLLR(w, d, l, cfg.elo0, cfg.elo1)
		est, lo, hi := rating.RelativeElo(w, d, l)
		fmt.Printf("G%-4d %2dW %2dD %2dL  %5.1f%%  elo %+6.1f [%+.0f,%+.0f]  LLR %+6.2f  %-12s\n",
			n, w, d, l, 100*rating.Score(w, d)/float64(n), est, lo, hi, llr, r.Reason)

		if !stopped && n >= cfg.minGames && (llr >= cfg.upper || llr <= cfg.lower) {
			if llr >= cfg.upper {
				verdict = fmt.Sprintf("H1 ACCEPTED: new is stronger (>= %.0f ELO)", cfg.elo1)
			} else {
				verdict = fmt.Sprintf("H0 ACCEPTED: new is NOT better (<= %.0f ELO)", cfg.elo0)
			}
			close(stop)
			stopped = true
		}
	}

	w0, d0, l0 := w, d, l
	n := w0 + d0 + l0
	est, lo, hi := rating.RelativeElo(w0, d0, l0)
	fmt.Printf("\n=== RESULT (%v) ===\n", time.Since(start).Truncate(time.Second))
	fmt.Printf("Games: %d   W-D-L: %d-%d-%d   score: %.1f%%\n", n, w0, d0, l0, 100*rating.Score(w0, d0)/math.Max(1, float64(n)))
	fmt.Printf("Elo(new - base): %+.1f   95%% CI [%+.0f, %+.0f]\n", est, lo, hi)
	fmt.Printf("LLR: %+.2f   bounds [%.2f, %.2f]\n", sprtLLR(w0, d0, l0, cfg.elo0, cfg.elo1), cfg.lower, cfg.upper)
	fmt.Printf("Verdict: %s\n", verdict)
}

// --- SPRT statistics (Generalized SPRT on the trinomial score) ---

func eloToScore(elo float64) float64 { return 1.0 / (1.0 + math.Pow(10, -elo/400)) }

// sprtLLR is the GSPRT linear log-likelihood ratio for H0:elo0 vs H1:elo1,
// using the empirical mean and variance of the per-game score. Positive LLR
// favors H1 (new stronger), negative favors H0.
func sprtLLR(w, d, l int, elo0, elo1 float64) float64 {
	n := w + d + l
	if n == 0 {
		return 0
	}
	nf := float64(n)
	S := rating.Score(w, d)
	mean := S / nf
	// outcome second moment: win contributes 1, draw 0.25, loss 0
	sumSq := float64(w) + 0.25*float64(d)
	variance := sumSq/nf - mean*mean
	// Floor at a realistic minimum (~a very drawish match). Binds only in
	// degenerate cases (tiny n, all-identical results) where it keeps the LLR
	// from blowing up; for normal score variance (0.1-0.25) it never binds.
	variance = max(variance, 0.05)
	s0, s1 := eloToScore(elo0), eloToScore(elo1)
	return (s1 - s0) / variance * (S - nf*(s0+s1)/2.0)
}

// --- opening book generation ---

func runGenBook(n, plies, balanceCp, balanceDepth int, seed int64, outPath string) {
	rng := rand.New(rand.NewSource(seed))
	seen := map[string]bool{}
	var out []string
	maxAttempts := n * 200
	for attempts := 0; len(out) < n && attempts < maxAttempts; attempts++ {
		pos, _ := engine.ParseFEN(uci.StartFEN)
		pos.Positions = make(map[uint64]int)
		pos.Positions[pos.Hash()]++
		moves := make([]string, 0, plies)
		ok := true
		for p := 0; p < plies; p++ {
			legal := engine.GenerateLegalMoves(pos)
			if len(legal) == 0 {
				ok = false
				break
			}
			m := legal[rng.Intn(len(legal))]
			pos.GameMakeMove(m)
			moves = append(moves, m.ToString())
		}
		if !ok || len(engine.GenerateLegalMoves(pos)) == 0 {
			continue
		}
		key := strings.Join(moves, " ")
		if seen[key] {
			continue
		}
		info := engine.SearchIterativeDeepening(pos, balanceDepth, nil)
		sc := info.BestScore
		if sc < 0 {
			sc = -sc
		}
		if sc > balanceCp { // unbalanced or mate-ish
			continue
		}
		seen[key] = true
		out = append(out, key)
		if len(out)%25 == 0 {
			fmt.Printf("  generated %d/%d (after %d attempts)\n", len(out), n, attempts+1)
		}
	}
	body := strings.Join(out, "\n") + "\n"
	if outPath == "" {
		fmt.Print(body)
	} else {
		if err := os.WriteFile(outPath, []byte(body), 0644); err != nil {
			log.Fatalf("writing %s: %v", outPath, err)
		}
		fmt.Printf("wrote %d balanced openings to %s\n", len(out), outPath)
	}
	if len(out) < n {
		fmt.Printf("NOTE: only %d/%d found; raise -balancecp or lower -genplies for a higher yield.\n", len(out), n)
	}
}
