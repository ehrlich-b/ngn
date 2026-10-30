package main

import (
	"github.com/ehrlich-b/ngn/engine"
	"math/rand"
	"testing"
	"time"
)

func TestTrainingGameIDsExcludeHoldout(t *testing.T) {
	last := 0
	for i := 0; i < 10000; i++ {
		id := trainingGameID(i)
		if id <= last || id%50 == 0 {
			t.Fatalf("game %d: id %d after %d", i, id, last)
		}
		last = id
	}
}

func TestExpiredGameIsDiscarded(t *testing.T) {
	play, label := engine.NewSearchEngine(), engine.NewSearchEngine()
	rows, _, ok := game(play, label, rand.New(rand.NewSource(1)), 8, 200, time.Now().Add(-time.Second))
	if ok || len(rows) != 0 {
		t.Fatalf("expired game returned %d rows, ok=%v", len(rows), ok)
	}
}
