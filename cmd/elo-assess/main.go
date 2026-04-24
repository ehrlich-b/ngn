package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Assessment levels
const (
	Quick    = 1  // 1 minute - 4 games at 0.5s/move
	Standard = 5  // 5 minutes - 10 games at 1s/move
	Thorough = 10 // 10 minutes - 20 games at 1.5s/move
)

type Config struct {
	NGNPath       string
	StockfishPath string
	Level         int
	Verbose       bool
	TargetELO     int
	MinGames      int
	MaxGames      int
	MovetimeMs    int
	StopWidthELO  float64 // stop early when 95% ELO CI width ≤ this
	LowPower      bool    // route to macOS E-cores via taskpolicy -b
}

type GameResult struct {
	NGNWins    int
	SFWins     int
	Draws      int
	TotalGames int
}

func main() {
	level := flag.Int("level", 0, "Assessment level (1/5/10). 0 = use explicit flags below.")
	verbose := flag.Bool("v", false, "Verbose output")
	targetELO := flag.Int("elo", 1320, "Stockfish ELO target")
	games := flag.Int("games", 0, "Override max games (0 = use level default)")
	movetimeMs := flag.Int("movetime", 0, "ms per move (0 = use level default)")
	minGames := flag.Int("min", 6, "Minimum games before early-stop can fire")
	stopWidth := flag.Float64("ci", 200, "Stop when 95% ELO CI width ≤ this (0 = never early-stop)")
	lowPower := flag.Bool("lowpower", true, "Route engines to macOS E-cores (taskpolicy -b)")
	flag.Parse()

	// Level presets kept for backward compat. Lighter defaults than before.
	var maxGames, moveMs int
	switch *level {
	case 1:
		maxGames, moveMs = 8, 250
	case 5:
		maxGames, moveMs = 20, 400
	case 10:
		maxGames, moveMs = 40, 600
	default:
		maxGames, moveMs = 24, 250 // sensible cool default
	}
	if *games > 0 {
		maxGames = *games
	}
	if *movetimeMs > 0 {
		moveMs = *movetimeMs
	}
	if *minGames > maxGames {
		*minGames = maxGames
	}

	config := &Config{
		NGNPath:       "./build/ngn",
		StockfishPath: "./stockfish/stockfish-ubuntu-x86-64-avx2",
		Level:         *level,
		Verbose:       *verbose,
		TargetELO:     *targetELO,
		MinGames:      *minGames,
		MaxGames:      maxGames,
		MovetimeMs:    moveMs,
		StopWidthELO:  *stopWidth,
		LowPower:      *lowPower,
	}

	fmt.Printf("NGN ELO Assessment (Bayesian early-stop)\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Stockfish target: %d ELO | movetime: %dms\n", config.TargetELO, config.MovetimeMs)
	fmt.Printf("Games: min=%d max=%d | stop at 95%% ELO CI ≤ %.0f\n",
		config.MinGames, config.MaxGames, config.StopWidthELO)
	if config.LowPower && runtime.GOOS == "darwin" {
		fmt.Printf("Power: taskpolicy -b (E-cores, low power)\n")
	} else {
		fmt.Printf("Power: normal\n")
	}
	fmt.Println()

	result := runGamesBayesian(config)
	displayResults(result, config.TargetELO)
}

// runGamesBayesian plays games until CI is tight enough or max games reached.
// Colors alternate each game.
func runGamesBayesian(config *Config) GameResult {
	result := GameResult{}
	movetime := float64(config.MovetimeMs) / 1000.0

	for g := 1; g <= config.MaxGames; g++ {
		fmt.Printf("G%2d ", g)
		ngnWhite := g%2 == 1

		start := time.Now()
		winner, reason := playGame(config, ngnWhite, movetime)
		elapsed := time.Since(start).Truncate(time.Second)

		color := "B"
		if ngnWhite {
			color = "W"
		}

		switch winner {
		case "ngn":
			result.NGNWins++
		case "stockfish":
			result.SFWins++
		case "draw":
			result.Draws++
		}
		result.TotalGames++

		score := float64(result.NGNWins) + 0.5*float64(result.Draws)
		eloEst, eloLo, eloHi := eloEstimate(score, result.TotalGames, config.TargetELO)
		ciWidth := eloHi - eloLo

		fmt.Printf("[%s] %-10s %-12s | W%d D%d L%d  est %.0f  CI [%.0f, %.0f]  (%v)\n",
			color, winner, reason,
			result.NGNWins, result.Draws, result.SFWins,
			eloEst, eloLo, eloHi, elapsed)

		// Early stop: CI tight enough AND past min games.
		if config.StopWidthELO > 0 &&
			result.TotalGames >= config.MinGames &&
			!math.IsInf(ciWidth, 0) &&
			ciWidth <= config.StopWidthELO {
			fmt.Printf("\nEarly stop: ELO CI width %.0f ≤ %.0f after %d games\n",
				ciWidth, config.StopWidthELO, result.TotalGames)
			break
		}
	}
	return result
}

// wilsonCI returns the Wilson 95% confidence interval for expected score-per-game
// given total score (wins + 0.5*draws) over n games. Robust at edges unlike
// the Normal approximation.
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

// eloEstimate returns (point estimate, lower CI, upper CI) in absolute ELO
// against an opponent rated targetELO. Uses Wilson CI on score fraction.
func eloEstimate(score float64, n, targetELO int) (est, lo, hi float64) {
	if n == 0 {
		return float64(targetELO), float64(targetELO) - 400, float64(targetELO) + 400
	}
	p := score / float64(n)
	pLo, pHi := wilsonCI(score, n)
	est = float64(targetELO) + scoreToEloDiff(p)
	// Higher p → higher ELO; CI mapping preserves that order.
	lo = float64(targetELO) + scoreToEloDiff(pLo)
	hi = float64(targetELO) + scoreToEloDiff(pHi)
	return
}

func scoreToEloDiff(p float64) float64 {
	// Clamp to avoid ±Inf when all games lost/won; use sentinel bounds.
	const eps = 1e-4
	if p <= eps {
		return -800
	}
	if p >= 1-eps {
		return 800
	}
	return 400 * math.Log10(p/(1-p))
}

func playGame(config *Config, ngnWhite bool, timePerMove float64) (string, string) {
	ngn := startEngine(config.NGNPath, "NGN", config.LowPower)
	if ngn == nil {
		return "error", "failed to start NGN"
	}
	defer stopEngine(ngn)

	sf := startEngine(config.StockfishPath, "Stockfish", config.LowPower)
	if sf == nil {
		return "error", "failed to start Stockfish"
	}
	defer stopEngine(sf)

	sendCommand(sf, "setoption name UCI_LimitStrength value true")
	sendCommand(sf, fmt.Sprintf("setoption name UCI_Elo value %d", config.TargetELO))

	moves := []string{}
	maxMoves := 200

	for moveNum := 0; moveNum < maxMoves; moveNum++ {
		isWhiteMove := moveNum%2 == 0
		var currentEngine *Engine

		if (isWhiteMove && ngnWhite) || (!isWhiteMove && !ngnWhite) {
			currentEngine = ngn
		} else {
			currentEngine = sf
		}

		posCmd := "position startpos"
		if len(moves) > 0 {
			posCmd += " moves " + strings.Join(moves, " ")
		}
		sendCommand(currentEngine, posCmd)

		goCmd := fmt.Sprintf("go movetime %d", int(timePerMove*1000))
		move := getMove(currentEngine, goCmd, timePerMove+2)

		if move == "" || move == "(none)" {
			// Game over — side to move can't move. If it's NGN, NGN lost.
			if currentEngine == ngn {
				return "stockfish", "checkmate"
			}
			return "ngn", "checkmate"
		}

		moves = append(moves, move)
		if config.Verbose {
			fmt.Printf("  %d. %s\n", len(moves), move)
		}
	}

	return "draw", "max_moves"
}

type Engine struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
}

func startEngine(path, name string, lowPower bool) *Engine {
	var cmd *exec.Cmd
	if lowPower && runtime.GOOS == "darwin" {
		// taskpolicy -b sets "background" QoS → Apple Silicon E-cores.
		// Dramatic power/heat reduction vs P-cores for modest throughput cost.
		cmd = exec.Command("taskpolicy", "-b", path)
	} else {
		cmd = exec.Command(path)
	}
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start %s: %v", name, err)
		return nil
	}

	engine := &Engine{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdin),
		stdout: bufio.NewScanner(stdout),
	}

	sendCommand(engine, "uci")
	waitFor(engine, "uciok", 5*time.Second)
	sendCommand(engine, "isready")
	waitFor(engine, "readyok", 5*time.Second)

	return engine
}

func sendCommand(engine *Engine, cmd string) {
	engine.stdin.WriteString(cmd + "\n")
	engine.stdin.Flush()
}

func waitFor(engine *Engine, expected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if engine.stdout.Scan() {
			if strings.Contains(engine.stdout.Text(), expected) {
				return true
			}
		}
	}
	return false
}

func getMove(engine *Engine, goCmd string, timeout float64) string {
	sendCommand(engine, goCmd)

	deadline := time.Now().Add(time.Duration(timeout * float64(time.Second)))
	for time.Now().Before(deadline) {
		if engine.stdout.Scan() {
			line := engine.stdout.Text()
			if strings.HasPrefix(line, "bestmove ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return parts[1]
				}
			}
		}
	}
	return ""
}

func stopEngine(engine *Engine) {
	sendCommand(engine, "quit")
	time.Sleep(100 * time.Millisecond)
	engine.cmd.Process.Kill()
}

func displayResults(result GameResult, targetELO int) {
	fmt.Println()
	fmt.Println("=== RESULTS ===")
	fmt.Printf("Games played: %d\n", result.TotalGames)
	fmt.Printf("NGN wins: %d\n", result.NGNWins)
	fmt.Printf("Stockfish wins: %d\n", result.SFWins)
	fmt.Printf("Draws: %d\n", result.Draws)

	if result.TotalGames == 0 {
		return
	}
	score := float64(result.NGNWins) + 0.5*float64(result.Draws)
	p := score / float64(result.TotalGames)
	est, lo, hi := eloEstimate(score, result.TotalGames, targetELO)

	fmt.Printf("\nScore: %.1f/%d  (%.1f%%)\n", score, result.TotalGames, p*100)
	fmt.Printf("Estimated NGN ELO: %.0f   95%% CI: [%.0f, %.0f]  (width %.0f)\n",
		est, lo, hi, hi-lo)

	// Interpretive hint
	switch {
	case lo > float64(targetELO)+50:
		fmt.Printf("Verdict: likely stronger than SF %d (raise target)\n", targetELO)
	case hi < float64(targetELO)-50:
		fmt.Printf("Verdict: likely weaker than SF %d (lower target)\n", targetELO)
	default:
		fmt.Printf("Verdict: roughly matched with SF %d\n", targetELO)
	}
}
