package engine

import (
	"testing"
	"time"
)

func TestBasicSearchWorks(t *testing.T) {
	pos := newUCIStartingPosition()
	
	// Test that search returns something within reasonable time
	start := time.Now()
	info := Search(pos, 1)
	elapsed := time.Since(start)
	
	if elapsed > 5*time.Second {
		t.Errorf("Search took too long: %v", elapsed)
	}
	
	if info == nil {
		t.Fatal("Search returned nil")
	}
	
	if info.BestMove == EmptyMove {
		t.Error("Search didn't find any move")
	}
	
	t.Logf("Search depth 1 took %v, found move: %v, score: %d", elapsed, info.BestMove.ToString(), info.BestScore)
}

func TestSearchDepth2(t *testing.T) {
	pos := newUCIStartingPosition()
	
	start := time.Now()
	info := Search(pos, 2)
	elapsed := time.Since(start)
	
	if elapsed > 10*time.Second {
		t.Errorf("Depth 2 search took too long: %v", elapsed)
	}
	
	if info == nil {
		t.Fatal("Search returned nil")
	}
	
	if info.BestMove == EmptyMove {
		t.Error("Search didn't find any move")
	}
	
	t.Logf("Search depth 2 took %v, found move: %v, score: %d", elapsed, info.BestMove.ToString(), info.BestScore)
}