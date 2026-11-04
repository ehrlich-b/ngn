package uci

import (
	"os"
	"testing"
)

// validSmokeResult rejects harness faults even when a game made progress.
func validSmokeResult(result GameResult, moves []string) bool {
	if len(moves) == 0 {
		return false
	}
	switch result.Reason {
	case "checkmate", "stalemate", "draw-rule", "max-moves":
		return true
	default:
		return false
	}
}

func TestValidSmokeResult(t *testing.T) {
	for _, reason := range []string{"checkmate", "stalemate", "draw-rule", "max-moves"} {
		if !validSmokeResult(GameResult{Reason: reason}, []string{"e2e4"}) {
			t.Errorf("valid ending %q rejected", reason)
		}
	}
	for _, reason := range []string{"illegal-move", "no-move", "fen-error", "bad-opening", "illegal-opening", "time-forfeit", "adj-win", "adj-draw", "unknown", ""} {
		if validSmokeResult(GameResult{Reason: reason}, []string{"e2e4"}) {
			t.Errorf("fault or unexpected ending %q accepted after a legal move", reason)
		}
	}
	if validSmokeResult(GameResult{Reason: "max-moves"}, nil) {
		t.Error("game with no moves accepted")
	}
}

// TestFixedNodeSmoke exercises the game harness on a color-swapped pair when
// explicit engine binaries are supplied. It makes no strength assessment.
func TestFixedNodeSmoke(t *testing.T) {
	a, b := os.Getenv("SMOKE_ENGINE_A"), os.Getenv("SMOKE_ENGINE_B")
	if a == "" || b == "" {
		t.Skip("engine binaries not supplied")
	}
	for _, white := range []bool{true, false} {
		candidate := Start(a, "candidate", false)
		baseline := Start(b, "baseline", false)
		var moves []string
		result := PlayGame(candidate, baseline, nil, white, GameConfig{Nodes: 100, MaxMoves: 20, MovesOut: &moves})
		Stop(candidate)
		Stop(baseline)
		if !validSmokeResult(result, moves) {
			t.Fatalf("game failed: %+v, moves %v", result, moves)
		}
		t.Logf("white=%t plies=%d reason=%s", white, len(moves), result.Reason)
	}
}
