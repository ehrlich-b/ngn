// Smoke test for fast iteration. Two phases:
//
//  1. Tactical suite — load positions from an EPD and run the engine in-process
//     with a short per-position budget. A sharp drop in solve count is a clear
//     search regression and we fail fast.
//  2. Game smoke — short asymmetric game match (default 5 games at 1s NGN /
//     200ms SF) via the existing elo-assess binary. We only fire on extreme
//     outcomes (0 wins or all wins); mid outcomes are treated as "no signal".
//
// Designed to run on battery in a few minutes. If it passes, you have weak
// evidence the change isn't broken; you can still run a full ELO test on AC
// power before claiming an actual ELO delta.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

// runRecord is one row of the rolling-ELO history. Append-only JSONL.
type runRecord struct {
	Timestamp time.Time `json:"ts"`
	SHA       string    `json:"sha"`
	NgnMs     int       `json:"ngn_ms"`
	SfMs      int       `json:"sf_ms"`
	TargetELO int       `json:"target_elo"`
	Games     int       `json:"games"`
	NgnW      int       `json:"ngn_w"`
	Draws     int       `json:"draws"`
	SfW       int       `json:"sf_w"`
}

func main() {
	nTactical := flag.Int("tactical", 30, "Number of tactical positions to test")
	tMs := flag.Int("tactical-time", 200, "ms per tactical position")
	nGames := flag.Int("games", 5, "Game-smoke games (0 = skip)")
	ngnMs := flag.Int("ngn-ms", 1000, "NGN ms per move (game phase)")
	sfMs := flag.Int("sf-ms", 200, "Stockfish ms per move (game phase)")
	targetELO := flag.Int("elo", 1500, "Stockfish ELO target")
	skipTactical := flag.Bool("skip-tactical", false, "Skip tactical phase")
	skipGames := flag.Bool("skip-games", false, "Skip game phase")
	epdPath := flag.String("epd", "validated_tactical_positions.epd", "EPD file path")
	threshold := flag.Float64("threshold", 0.40, "Min tactical solve rate to pass (0-1)")
	seed := flag.Int64("seed", 42, "RNG seed for deterministic position selection")
	historyPath := flag.String("history", "output/smoke_history.jsonl", "JSONL file for rolling-ELO history")
	noRecord := flag.Bool("no-record", false, "Don't append this run to history")
	rollingOnly := flag.Bool("rolling", false, "Just print the rolling-ELO summary and exit (no smoke run)")
	flag.Parse()

	fmt.Println("NGN smoke test")
	fmt.Println("==============")

	if *rollingOnly {
		printRolling(*historyPath, *targetELO, *ngnMs, *sfMs)
		return
	}

	fail := false

	if !*skipTactical {
		if !runTactical(*epdPath, *nTactical, *tMs, *threshold, *seed) {
			fail = true
		}
	}

	var gameStats *runRecord
	if !*skipGames && *nGames > 0 {
		v, rec := runGameSmoke(*nGames, *ngnMs, *sfMs, *targetELO)
		if v == "regression" {
			fail = true
		}
		if rec != nil && !*noRecord {
			if err := appendRun(*historyPath, *rec); err != nil {
				fmt.Printf("WARN: could not record run: %v\n", err)
			}
			gameStats = rec
		}
	}

	if gameStats != nil {
		printRolling(*historyPath, gameStats.TargetELO, gameStats.NgnMs, gameStats.SfMs)
	}

	fmt.Println()
	if fail {
		fmt.Println("=== SMOKE FAIL ===")
		os.Exit(1)
	}
	fmt.Println("=== SMOKE OK ===")
}

type puzzle struct {
	FEN string
	BM  string // best move in UCI form
}

func runTactical(epdPath string, n, msPerPos int, threshold float64, seed int64) bool {
	fmt.Printf("\n== Phase 1: tactical (%d positions @ %dms each) ==\n", n, msPerPos)
	all, err := loadEPD(epdPath)
	if err != nil {
		fmt.Printf("WARN: cannot load %s: %v (skipping tactical)\n", epdPath, err)
		return true
	}
	if len(all) == 0 {
		fmt.Println("WARN: no positions in EPD; skipping tactical")
		return true
	}

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	if n > len(all) {
		n = len(all)
	}
	picks := all[:n]

	correct := 0
	start := time.Now()
	for i, p := range picks {
		// Fresh global state per puzzle so cross-contamination from one
		// puzzle's TT/history doesn't bias the next one.
		engine.TranspositionTable = engine.NewCache(engine.DEFAULT_CACHE_SIZE)
		engine.ClearHistoryTable()
		engine.ClearKillerMoves()
		engine.ClearCounterMoves()
		engine.SetLastMovePlayed(engine.EmptyMove)

		pos, err := engine.ParseFEN(p.FEN)
		if err != nil {
			continue
		}
		tm := engine.NewTimeManager()
		tm.SetTimeControl(engine.SearchParams{MoveTime: msPerPos}, pos.Turn() == engine.White)
		info := engine.SearchIterativeDeepening(pos, 64, tm)
		got := info.BestMove.ToString()
		if got == p.BM {
			correct++
		}
		// Light progress trace every 10 puzzles so the user sees movement.
		if (i+1)%10 == 0 {
			fmt.Printf("  ... %d/%d done (%d correct)\n", i+1, len(picks), correct)
		}
	}
	elapsed := time.Since(start)
	rate := float64(correct) / float64(len(picks))
	fmt.Printf("\nSolved: %d/%d (%.0f%%) in %v\n", correct, len(picks), rate*100, elapsed.Round(time.Millisecond))
	if rate < threshold {
		fmt.Printf("FAIL: solve rate %.0f%% below threshold %.0f%%\n", rate*100, threshold*100)
		return false
	}
	fmt.Printf("OK: solve rate ≥ threshold (%.0f%%)\n", threshold*100)
	return true
}

func loadEPD(path string) ([]puzzle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []puzzle
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "bm ", 2)
		if len(parts) != 2 {
			continue
		}
		fen := strings.TrimSpace(parts[0])
		bm := parts[1]
		if i := strings.Index(bm, ";"); i >= 0 {
			bm = bm[:i]
		}
		out = append(out, puzzle{FEN: fen, BM: strings.TrimSpace(bm)})
	}
	return out, scan.Err()
}

// runGameSmoke shells out to ./build/elo-assess for a fixed-game match and
// returns (verdict, record). verdict is "regression"/"neutral"/"improvement"/"error".
// record is non-nil iff the games actually completed and counts parsed cleanly.
func runGameSmoke(games, ngnMs, sfMs, targetELO int) (string, *runRecord) {
	fmt.Printf("\n== Phase 2: %d-game smoke (NGN %dms vs SF%d %dms) ==\n",
		games, ngnMs, targetELO, sfMs)

	cmd := exec.Command("./build/elo-assess",
		"-elo", strconv.Itoa(targetELO),
		"-movetime", strconv.Itoa(ngnMs),
		"-sfmovetime", strconv.Itoa(sfMs),
		"-games", strconv.Itoa(games),
		"-min", strconv.Itoa(games),
		"-ci", "99999", // disable CI-based early stop
	)
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("WARN: elo-assess failed: %v\n", err)
		return "error", nil
	}

	out := buf.String()
	parseInt := func(re *regexp.Regexp) int {
		m := re.FindStringSubmatch(out)
		if len(m) < 2 {
			return -1
		}
		n, _ := strconv.Atoi(m[1])
		return n
	}
	ngnW := parseInt(regexp.MustCompile(`NGN wins: (\d+)`))
	sfW := parseInt(regexp.MustCompile(`Stockfish wins: (\d+)`))
	draws := parseInt(regexp.MustCompile(`Draws: (\d+)`))
	if ngnW < 0 || sfW < 0 || draws < 0 {
		fmt.Println("WARN: could not parse game counts from elo-assess output")
		return "error", nil
	}

	totalPlayed := ngnW + sfW + draws
	score := float64(ngnW) + 0.5*float64(draws)

	// Posterior under a Beta(4,2) prior centered near the prior baseline
	// (~67% win rate). Reported for context only; verdict uses raw counts so
	// the rule is dead simple to reason about.
	pA := 4.0 + score
	pB := 2.0 + float64(totalPlayed) - score
	postMean := pA / (pA + pB)

	fmt.Println("\n== Smoke verdict ==")
	fmt.Printf("Result: %dW %dD %dL  (raw score %.1f/%d)\n", ngnW, draws, sfW, score, totalPlayed)
	fmt.Printf("Beta(4,2) posterior win rate: %.0f%%\n", postMean*100)

	rec := &runRecord{
		Timestamp: time.Now().UTC(),
		SHA:       currentSHA(),
		NgnMs:     ngnMs,
		SfMs:      sfMs,
		TargetELO: targetELO,
		Games:     totalPlayed,
		NgnW:      ngnW,
		Draws:     draws,
		SfW:       sfW,
	}

	switch {
	case ngnW == 0 && sfW >= games-1:
		fmt.Println("REGRESSION — investigate before continuing")
		return "regression", rec
	case ngnW >= games-1 && sfW == 0:
		fmt.Println("IMPROVEMENT — clear wins")
		return "improvement", rec
	default:
		fmt.Println("NEUTRAL — no clear signal at this sample size")
		return "neutral", rec
	}
}

// appendRun appends a single record to the JSONL history file, creating the
// parent dir if needed.
func appendRun(path string, rec runRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// loadHistory reads all run records from the JSONL file. Missing file returns
// (nil, nil) — the caller treats that as "no history yet".
func loadHistory(path string) ([]runRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []runRecord
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scan.Scan() {
		var r runRecord
		if err := json.Unmarshal(scan.Bytes(), &r); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, scan.Err()
}

// printRolling reports the rolling estimate restricted to runs at the same
// (targetELO, ngnMs, sfMs) configuration. Different configs are not directly
// comparable so we don't pool them.
func printRolling(historyPath string, targetELO, ngnMs, sfMs int) {
	all, err := loadHistory(historyPath)
	if err != nil {
		fmt.Printf("WARN: could not load history %s: %v\n", historyPath, err)
		return
	}
	var matching []runRecord
	for _, r := range all {
		if r.TargetELO == targetELO && r.NgnMs == ngnMs && r.SfMs == sfMs {
			matching = append(matching, r)
		}
	}
	if len(matching) == 0 {
		fmt.Println("\n== Rolling ELO ==")
		fmt.Printf("(no prior runs at NGN %dms vs SF%d %dms)\n", ngnMs, targetELO, sfMs)
		return
	}

	var w, d, l, total int
	for _, r := range matching {
		w += r.NgnW
		d += r.Draws
		l += r.SfW
		total += r.Games
	}
	score := float64(w) + 0.5*float64(d)
	est, lo, hi := eloEstimate(score, total, targetELO)

	fmt.Printf("\n== Rolling ELO (%d games across %d runs at NGN %dms vs SF%d %dms) ==\n",
		total, len(matching), ngnMs, targetELO, sfMs)
	fmt.Printf("Cumulative: %dW %dD %dL = %.1f%%\n", w, d, l, score/float64(total)*100)
	fmt.Printf("Estimated ELO: %.0f  CI [%.0f, %.0f]  (width %.0f)\n",
		est, lo, hi, hi-lo)

	// Show the last few runs for trend visibility.
	const recentN = 6
	start := len(matching) - recentN
	if start < 0 {
		start = 0
	}
	fmt.Println("Recent runs (most recent last):")
	for _, r := range matching[start:] {
		ts := r.Timestamp.Local().Format("Mon 15:04")
		runScore := float64(r.NgnW) + 0.5*float64(r.Draws)
		runRate := runScore / float64(r.Games)
		fmt.Printf("  %s  %s  %dW %dD %dL = %.0f%%\n",
			ts, shortSHA(r.SHA), r.NgnW, r.Draws, r.SfW, runRate*100)
	}
}

func currentSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func shortSHA(sha string) string {
	if len(sha) >= 7 {
		return sha[:7]
	}
	return sha
}

// Wilson CI on a Bernoulli sample (with score = wins + 0.5*draws used as the
// "successes" count — this is the standard chess-Elo trick: it overestimates
// variance slightly vs a two-parameter draw model, but is the right thing to
// do given typical engine-vs-engine sample sizes).
func wilsonCI(score float64, n int) (lo, hi float64) {
	const z = 1.96
	if n == 0 {
		return 0, 1
	}
	p := score / float64(n)
	nf := float64(n)
	denom := 1.0 + z*z/nf
	center := (p + z*z/(2*nf)) / denom
	margin := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / denom
	lo = center - margin
	hi = center + margin
	if lo < 0 {
		lo = 0
	}
	if hi > 1 {
		hi = 1
	}
	return
}

func scoreToEloDiff(p float64) float64 {
	const eps = 1e-4
	if p <= eps {
		return -800
	}
	if p >= 1-eps {
		return 800
	}
	return 400 * math.Log10(p/(1-p))
}

func eloEstimate(score float64, n, targetELO int) (est, lo, hi float64) {
	if n == 0 {
		return float64(targetELO), float64(targetELO) - 400, float64(targetELO) + 400
	}
	p := score / float64(n)
	pLo, pHi := wilsonCI(score, n)
	return float64(targetELO) + scoreToEloDiff(p),
		float64(targetELO) + scoreToEloDiff(pLo),
		float64(targetELO) + scoreToEloDiff(pHi)
}
