package engine

import (
	"testing"
	"time"
)

func TestProfileHotPath(t *testing.T) {
	// Run a longer search to get meaningful profiling data
	pos, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to parse FEN: %v", err)
	}

	t.Log("=== PROFILING HOT PATH PERFORMANCE ===")

	// Profile fixed depth search
	start := time.Now()
	info := SearchFixed(pos, 5, nil) // Depth 5 for meaningful work
	elapsed := time.Since(start)

	nps := float64(info.Nodes) / elapsed.Seconds()

	t.Logf("Depth 5 search profile:")
	t.Logf("  Nodes: %d", info.Nodes)
	t.Logf("  Time: %v", elapsed)
	t.Logf("  NPS: %.0f", nps)
	t.Logf("  Target: 100,000+ NPS")

	if nps < 100000 {
		t.Logf("❌ BELOW RED LINE: %.0f NPS (need 100K+)", nps)
		t.Logf("Performance gap: %.1fx slower than target", 100000.0/nps)
	} else {
		t.Logf("✅ ABOVE RED LINE: %.0f NPS", nps)
	}

	// Calculate node-per-millisecond for micro-optimization targets
	npms := float64(info.Nodes) / float64(elapsed.Nanoseconds()) * 1000000
	t.Logf("  Micro-optimization target: %.2f nodes per millisecond", npms)
}

func BenchmarkCriticalPath(b *testing.B) {
	pos, _ := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SearchFixed(pos, 4, nil)
	}
}
