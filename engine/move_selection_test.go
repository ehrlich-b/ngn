package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// Preserve the previous strict-greater scan as an independent ordering oracle.
func selectNextMoveReference(out []Move, scores []int, n, i int) {
	best := i
	for j := i + 1; j < n; j++ {
		if scores[j] > scores[best] {
			best = j
		}
	}
	if best != i {
		scores[i], scores[best] = scores[best], scores[i]
		out[i], out[best] = out[best], out[i]
	}
}

func TestSelectNextMoveIdentity(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	cases := [][]int{
		{0}, {0, 0, 0}, {4, 9, 9, -1}, {9, 4, 9},
		{-7, -3, -3, -9}, {1, 2, 3, 4}, {4, 3, 2, 1},
		{-maxInt - 1, maxInt, -maxInt - 1, maxInt},
	}
	rng := rand.New(rand.NewSource(20261017))
	for trial := 0; trial < 2000; trial++ {
		scores := make([]int, 1+rng.Intn(256))
		for j := range scores {
			if trial%2 == 0 {
				scores[j] = rng.Intn(9) - 4 // Exercise repeated maxima.
			} else {
				scores[j] = int(rng.Int63())
				if rng.Intn(2) == 0 {
					scores[j] = -scores[j]
				}
			}
		}
		cases = append(cases, scores)
	}
	for trial, input := range cases {
		// The unused tail must remain untouched, even when it has a higher score.
		n := len(input)
		gotScores := append(append([]int(nil), input...), maxInt, maxInt)
		wantScores := append([]int(nil), gotScores...)
		gotMoves := make([]Move, len(gotScores))
		for j := range gotMoves {
			gotMoves[j] = Move(j + 1)
		}
		wantMoves := append([]Move(nil), gotMoves...)
		for i := 0; i < n; i++ {
			selectNextMove(gotMoves, gotScores, n, i)
			selectNextMoveReference(wantMoves, wantScores, n, i)
			if !reflect.DeepEqual(gotScores, wantScores) || !reflect.DeepEqual(gotMoves, wantMoves) {
				t.Fatalf("case %d step %d: scores %v want %v; moves %v want %v",
					trial, i, gotScores, wantScores, gotMoves, wantMoves)
			}
		}
	}
}
