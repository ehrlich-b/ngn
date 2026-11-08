package main

import (
	"math"
	"math/rand"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ehrlich-b/ngn/internal/uci"
)

func TestFixedOptionsPreserveOrderAndPaths(t *testing.T) {
	var options optionFlags
	for _, value := range []string{"EvalFile=/net directory/accepted.nnue", "UseNNUE=true", "Hash=64", "Threads=1"} {
		if err := options.Set(value); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"setoption name EvalFile value /net directory/accepted.nnue", "setoption name UseNNUE value true", "setoption name Hash value 64", "setoption name Threads value 1"}
	if !reflect.DeepEqual([]string(options), want) {
		t.Fatalf("options=%v", options)
	}
	for _, invalid := range []string{"EvalFile", "=value", "EvalFile=x\nquit"} {
		if options.Set(invalid) == nil {
			t.Fatalf("accepted malformed option %q", invalid)
		}
	}
}

func TestTuningGameFaultCannotUpdateTheta(t *testing.T) {
	for _, reason := range []string{"time-forfeit", "no-move", "illegal-move", "search-error", "illegal-opening", "watchdog"} {
		if _, err := validGamePoints(uci.GameResult{Res: uci.Win, Reason: reason}); err == nil {
			t.Fatalf("scored fault %s", reason)
		}
	}
	for _, reason := range []string{"checkmate", "stalemate", "draw-rule", "max-moves"} {
		if _, err := validGamePoints(uci.GameResult{Res: uci.Draw, Reason: reason}); err != nil {
			t.Fatal(err)
		}
	}
	params, before := testParams(100), testParams(100)
	reported := false
	err := runCampaign(params, 1, 100, 10, rand.New(rand.NewSource(1)), 25, 1, 1,
		func([][]int, [][]int, []int) []float64 { return []float64{math.NaN()} },
		func(int) error { reported = true; return nil })
	if err == nil || reported || !reflect.DeepEqual(params, before) {
		t.Fatalf("faulted batch mutated theta or checkpoint: %v reported=%v", err, reported)
	}
}

func TestCheckpointBindsFixedOptions(t *testing.T) {
	params := testParams(100)
	checkpoint, recipe := checkpointFixture(t)
	recipe.EngineOptions = []string{"setoption name EvalFile value final115.nnue", "setoption name Threads value 1"}
	path := filepath.Join(t.TempDir(), "spsa.jsonl")
	if err := writeCheckpoint(path, checkpoint.Iter, params, 50, recipe, checkpoint.RNGDraws); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadCheckpoint(path, testParams(100), recipe); err != nil {
		t.Fatal(err)
	}
	for _, options := range [][]string{nil, {"setoption name EvalFile value old.nnue", "setoption name Threads value 1"}, {"setoption name Threads value 1", "setoption name EvalFile value final115.nnue"}} {
		changed := recipe
		changed.EngineOptions = options
		if _, _, _, err := loadCheckpoint(path, testParams(100), changed); err == nil {
			t.Fatalf("resumed with changed options %v", options)
		}
	}
}
