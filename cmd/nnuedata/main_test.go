package main

import (
	"math/rand"
	"testing"

	"github.com/ehrlich-b/ngn/engine"
)

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
