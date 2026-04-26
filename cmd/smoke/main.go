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
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

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
	flag.Parse()

	fmt.Println("NGN smoke test")
	fmt.Println("==============")
	fail := false

	if !*skipTactical {
		if !runTactical(*epdPath, *nTactical, *tMs, *threshold, *seed) {
			fail = true
		}
	}

	if !*skipGames && *nGames > 0 {
		if v := runGameSmoke(*nGames, *ngnMs, *sfMs, *targetELO); v == "regression" {
			fail = true
		}
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
// returns one of "regression", "neutral", "improvement", "error".
func runGameSmoke(games, ngnMs, sfMs, targetELO int) string {
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
		return "error"
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
		return "error"
	}

	score := float64(ngnW) + 0.5*float64(draws)
	total := float64(ngnW + sfW + draws)

	// Posterior under a Beta(4,2) prior centered near the prior baseline
	// (~67% win rate). Reported for context only; verdict uses raw counts so
	// the rule is dead simple to reason about.
	pA := 4.0 + score
	pB := 2.0 + (total - score)
	postMean := pA / (pA + pB)

	fmt.Println("\n== Smoke verdict ==")
	fmt.Printf("Result: %dW %dD %dL  (raw score %.1f/%d)\n", ngnW, draws, sfW, score, int(total))
	fmt.Printf("Beta(4,2) posterior win rate: %.0f%%\n", postMean*100)

	switch {
	case ngnW == 0 && sfW >= games-1:
		// Lost all (or all-but-one to a draw). Very likely a regression.
		fmt.Println("REGRESSION — investigate before continuing")
		return "regression"
	case ngnW >= games-1 && sfW == 0:
		fmt.Println("IMPROVEMENT — clear wins")
		return "improvement"
	default:
		fmt.Println("NEUTRAL — no clear signal at this sample size")
		return "neutral"
	}
}
