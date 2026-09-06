package countereval

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestEvaluateFullRefreshUsesCounterFeatureAndArithmeticOrder(t *testing.T) {
	model := new(Model)
	model.hiddenBiases[0] = 3
	model.hiddenBiases[1] = -2
	model.hiddenWeights[(0*64+0)*HiddenSize+0] = 2
	model.hiddenWeights[(0*64+0)*HiddenSize+1] = 3
	model.hiddenWeights[(10*64+63)*HiddenSize+0] = 4
	model.hiddenWeights[(10*64+63)*HiddenSize+1] = -1
	model.outputWeights[0] = 2
	model.outputWeights[1] = 5
	model.outputBias = 7

	var board Board
	board[0] = uint64(1) << 0
	board[10] = uint64(1) << 63
	features, accumulator, output, err := model.evaluateFullRefreshTrace(board)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{0, 703}; !reflect.DeepEqual(features, want) {
		t.Fatalf("features=%v want=%v", features, want)
	}
	if accumulator[0] != 9 || accumulator[1] != 0 {
		t.Fatalf("accumulator[0:2]=%v", accumulator[:2])
	}
	if output != 25 {
		t.Fatalf("output=%v want=25", output)
	}
	publicOutput, err := model.EvaluateFullRefresh(board)
	if err != nil || math.Float32bits(publicOutput) != math.Float32bits(output) {
		t.Fatalf("public output=%v err=%v", publicOutput, err)
	}
}

func TestEvaluateFullRefreshRejectsOverlapAndNilModel(t *testing.T) {
	var board Board
	board[0] = 1 << 12
	board[11] = 1 << 12
	if _, err := new(Model).EvaluateFullRefresh(board); err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("overlap err=%v", err)
	}
	var model *Model
	if _, err := model.EvaluateFullRefresh(Board{}); err == nil || !strings.Contains(err.Error(), "nil model") {
		t.Fatalf("nil model err=%v", err)
	}
}

func TestEvaluateFullRefreshScansSquaresBeforePlanes(t *testing.T) {
	model := new(Model)
	var board Board
	board[11] = 1 << 0
	board[0] = 1 << 63
	features, _, _, err := model.evaluateFullRefreshTrace(board)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{704, 63}; !reflect.DeepEqual(features, want) {
		t.Fatalf("features=%v want=%v", features, want)
	}
}
