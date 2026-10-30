package main

import (
	"flag"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ehrlich-b/ngn/engine"
)

func TestGeneratorCLI(t *testing.T) {
	if os.Getenv("NGN_DATA_HELPER") == "1" {
		flag.CommandLine = flag.NewFlagSet("nnuedata", flag.ExitOnError)
		os.Args = append([]string{"nnuedata"}, strings.Split(os.Getenv("NGN_DATA_ARGS"), "|")...)
		main()
		return
	}
	for _, tc := range []struct {
		name string
		args []string
		ok   bool
	}{
		{"default", nil, true},
		{"empty", []string{"-nodes", "1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "data.tsv")
			args := append([]string{"-games", "20", "-out", path}, tc.args...)
			cmd := exec.Command(os.Args[0], "-test.run=^TestGeneratorCLI$")
			cmd.Env = append(os.Environ(), "NGN_DATA_HELPER=1", "NGN_DATA_ARGS="+strings.Join(args, "|"))
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.ok {
				t.Fatalf("error=%v output=%s", err, output)
			}
			info, err := os.Stat(path)
			if err != nil || (info.Size() > 0) != tc.ok {
				t.Fatalf("output file: %v, error=%v", info, err)
			}
			if !tc.ok && !strings.Contains(string(output), "no positions generated") {
				t.Fatalf("missing diagnostic: %s", output)
			}
		})
	}
}

func TestMateOnFinalAllowedPly(t *testing.T) {
	pos, err := engine.ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	engine.ClearHash()
	engine.ClearHistoryTable()
	engine.ClearKillerMoves()
	engine.ClearCounterMoves()
	engine.SetLastMovePlayed(engine.EmptyMove)
	samples, result, failed := playGame(pos, rand.New(rand.NewSource(10599)), 3, 4, 1000)
	if failed || len(samples) != 1 || result != 0 || samples[0].white || samples[0].score != 29999 {
		t.Fatalf("samples=%+v result=%v failed=%v", samples, result, failed)
	}
}

func TestTerminalResults(t *testing.T) {
	for _, tc := range []struct {
		fen    string
		result float64
	}{
		{"7k/6Q1/6K1/8/8/8/8/8 b - - 0 1", 1},
		{"7K/6q1/6k1/8/8/8/8/8 w - - 0 1", 0},
		{"7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", 0.5},
	} {
		pos, err := engine.ParseFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		if len(engine.GenerateLegalMoves(pos)) != 0 || terminalResult(pos) != tc.result {
			t.Errorf("unexpected terminal result for %s", tc.fen)
		}
	}
}
