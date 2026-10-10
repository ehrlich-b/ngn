package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestNGNN1CapturePairExact(t *testing.T) {
	rng := rand.New(rand.NewSource(6101042))
	for h := 16; h <= ngnn1MaxHidden; h += 16 {
		for sample := 0; sample < 8; sample++ {
			parentStorage, parent := ngnn1Guarded32(2*h, 1+sample%7)
			for i := range parent {
				parent[i] = int32(rng.Uint32())
			}
			beforeParent := append([]int32(nil), parentStorage...)
			var rows [6][]int16
			for r := range rows {
				storage := make([]int16, h+16)
				rows[r] = storage[1+r : 1+r+h]
				for i := range rows[r] {
					rows[r][i] = int16(rng.Uint32())
					if sample < 2 {
						rows[r][i] = []int16{-32768, 32767}[(i+r+sample)%2]
					}
				}
			}
			for _, alias := range []bool{false, true} {
				storage, dst := ngnn1Guarded32(2*h, 1+(sample+3)%7)
				wantStorage := append([]int32(nil), storage...)
				want := wantStorage[1+(sample+3)%7 : 1+(sample+3)%7+2*h]
				src, reference := parent, parent
				if alias {
					copy(dst, parent)
					copy(want, parent)
					src, reference = dst, want
				}
				ngnn1CapturePair(dst, src, rows[0], rows[1], rows[2], rows[3], rows[4], rows[5])
				ngnn1CaptureRowsGo(want[:h], reference[:h], rows[0], rows[1], rows[2])
				ngnn1CaptureRowsGo(want[h:], reference[h:], rows[3], rows[4], rows[5])
				if !reflect.DeepEqual(storage, wantStorage) {
					t.Fatalf("width=%d sample=%d alias=%v backend=%s", h, sample, alias, ngnn1KernelBackend)
				}
			}
			if !reflect.DeepEqual(parentStorage, beforeParent) {
				t.Fatal("source frame changed")
			}
		}
	}
}
