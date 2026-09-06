package engine

import (
	"bytes"
	"io"
	"strings"
	"sync"
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
	if n := strings.Count(out, "bestmove "); n != 1 {
		t.Errorf("stop after go produced %d bestmoves, want exactly one:\n%s", n, out)
	}
}

// TestUCIRunRepeatedGo: a second go while the first search is still running must
// not spawn a concurrent search (the active session rejects it) and must not race.
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
	if n := strings.Count(out, "bestmove "); n != 2 {
		t.Errorf("expected exactly two bestmoves (one per search), got %d:\n%s", n, out)
	}
}

func TestUCIPreparedSessionStopSurvivesUntilSearchEntry(t *testing.T) {
	uci := NewUCIEngine()
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	uci.searcher = searcher
	uci.position, err = ParseFEN("7k/8/8/8/3Q4/8/8/K7 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	params := SearchParams{Depth: 1}
	uci.timeManager.SetTimeControl(params, true)
	uci.timeManager.UpdateGameState(uci.movesPlayed, uci.lastScore, uci.lastScore, uci.position)

	session, accepted := uci.beginSearchSession()
	if !accepted {
		t.Fatal("first search session was rejected")
	}
	t.Cleanup(func() {
		session.cancel(uci.searcher)
		uci.finishSearchSession(session)
	})
	stopDone := make(chan struct{}, 2)
	for range 2 {
		go func() {
			uci.handleStop(io.Discard)
			stopDone <- struct{}{}
		}()
	}
	select {
	case <-session.cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not cancel the published prepared session")
	}
	if !uci.searcher.StopRequested() {
		t.Fatal("prepared session cancellation did not reach search control")
	}
	uci.lifecycleMu.Lock()
	stoppingState, stoppingActive := uci.searchState, uci.activeSearch
	uci.lifecycleMu.Unlock()
	if stoppingState != uciSearchStopping || stoppingActive != session {
		t.Fatalf("prepared stop state=%d active=%p, want stopping/%p", stoppingState, stoppingActive, session)
	}

	// A duplicate go must be rejected without clearing the stop already attached
	// to the published session.
	uci.handleGo([]string{"depth", "1"}, io.Discard)
	if !uci.searcher.StopRequested() {
		t.Fatal("rejected duplicate go cleared the active session stop")
	}

	var output bytes.Buffer
	uci.markSearchRunning(session)
	go uci.search(params, &output, session)
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for range 2 {
		select {
		case <-stopDone:
		case <-deadline.C:
			t.Fatal("idempotent stop callers did not join the prepared search after launch")
		}
	}
	if n := strings.Count(output.String(), "bestmove "); n != 1 {
		t.Fatalf("prepared-stop search produced %d bestmoves, want exactly one:\n%s", n, output.String())
	}
	if uci.isSearching() {
		t.Fatal("joined prepared-stop search remained active")
	}
	uci.lifecycleMu.Lock()
	state, active := uci.searchState, uci.activeSearch
	uci.lifecycleMu.Unlock()
	if state != uciSearchIdle || active != nil {
		t.Fatalf("completed lifecycle state=%d active=%p, want idle/nil", state, active)
	}
}

type blockingBestmoveWriter struct {
	entered     chan struct{}
	release     chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
	buf         bytes.Buffer
}

func (w *blockingBestmoveWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("bestmove ")) {
		w.enterOnce.Do(func() {
			close(w.entered)
			<-w.release
		})
	}
	return w.buf.Write(p)
}

func (w *blockingBestmoveWriter) unblock() {
	w.releaseOnce.Do(func() { close(w.release) })
}

func TestUCICompletionFollowsFinalOutputWithoutHoldingLifecycleLock(t *testing.T) {
	uci := NewUCIEngine()
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	uci.searcher = searcher
	uci.position, err = ParseFEN("7k/8/8/8/3Q4/8/8/K7 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	output := &blockingBestmoveWriter{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() {
		output.unblock()
		uci.searcher.RequestStop()
	})
	uci.handleGo([]string{"depth", "1"}, output)

	select {
	case <-output.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("search did not reach final bestmove write")
	}
	uci.lifecycleMu.Lock()
	session := uci.activeSearch
	uci.lifecycleMu.Unlock()
	if session == nil {
		t.Fatal("session completed while its final output was blocked")
	}
	stopDone := make(chan struct{})
	go func() {
		uci.handleStop(io.Discard)
		close(stopDone)
	}()
	select {
	case <-session.cancelled:
	case <-time.After(5 * time.Second):
		output.unblock()
		t.Fatal("stop did not reach the output-blocked session")
	}
	select {
	case <-stopDone:
		output.unblock()
		t.Fatal("stop returned before the final output write completed")
	default:
	}
	if !uci.lifecycleMu.TryLock() {
		output.unblock()
		t.Fatal("final output or join held lifecycleMu while blocking")
	}
	uci.lifecycleMu.Unlock()

	output.unblock()
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not receive completion after final output")
	}
	if n := strings.Count(output.buf.String(), "bestmove "); n != 1 {
		t.Fatalf("output-blocked search produced %d bestmoves, want one:\n%s", n, output.buf.String())
	}
}
