package main

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/ehrlich-b/ngn/internal/uci"
)

func TestPGNGamePreservesPairColorResultAndSAN(t *testing.T) {
	for _, white := range []bool{true, false} {
		var out bytes.Buffer
		game := gameOutcome{pairID: 3, game: spec{newIsWhite: white},
			moves: []string{"e2e4", "e7e5", "g1f3"}, res: uci.GameResult{Res: uci.Win, Reason: "checkmate"}}
		if err := writeGamePGN(&out, game, runConfig{tcTimeMs: 10000, tcIncMs: 100}); err != nil {
			t.Fatal(err)
		}
		want := `[Result "1-0"]`
		if !white {
			want = `[Result "0-1"]`
		}
		for _, fragment := range []string{want, `[Termination "checkmate"]`, `[TimeControl "10+0.1"]`, "1. e4 e5 2. Nf3"} {
			if !strings.Contains(out.String(), fragment) {
				t.Fatalf("missing %q in %s", fragment, out.String())
			}
		}
	}
}

func TestPGNBlackFENAndIllegalMove(t *testing.T) {
	game := gameOutcome{game: spec{opening: []string{"4k3/8/8/8/8/8/8/4K3 b - - 0 7"}},
		moves: []string{"e8d8"}, res: uci.GameResult{Res: uci.Draw, Reason: "draw-rule"}}
	var out bytes.Buffer
	if err := writeGamePGN(&out, game, runConfig{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "7... Kd8 1/2-1/2") || !strings.Contains(out.String(), `[SetUp "1"]`) {
		t.Fatal(out.String())
	}
	game.moves = []string{"e8e1"}
	if err := writeGamePGN(&out, game, runConfig{}); err == nil {
		t.Fatal("illegal replay was accepted")
	}
}

func TestCalibrationTiltMatchesBothNativeHypotheses(t *testing.T) {
	for _, elo := range []float64{0, 5} {
		shape := tiltedShape([5]float64{6, 32, 116, 44, 2}, eloToScore(elo))
		mean, total := 0.0, 0.0
		for k, probability := range shape {
			mean += float64(k) / 4 * probability
			total += probability
		}
		if math.Abs(total-1) > 1e-12 || math.Abs(mean-eloToScore(elo)) > 1e-12 {
			t.Fatalf("elo%g mean%g total%g", elo, mean, total)
		}
	}
}
