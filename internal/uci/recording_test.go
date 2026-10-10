package uci

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// A separate helper process verifies the real pipe, shutdown and timer paths.
func TestRecordedHelper(t *testing.T) {
	if os.Getenv("NGN_RECORDED_HELPER") != "1" {
		return
	}
	fmt.Fprintln(os.Stderr, "helper stderr retained")
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		line := scan.Text()
		switch {
		case line == "uci":
			fmt.Println("uciok")
		case line == "isready":
			fmt.Println("readyok")
		case strings.HasPrefix(line, "go"):
			if line == "go nodes 42" {
				continue
			}
			fmt.Println("info depth 2 nodes 77 time 1 score cp 0 pv e2e4")
			fmt.Println("bestmove e2e4")
		case line == "quit":
			fmt.Println("info string final stdout retained")
			os.Exit(0)
		}
	}
	os.Exit(0)
}
func recordedFixture(t *testing.T) (*Engine, *[]string, *sync.Mutex) {
	t.Helper()
	t.Setenv("NGN_RECORDED_HELPER", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "helper.sh")
	// Test-binary paths are quoted independently of the shell's environment.
	quoted := "'" + strings.ReplaceAll(exe, "'", "'\\''") + "'"
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec "+quoted+" -test.run '^TestRecordedHelper$'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var records []string
	e := StartRecorded(path, []string{"setoption name Hash value 64"}, func(k, s string) { mu.Lock(); records = append(records, k+":"+s); mu.Unlock() })
	if e == nil {
		t.Fatal("recorded helper failed to start")
	}
	t.Cleanup(func() { Stop(e) })
	return e, &records, &mu
}
func TestRecordedTrafficAndExit(t *testing.T) {
	e, events, mu := recordedFixture(t)
	Stop(e)
	Stop(e)
	mu.Lock()
	s := strings.Join(*events, "\n")
	mu.Unlock()
	for _, want := range []string{"stdin:uci", "stdout:uciok", "stderr:helper stderr retained", "stdin:setoption name Hash value 64", "stdout:info string final stdout retained", "exit:0"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
	if strings.Index(s, "stdin:setoption") > strings.Index(s, "stdin:go movetime 200") {
		t.Fatal("option applied after warmup")
	}
}
func TestRecordedSilentSearchStopsAtDeadline(t *testing.T) {
	e, events, mu := recordedFixture(t)
	start := time.Now()
	mv := GetMove(e, "go nodes 42", 40*time.Millisecond, 10*time.Millisecond)
	if mv != "" || time.Since(start) > time.Second {
		t.Fatalf("silent engine escaped deadline: move=%q elapsed=%s", mv, time.Since(start))
	}
	Stop(e)
	mu.Lock()
	s := strings.Join(*events, "\n")
	mu.Unlock()
	if !strings.Contains(s, "stdin:stop") || !strings.Contains(s, "exit:0") {
		t.Fatal(s)
	}
}
func TestRecordedPlayGameBudgetSelection(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		cfg        GameConfig
	}{
		{"nodes", "go nodes 77", GameConfig{Nodes: 77, MaxMoves: 1}},
		{"movetime", "go movetime 51", GameConfig{Movetime: 51, MaxMoves: 1}},
		{"clock", "go wtime 500 btime 500 winc 7 binc 7", GameConfig{TCTimeMs: 500, TCIncMs: 7, MaxMoves: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, events, mu := recordedFixture(t)
			b, _, _ := recordedFixture(t)
			g := PlayGame(a, b, nil, true, tc.cfg)
			if g.Res != Draw || g.Reason != "max-moves" {
				t.Fatal(g)
			}
			Stop(a)
			Stop(b)
			mu.Lock()
			s := strings.Join(*events, "\n")
			mu.Unlock()
			if !strings.Contains(s, "stdin:"+tc.want) {
				t.Fatalf("budget command missing: %s", s)
			}
		})
	}
}
