package engine

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// runUCI drives the full UCI loop (the production Run entry point) with an
// in-memory command stream and returns everything the engine wrote. It reads the
// buffer only AFTER Run returns, which — because Run joins the search goroutine on
// exit — is race-clean (verify with `go test -race`). A hung loop (quit failing to
// terminate) trips the timeout.
func runUCI(t *testing.T, commands string) string {
	t.Helper()
	engine := NewUCIEngine()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		engine.Run(strings.NewReader(commands), &buf)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Run did not return within 15s — quit/EOF failed to terminate the loop")
	}
	return buf.String()
}

// TestUCIRunQuitTerminatesLoop proves `quit` breaks the scanner loop rather than
// relying on input EOF. The command AFTER quit must NOT be processed: if the loop
// kept scanning it would handle the trailing "uci" and emit an id banner.
func TestUCIRunQuitTerminatesLoop(t *testing.T) {
	out := runUCI(t, "isready\nquit\nuci\n")
	if !strings.Contains(out, "readyok") {
		t.Errorf("expected readyok from the pre-quit isready, got:\n%s", out)
	}
	if strings.Contains(out, "id name") {
		t.Errorf("a command after quit was processed — quit did not terminate the loop:\n%s", out)
	}
}

// TestUCIRunImmediateStop: stop right after go must produce a bestmove and be
// race-clean (the join in handleStop guarantees the searcher is done).
func TestUCIRunImmediateStop(t *testing.T) {
	out := runUCI(t, "uci\nposition startpos\ngo depth 30\nstop\nquit\n")
	if !strings.Contains(out, "uciok") {
		t.Errorf("missing uciok:\n%s", out)
	}
	if !strings.Contains(out, "bestmove") {
		t.Errorf("stop after go did not produce a bestmove:\n%s", out)
	}
}

// TestUCIRunRepeatedGo: a second go while the first search is still running must
// not spawn a concurrent search (the searching CAS rejects it) and must not race.
func TestUCIRunRepeatedGo(t *testing.T) {
	out := runUCI(t, "position startpos\ngo depth 30\ngo depth 30\nstop\nquit\n")
	if n := strings.Count(out, "bestmove"); n < 1 {
		t.Errorf("expected at least one bestmove, got %d:\n%s", n, out)
	}
}

// TestUCIRunPositionAfterStop: changing the position after stop must be race-clean
// — handleStop joins the search goroutine before the loop mutates uci.position, so
// the searcher can never read a position pointer being swapped underneath it.
func TestUCIRunPositionAfterStop(t *testing.T) {
	out := runUCI(t, "position startpos\ngo depth 30\nstop\nposition startpos moves e2e4 e7e5\ngo depth 4\nstop\nquit\n")
	if n := strings.Count(out, "bestmove"); n < 2 {
		t.Errorf("expected two bestmoves (one per search), got %d:\n%s", n, out)
	}
}
