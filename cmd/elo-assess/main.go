package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os/exec"
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
	Level         int  // 1, 5, or 10 minutes
	Verbose       bool
	TargetELO     int  // Starting target ELO for calibration
}

type GameResult struct {
	NGNWins    int
	SFWins     int
	Draws      int
	TotalGames int
}

func main() {
	level := flag.Int("level", 1, "Assessment level: 1 (quick), 5 (standard), 10 (thorough) minutes")
	verbose := flag.Bool("v", false, "Verbose output")
	targetELO := flag.Int("elo", 1320, "Starting Stockfish ELO target")
	flag.Parse()

	config := &Config{
		NGNPath:       "./build/ngn",
		StockfishPath: "./stockfish/stockfish-ubuntu-x86-64-avx2",
		Level:         *level,
		Verbose:       *verbose,
		TargetELO:     *targetELO,
	}

	fmt.Printf("NGN ELO Assessment Tool\n")
	fmt.Printf("=======================\n")
	fmt.Printf("Level: %d minute(s)\n", config.Level)
	fmt.Printf("Target Stockfish ELO: %d\n\n", config.TargetELO)

	// Determine game parameters based on level
	var numGames int
	var timePerMove float64

	switch config.Level {
	case 1:
		numGames = 4
		timePerMove = 0.5
		fmt.Println("Quick assessment: 4 games @ 0.5s/move (~1 min)")
	case 5:
		numGames = 10
		timePerMove = 1.0
		fmt.Println("Standard assessment: 10 games @ 1s/move (~5 min)")
	case 10:
		numGames = 20
		timePerMove = 1.5
		fmt.Println("Thorough assessment: 20 games @ 1.5s/move (~10 min)")
	default:
		numGames = 4
		timePerMove = 0.5
		fmt.Println("Using quick assessment (default)")
	}

	fmt.Println()

	// Run games
	result := runGames(config, numGames, timePerMove)

	// Calculate and display results
	displayResults(result, config.TargetELO)
}

func runGames(config *Config, numGames int, timePerMove float64) GameResult {
	result := GameResult{TotalGames: numGames}

	for gameNum := 1; gameNum <= numGames; gameNum++ {
		fmt.Printf("Game %d/%d... ", gameNum, numGames)

		// Alternate colors
		ngnWhite := gameNum%2 == 1

		winner, reason := playGame(config, ngnWhite, timePerMove)

		switch winner {
		case "ngn":
			result.NGNWins++
			fmt.Printf("NGN wins (%s)\n", reason)
		case "stockfish":
			result.SFWins++
			fmt.Printf("Stockfish wins (%s)\n", reason)
		case "draw":
			result.Draws++
			fmt.Printf("Draw (%s)\n", reason)
		default:
			fmt.Printf("Unknown result: %s\n", reason)
		}
	}

	return result
}

func playGame(config *Config, ngnWhite bool, timePerMove float64) (string, string) {
	// Start engines
	ngn := startEngine(config.NGNPath, "NGN")
	if ngn == nil {
		return "error", "failed to start NGN"
	}
	defer stopEngine(ngn)

	sf := startEngine(config.StockfishPath, "Stockfish")
	if sf == nil {
		return "error", "failed to start Stockfish"
	}
	defer stopEngine(sf)

	// Configure Stockfish ELO
	sendCommand(sf, fmt.Sprintf("setoption name UCI_LimitStrength value true"))
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

		// Set position
		posCmd := "position startpos"
		if len(moves) > 0 {
			posCmd += " moves " + strings.Join(moves, " ")
		}
		sendCommand(currentEngine, posCmd)

		// Get move
		goCmd := fmt.Sprintf("go movetime %d", int(timePerMove*1000))
		move := getMove(currentEngine, goCmd, timePerMove+2)

		if move == "" || move == "(none)" {
			// Game over - determine result
			if currentEngine == ngn {
				if ngnWhite == isWhiteMove {
					return "stockfish", "checkmate"
				} else {
					return "stockfish", "checkmate"
				}
			} else {
				if ngnWhite == isWhiteMove {
					return "ngn", "checkmate"
				} else {
					return "ngn", "checkmate"
				}
			}
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

func startEngine(path, name string) *Engine {
	cmd := exec.Command(path)
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

	// Initialize UCI
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

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
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

	// Calculate score
	ngnScore := float64(result.NGNWins) + float64(result.Draws)*0.5
	percentage := ngnScore / float64(result.TotalGames) * 100

	fmt.Printf("\nNGN score: %.1f/%.1f (%.1f%%)\n", ngnScore, float64(result.TotalGames), percentage)

	// Estimate ELO using Elo formula
	// Expected score = 1 / (1 + 10^((Ra-Rb)/400))
	// If we get score S against opponent rated R, our rating estimate:
	// Rating = R + 400 * log10(S/(1-S))

	if percentage > 0 && percentage < 100 {
		scoreFraction := ngnScore / float64(result.TotalGames)
		eloDiff := 400 * math.Log10(scoreFraction/(1-scoreFraction))
		estimatedELO := float64(targetELO) + eloDiff

		fmt.Printf("\n=== ELO ESTIMATE ===\n")
		fmt.Printf("Against Stockfish %d ELO:\n", targetELO)
		fmt.Printf("Estimated NGN ELO: %.0f (±100)\n", estimatedELO)

		// Confidence note
		if result.TotalGames < 10 {
			fmt.Println("\nNote: Low sample size. Use 'standard' or 'thorough' for better accuracy.")
		}
	} else if percentage == 100 {
		fmt.Printf("\n=== ELO ESTIMATE ===\n")
		fmt.Printf("NGN won all games! Try higher --elo target.\n")
		fmt.Printf("Estimated NGN ELO: >%d\n", targetELO+200)
	} else {
		fmt.Printf("\n=== ELO ESTIMATE ===\n")
		fmt.Printf("NGN lost all games. Try lower --elo target.\n")
		fmt.Printf("Estimated NGN ELO: <%d\n", targetELO-200)
	}
}
