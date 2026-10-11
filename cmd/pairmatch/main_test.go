package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ehrlich-b/ngn/internal/uci"
)

func TestPairQueueConcurrentBoundAndFailure(t *testing.T) {
	q := pairQueue{limit: 17}
	ids := make(chan int, 17)
	var wg sync.WaitGroup
	for worker := 0; worker < 6; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				id, ok := q.admit(time.Now())
				if !ok {
					return
				}
				ids <- id
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatal("duplicate admission")
		}
		seen[id] = true
	}
	if len(seen) != 17 || !seen[0] || !seen[16] {
		t.Fatal(seen)
	}
	q = pairQueue{limit: 2}
	q.fail()
	if _, ok := q.admit(time.Now()); ok {
		t.Fatal("admitted after failure")
	}
	q = pairQueue{limit: 2, cutoff: time.Now().Add(-time.Second)}
	if _, ok := q.admit(time.Now()); ok {
		t.Fatal("admitted after deadline")
	}
}

func TestPairLimitValidation(t *testing.T) {
	for _, tc := range []struct {
		games, book int
		deadline    time.Time
		want        int
		bad         bool
	}{
		{1000, 500, time.Time{}, 500, false}, {0, 500, time.Now(), 500, false},
		{999, 500, time.Time{}, 0, true}, {-2, 500, time.Time{}, 0, true},
		{1000, 499, time.Time{}, 0, true}, {0, 500, time.Time{}, 0, true},
	} {
		got, err := pairLimit(tc.games, tc.book, tc.deadline)
		if (err != nil) != tc.bad || got != tc.want {
			t.Fatalf("%+v: %d %v", tc, got, err)
		}
	}
	for _, reason := range []string{"illegal-move", "no-move", "fen-error", "bad-opening", "engine-error", "time-forfeit", ""} {
		if validResult(uci.GameResult{Res: uci.Draw, Reason: reason}) {
			t.Fatalf("accepted %q", reason)
		}
	}
}

func TestFixedCountCommand(t *testing.T) {
	if os.Getenv("NGN_PAIRMATCH_HELPER") == "1" {
		flag.CommandLine = flag.NewFlagSet("pairmatch", flag.ExitOnError)
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{"pairmatch"}, os.Args[i+1:]...)
				break
			}
		}
		main()
		os.Exit(0)
	}
	if runtime.GOOS == "windows" {
		t.Skip("POSIX engine fixture")
	}
	dir := t.TempDir()
	engine := filepath.Join(dir, "engine")
	script := `#!/bin/sh
while IFS= read -r line; do
 case "$line" in
 uci) printf 'uciok\n' ;;
 isready) printf 'readyok\n' ;;
 'go movetime 200') printf 'bestmove e2e4\n' ;;
 'go wtime '*) printf 'bestmove g6g7\n' ;;
 quit) exit 0 ;;
 esac
done
`
	if err := os.WriteFile(engine, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	book := filepath.Join(dir, "openings")
	if err := os.WriteFile(book, []byte(strings.Repeat("7k/5K2/6Q1/8/8/8/8/8 w - - 0 1\n", 8)), 0600); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		extra []string
		ok    bool
		fault bool
	}{
		{"complete", nil, true, false}, {"deadline", []string{"-until", time.Now().Add(time.Minute).Format(time.RFC3339)}, false, false},
		{"fault", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(dir, tc.name+".jsonl")
			if tc.fault {
				if err := os.WriteFile(book, []byte(strings.Repeat("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1\n", 8)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"-test.run=^TestFixedCountCommand$", "--", "-ngn", engine, "-opponent", engine, "-openings", book, "-games", "10", "-concurrency", "6", "-out", out}
			args = append(args, tc.extra...)
			cmd := exec.Command(exe, args...)
			cmd.Env = append(os.Environ(), "NGN_PAIRMATCH_HELPER=1")
			data, err := cmd.Output()
			if (err == nil) != tc.ok {
				t.Fatalf("exit %v; %s", err, data)
			}
			var summary struct {
				Games, Pairs, RequestedGames int
				Complete                     bool
				OperationalFaults            int
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("%s: %v", data, err)
			}
			json.Unmarshal(raw["games"], &summary.Games)
			json.Unmarshal(raw["pairs"], &summary.Pairs)
			json.Unmarshal(raw["complete"], &summary.Complete)
			json.Unmarshal(raw["operational_faults"], &summary.OperationalFaults)
			if summary.Complete != tc.ok || (summary.OperationalFaults > 0) != tc.fault {
				t.Fatalf("summary %s", data)
			}
			if tc.ok && (summary.Games != 10 || summary.Pairs != 5) {
				t.Fatalf("count %s", data)
			}
			lines, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if tc.ok && strings.Count(string(lines), "\n") != 5 {
				t.Fatalf("raw pairs %s", lines)
			}
			if tc.ok {
				for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
					var row pair
					if err := json.Unmarshal([]byte(line), &row); err != nil {
						t.Fatal(err)
					}
					for _, g := range row.Games {
						if len(g.Clocks) != 1 || len(g.Moves) != 1 {
							t.Fatalf("missing per-ply clock wire evidence: %+v", g)
						}
						clock := g.Clocks[0]
						if clock.Go != "go wtime 120000 btime 120000 winc 1000 binc 1000" ||
							clock.Ply != 1 || !clock.White || clock.Move != "g6g7" ||
							clock.Flagged || clock.IncrementAppliedMs != 1000 ||
							clock.WhiteAfterMs != 121000-clock.ElapsedMs || clock.BlackAfterMs != 120000 {
							t.Fatalf("clock wire mismatch: %+v", clock)
						}
					}
				}
			}
		})
	}
}
