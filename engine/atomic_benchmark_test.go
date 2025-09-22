// Benchmark testing to measure atomic vs non-atomic TT performance
package engine

import (
	"testing"
	"time"
)

// BenchmarkAtomicVsNonAtomic compares search performance with and without atomic operations
func TestAtomicVsNonAtomicPerformance(t *testing.T) {
	// Test position that exercises the TT heavily
	position, err := ParseFEN("r1bqk2r/ppp2ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 4 6")
	if err != nil {
		t.Fatalf("Failed to parse test position: %v", err)
	}

	const testDepth = 6
	const iterations = 3

	// Test with atomic operations (normal mode)
	BenchmarkMode = false
	var atomicNodes uint64
	atomicStart := time.Now()

	for i := 0; i < iterations; i++ {
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE) // Fresh TT
		info := Search(position, testDepth)
		atomicNodes += info.Nodes
	}

	atomicDuration := time.Since(atomicStart)
	atomicNPS := float64(atomicNodes) / atomicDuration.Seconds()

	// Test with non-atomic operations (benchmark mode)
	BenchmarkMode = true
	var nonAtomicNodes uint64
	nonAtomicStart := time.Now()

	for i := 0; i < iterations; i++ {
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE) // Fresh TT
		info := Search(position, testDepth)
		nonAtomicNodes += info.Nodes
	}

	nonAtomicDuration := time.Since(nonAtomicStart)
	nonAtomicNPS := float64(nonAtomicNodes) / nonAtomicDuration.Seconds()

	// Reset to normal mode
	BenchmarkMode = false

	// Calculate improvement
	improvement := (nonAtomicNPS - atomicNPS) / atomicNPS * 100

	t.Logf("=== ATOMIC VS NON-ATOMIC TT PERFORMANCE ===")
	t.Logf("Test: %d iterations at depth %d", iterations, testDepth)
	t.Logf("Position: Complex middlegame")
	t.Logf("")
	t.Logf("Atomic mode (normal):")
	t.Logf("  - Total nodes: %d", atomicNodes)
	t.Logf("  - Duration: %.3fs", atomicDuration.Seconds())
	t.Logf("  - NPS: %.0f", atomicNPS)
	t.Logf("")
	t.Logf("Non-atomic mode (benchmark):")
	t.Logf("  - Total nodes: %d", nonAtomicNodes)
	t.Logf("  - Duration: %.3fs", nonAtomicDuration.Seconds())
	t.Logf("  - NPS: %.0f", nonAtomicNPS)
	t.Logf("")
	t.Logf("Performance improvement: %.1f%%", improvement)

	// Sanity checks - allow small tolerance for non-atomic mode differences
	nodeDiff := float64(atomicNodes) - float64(nonAtomicNodes)
	if nodeDiff < 0 {
		nodeDiff = -nodeDiff
	}
	tolerance := float64(atomicNodes) * 0.05 // 5% tolerance
	if nodeDiff > tolerance {
		t.Errorf("Node counts differ by more than 5%%: atomic=%d, non-atomic=%d (diff=%.1f%%)",
			atomicNodes, nonAtomicNodes, (nodeDiff/float64(atomicNodes))*100)
	}

	if improvement < 0 {
		t.Logf("WARNING: Non-atomic mode slower by %.1f%% - this is unexpected", -improvement)
	} else if improvement < 5 {
		t.Logf("INFO: Small improvement of %.1f%% - atomic overhead is minimal", improvement)
	} else if improvement < 20 {
		t.Logf("SUCCESS: Good improvement of %.1f%% - atomic overhead reduced significantly", improvement)
	} else {
		t.Logf("EXCELLENT: Large improvement of %.1f%% - major atomic overhead eliminated", improvement)
	}
}

// BenchmarkSearchWithAtomicOperations benchmarks search with standard atomic TT
func BenchmarkSearchWithAtomicOperations(b *testing.B) {
	position, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		b.Fatalf("Failed to parse position: %v", err)
	}

	BenchmarkMode = false
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
		Search(position, 5)
	}
}

// BenchmarkSearchWithoutAtomicOperations benchmarks search with non-atomic TT
func BenchmarkSearchWithoutAtomicOperations(b *testing.B) {
	position, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		b.Fatalf("Failed to parse position: %v", err)
	}

	BenchmarkMode = true
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
		Search(position, 5)
	}

	// Reset to normal mode after benchmark
	BenchmarkMode = false
}

// Helper function to enable non-atomic mode for benchmarks
func EnableBenchmarkMode() {
	BenchmarkMode = true
}

// Helper function to disable non-atomic mode (return to thread-safe operation)
func DisableBenchmarkMode() {
	BenchmarkMode = false
}
