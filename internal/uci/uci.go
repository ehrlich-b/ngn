// Package uci drives UCI chess-engine subprocesses for self-play and gauntlet
// measurement. It is the hardened harness extracted from cmd/sprt: a retried
// handshake, a warmup move probe, and a UCI "stop" sent when an opponent
// overshoots its movetime budget. Without these, a slow or glitchy external
// engine (Blunder searched 6.6s for an 800ms request) gets forfeited as
// "no-move" and produces a bogus +800 result — keep this the ONE copy.
package uci

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

// StartFEN is the standard chess starting position.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Outcome is a game result from the NEW engine's point of view.
type Outcome int

const (
	Loss Outcome = iota
	Draw
	Win
)

// GameResult is one game's outcome plus a short reason (checkmate, draw-rule,
// no-move, ...) for reporting and integrity checks.
type GameResult struct {
	Res    Outcome
	Reason string
}

// GameConfig is the per-game search budget. Real-clock (TCTimeMs), then Nodes,
// then Movetime, then depth is used (in that order). NewDepth/BaseDepth let the
// sides search to different depths (a positive control). MaxMoves caps plies
// before an adjudicated draw. TCTimeMs/TCIncMs select a real tournament clock
// (the I4 instrument): each side gets TCTimeMs to start +TCIncMs per move, the
// harness sends `go wtime/btime/winc/binc` and a side that burns its bank loses
// on time ("time-forfeit") — the ONLY mode that exercises NGN's Tournament time
// management and can see flag-outs.
type GameConfig struct {
	NewDepth, BaseDepth, Movetime, Nodes, MaxMoves int
	TCTimeMs, TCIncMs                              int
}

// Engine is a persistent UCI subprocess.
type Engine struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
	// Moves/Overshoots track movetime-mode compliance for the gauntlet's integrity
	// guard; Overshoots counts moves that arrived well after the `stop` deadline
	// (the engine ignored `stop`). Reset per anchor via ResetCounters. Touched only
	// by the single goroutine running the game, so no synchronization is needed.
	Moves      int
	Overshoots int
}

// overshootGrace is how long past the stop deadline a move may arrive before we
// count it as an overshoot. Generous enough to absorb OS/scheduling jitter on
// throttled E-cores, tight enough to flag a multi-second `stop`-ignore.
const overshootGrace = 2 * time.Second

// stopGrace is how long past the movetime budget we wait before forcing `stop`.
// Small on purpose: an engine that ignores `go movetime` and only stops here should
// burn ~the same wall clock as one that self-limits to the budget, so neither side
// gets a time-odds edge (a movetime-honoring engine returns before this fires).
const stopGrace = 200 * time.Millisecond

// tcFlagGrace is how far past its remaining bank a move may arrive in real-clock
// mode before it counts as flagging. Absorbs measurement/IPC/scheduling jitter on
// throttled E-cores (a well-behaved engine reserves its own move-overhead and
// returns BEFORE the bank empties); a real flag overshoots by far more than this.
const tcFlagGrace = 300 * time.Millisecond

// Start launches the engine at path, completes the UCI handshake, and verifies it
// actually EMITS a move (warmup probe), retrying up to 3× on failure. Returns nil
// if it never comes up — callers MUST treat nil as fatal rather than play games
// (a dead engine forfeits every game as "no-move" → bogus +800).
func Start(path, name string, lowPower bool) *Engine {
	for attempt := 1; attempt <= 3; attempt++ {
		var cmd *exec.Cmd
		if lowPower && runtime.GOOS == "darwin" {
			cmd = exec.Command("taskpolicy", "-b", path)
		} else {
			cmd = exec.Command(path)
		}
		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			log.Printf("failed to start %s (%s): %v", name, path, err)
			return nil
		}
		e := &Engine{cmd: cmd, stdin: bufio.NewWriter(stdin), stdout: bufio.NewScanner(stdout)}
		e.stdout.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		Send(e, "uci")
		if !WaitFor(e, "uciok", 15*time.Second) {
			log.Printf("%s (%s): no uciok within 15s (attempt %d/3)", name, path, attempt)
			Stop(e)
			continue
		}
		Send(e, "isready")
		if !WaitFor(e, "readyok", 15*time.Second) {
			log.Printf("%s (%s): no readyok within 15s (attempt %d/3)", name, path, attempt)
			Stop(e)
			continue
		}
		// Warmup probe: confirm the engine EMITS a move, not just that it handshakes.
		// The observed failure was a process answering uci/isready but producing no
		// bestmove, forfeiting every game as "no-move".
		Send(e, "position startpos")
		if mv := GetMove(e, "go movetime 200", 10*time.Second, 1500*time.Millisecond); mv == "" {
			log.Printf("%s (%s): handshakes but emitted no warmup move (attempt %d/3)", name, path, attempt)
			Stop(e)
			continue
		}
		return e
	}
	return nil
}

// Reset issues ucinewgame between games.
func Reset(e *Engine) {
	Send(e, "ucinewgame")
	Send(e, "isready")
	WaitFor(e, "readyok", 5*time.Second)
}

// Send writes a single UCI command line to the engine.
func Send(e *Engine, cmd string) {
	e.stdin.WriteString(cmd + "\n")
	e.stdin.Flush()
}

// WaitFor scans stdout until a line contains expected or the timeout elapses.
func WaitFor(e *Engine, expected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if e.stdout.Scan() {
			if strings.Contains(e.stdout.Text(), expected) {
				return true
			}
		}
	}
	return false
}

// GetMove sends goCmd and returns the bestmove. If stopAfter > 0 and no bestmove
// has arrived by then, it sends UCI "stop" to force one — covering opponents that
// ignore/overshoot their movetime budget — then waits up to timeout for the reply.
func GetMove(e *Engine, goCmd string, timeout, stopAfter time.Duration) string {
	Send(e, goCmd)
	start := time.Now()
	deadline := start.Add(timeout)
	var stopAt time.Time
	if stopAfter > 0 {
		stopAt = start.Add(stopAfter)
	}
	stopSent := false
	for time.Now().Before(deadline) {
		if stopAfter > 0 && !stopSent && time.Now().After(stopAt) {
			Send(e, "stop")
			stopSent = true
		}
		if e.stdout.Scan() {
			line := e.stdout.Text()
			if strings.HasPrefix(line, "bestmove ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					// Movetime-compliance bookkeeping: a move that arrives well past
					// the stop deadline means the engine ignored `stop` (older Blunder),
					// so the gauntlet can skip it instead of measuring a distorted TC.
					if stopAfter > 0 {
						e.Moves++
						if time.Since(start) > stopAfter+overshootGrace {
							e.Overshoots++
						}
					}
					return parts[1]
				}
			}
		}
	}
	return ""
}

// ResetCounters zeroes the movetime-compliance counters; call it per anchor after
// the warmup/probe moves so only the rated games feed the integrity guard.
func ResetCounters(e *Engine) {
	e.Moves = 0
	e.Overshoots = 0
}

// RespectsMovetime probes whether the engine returns a move near the requested
// movetime, sending `stop` exactly as a real game does. Engines that ignore both
// `go movetime` and `stop` (older Blunder) blow past the budget and the stop
// grace; the gauntlet skips them rather than silently measure at a distorted,
// over-clocked time control. Returns whether it complied and the observed time.
func RespectsMovetime(e *Engine, movetimeMs int) (bool, time.Duration) {
	budget := time.Duration(movetimeMs) * time.Millisecond
	stopAfter := budget + stopGrace
	Send(e, "position startpos")
	start := time.Now()
	mv := GetMove(e, fmt.Sprintf("go movetime %d", movetimeMs), budget+30*time.Second, stopAfter)
	took := time.Since(start)
	return mv != "" && took <= stopAfter+overshootGrace, took
}

// Stop quits the engine and kills the process.
func Stop(e *Engine) {
	Send(e, "quit")
	time.Sleep(50 * time.Millisecond)
	if e.cmd.Process != nil {
		e.cmd.Process.Kill()
	}
}

// PowerNote describes the core-affinity routing, for display.
func PowerNote(lowPower bool) string {
	if lowPower && runtime.GOOS == "darwin" {
		return " | E-cores (taskpolicy -b)"
	}
	return ""
}

// SideToMoveIsNew reports whether the NEW engine is to move.
func SideToMoveIsNew(pos *engine.Position, newIsWhite bool) bool {
	return (pos.Turn() == engine.White) == newIsWhite
}

// PlayGame plays one game from the start position through the opening line,
// alternating the two engines by color, and returns the result from NEW's point
// of view. Game ends are detected authoritatively via the engine package.
func PlayGame(newEng, baseEng *Engine, opening []string, newIsWhite bool, cfg GameConfig) GameResult {
	Reset(newEng)
	Reset(baseEng)

	pos, err := engine.ParseFEN(StartFEN)
	if err != nil {
		return GameResult{Draw, "fen-error"}
	}
	if pos.Positions == nil {
		pos.Positions = make(map[uint64]int)
	}
	pos.Positions[pos.Hash()]++

	moves := make([]string, 0, cfg.MaxMoves)
	for _, mv := range opening {
		m, err := engine.ParseUCIMove(pos, mv)
		if err != nil {
			return GameResult{Draw, "bad-opening"}
		}
		if _, _, _, ok := pos.GameMakeMove(m); !ok {
			return GameResult{Draw, "illegal-opening"}
		}
		moves = append(moves, mv)
	}

	// Real-clock bank per side (ms), used only when cfg.TCTimeMs > 0. Openings are
	// free (played above before the clock starts), as in normal play.
	whiteMs, blackMs := cfg.TCTimeMs, cfg.TCTimeMs

	for len(moves) < cfg.MaxMoves {
		if legal := engine.GenerateLegalMoves(pos); len(legal) == 0 {
			if pos.IsInCheck() {
				// Side to move is checkmated; the side that just moved won.
				if !SideToMoveIsNew(pos, newIsWhite) {
					return GameResult{Win, "checkmate"}
				}
				return GameResult{Loss, "checkmate"}
			}
			return GameResult{Draw, "stalemate"}
		}
		if pos.IsFIDEDrawRule() {
			return GameResult{Draw, "draw-rule"}
		}

		newToMove := SideToMoveIsNew(pos, newIsWhite)
		eng, d := baseEng, cfg.BaseDepth
		if newToMove {
			eng, d = newEng, cfg.NewDepth
		}
		Send(eng, "position startpos moves "+strings.Join(moves, " "))
		var goCmd string
		var timeout, stopAfter time.Duration
		if cfg.TCTimeMs > 0 {
			// Real tournament clock: hand the engine both banks + increment and let
			// its own time management decide. stopAfter forces a hung/over-spending
			// engine to yield just past its remaining bank so we can record the flag.
			myMs := blackMs
			if pos.Turn() == engine.White {
				myMs = whiteMs
			}
			goCmd = fmt.Sprintf("go wtime %d btime %d winc %d binc %d", whiteMs, blackMs, cfg.TCIncMs, cfg.TCIncMs)
			stopAfter = time.Duration(myMs)*time.Millisecond + tcFlagGrace
			timeout = time.Duration(myMs)*time.Millisecond + 30*time.Second
		} else if cfg.Nodes > 0 {
			goCmd = fmt.Sprintf("go nodes %d", cfg.Nodes)
			timeout = 60 * time.Second
		} else if cfg.Movetime > 0 {
			goCmd = fmt.Sprintf("go movetime %d", cfg.Movetime)
			// Force-stop at budget + a small grace so an engine ignoring movetime
			// still searches ~the same wall time as a movetime-honoring one (equal
			// TC); allow +30s hard for it to actually answer the "stop".
			stopAfter = time.Duration(cfg.Movetime)*time.Millisecond + stopGrace
			timeout = time.Duration(cfg.Movetime)*time.Millisecond + 30*time.Second
		} else {
			goCmd = fmt.Sprintf("go depth %d", d)
			timeout = 180 * time.Second
		}
		moveStart := time.Now()
		mv := GetMove(eng, goCmd, timeout, stopAfter)
		if cfg.TCTimeMs > 0 {
			// Deduct wall time from the mover's bank; an overshoot past the grace is a
			// flag (loss on time) and ends the game before the move is even applied.
			rem := &blackMs
			if pos.Turn() == engine.White {
				rem = &whiteMs
			}
			*rem -= int(time.Since(moveStart).Milliseconds())
			if *rem < -int(tcFlagGrace.Milliseconds()) {
				if newToMove {
					return GameResult{Loss, "time-forfeit"}
				}
				return GameResult{Win, "time-forfeit"}
			}
			*rem += cfg.TCIncMs // Fischer increment for the completed move
		}
		if mv == "" || mv == "(none)" {
			// Engine failed to produce a move though legal moves exist: count it
			// as a loss for the offending side so a flaky binary can't masquerade.
			if newToMove {
				return GameResult{Loss, "no-move"}
			}
			return GameResult{Win, "no-move"}
		}
		m, err := engine.ParseUCIMove(pos, mv)
		if err != nil {
			if newToMove {
				return GameResult{Loss, "illegal-move"}
			}
			return GameResult{Win, "illegal-move"}
		}
		if _, _, _, ok := pos.GameMakeMove(m); !ok {
			if newToMove {
				return GameResult{Loss, "illegal-move"}
			}
			return GameResult{Win, "illegal-move"}
		}
		moves = append(moves, mv)
	}
	return GameResult{Draw, "max-moves"}
}

// builtinOpenings are standard, roughly-balanced lines to ~ply 5 as UCI move
// sequences from the start position; each is played twice (colors reversed).
var builtinOpenings = []string{
	"e2e4 e7e5 g1f3 b8c6 f1b5", // Ruy Lopez
	"e2e4 e7e5 g1f3 b8c6 f1c4", // Italian
	"e2e4 e7e5 g1f3 b8c6 d2d4", // Scotch
	"e2e4 e7e5 g1f3 g8f6",      // Petrov
	"e2e4 e7e5 f1c4",           // Bishop's Opening
	"e2e4 c7c5 g1f3 d7d6",      // Sicilian Najdorf-ish
	"e2e4 c7c5 g1f3 b8c6",      // Sicilian
	"e2e4 c7c5 b1c3",           // Closed Sicilian
	"e2e4 c7c5 c2c3",           // Alapin
	"e2e4 e7e6 d2d4 d7d5",      // French
	"e2e4 c7c6 d2d4 d7d5",      // Caro-Kann
	"e2e4 d7d6 d2d4 g8f6",      // Pirc
	"e2e4 g7g6 d2d4 f8g7",      // Modern
	"d2d4 d7d5 c2c4 e7e6",      // QGD
	"d2d4 d7d5 c2c4 c7c6",      // Slav
	"d2d4 d7d5 g1f3 g8f6",      // symmetric d4
	"d2d4 g8f6 c2c4 e7e6",      // Nimzo/QID
	"d2d4 g8f6 c2c4 g7g6",      // KID/Grunfeld
	"d2d4 f7f5",                // Dutch
	"d2d4 e7e6",                // various
	"c2c4 e7e5",                // English reversed Sicilian
	"c2c4 c7c5",                // Symmetric English
	"g1f3 d7d5 g2g3",           // Reti/KIA
	"g1f3 g8f6 c2c4",           // English/Indian
}

// LoadOpenings reads opening move-sequences (one per line) from path, or returns
// the built-in set when path is empty. Illegal lines are skipped with a warning.
func LoadOpenings(path string) ([][]string, error) {
	lines := builtinOpenings
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lines = strings.Split(string(data), "\n")
	}
	var out [][]string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		mvs := strings.Fields(ln)
		if validOpening(mvs) {
			out = append(out, mvs)
		} else {
			fmt.Printf("WARNING: skipping illegal opening line: %s\n", ln)
		}
	}
	return out, nil
}

func validOpening(mvs []string) bool {
	pos, err := engine.ParseFEN(StartFEN)
	if err != nil {
		return false
	}
	for _, mv := range mvs {
		m, err := engine.ParseUCIMove(pos, mv)
		if err != nil {
			return false
		}
		if _, _, _, ok := pos.GameMakeMove(m); !ok {
			return false
		}
	}
	return len(mvs) > 0
}
