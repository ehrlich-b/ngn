package uci

import (
	"os"
	"testing"
)

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
		if len(moves) == 0 || result.Reason == "no-move" {
			t.Fatalf("game failed: %+v, moves %v", result, moves)
		}
		t.Logf("white=%t plies=%d reason=%s", white, len(moves), result.Reason)
	}
}
