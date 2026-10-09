package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestNGNN4RowAffineDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(2026100922))
	for h := 16; h <= ngnn1MaxHidden; h += 16 {
		for _, l := range []int{16, 32, 48, 64} {
			width := 2 * h
			inputStore := make([]uint8, width+11)
			weightStore := make([]int8, l*width+13)
			dstStore := make([]int32, l+7)
			for i := range inputStore {
				inputStore[i] = 173
			}
			for i := range weightStore {
				weightStore[i] = 127
			}
			for i := range dstStore {
				dstStore[i] = 0x12345678
			}
			input := inputStore[3 : 3+width]
			weights := weightStore[5 : 5+l*width]
			dst := dstStore[1 : 1+l]
			for i := range input {
				input[i] = uint8(rng.Intn(256))
			}
			for i := range weights {
				weights[i] = int8(rng.Intn(129) - 64)
			}
			// Cover reinforcing signed extrema, lane cancellation and row order
			// at every width over the sweep. H=2048 reaches the full dot bound.
			if h%64 == 0 || h%64 == 16 {
				for i := range input {
					input[i] = 255
				}
				for j := 0; j < l; j++ {
					for i := 0; i < width; i++ {
						w := int8(64)
						if j%2 != 0 {
							w = -64
						}
						if h%64 == 16 && (i/2)%2 != 0 {
							w = -w
						}
						weights[j*width+i] = w
					}
				}
			}
			beforeInput := append([]uint8(nil), inputStore...)
			beforeWeights := append([]int8(nil), weightStore...)
			want := make([]int32, l)
			for j := range want {
				var sum int64
				for i, x := range input {
					sum += int64(x) * int64(weights[j*width+i])
				}
				want[j] = int32(sum)
			}
			ngnn4LinearRows(dst, input, weights)
			if !reflect.DeepEqual(dst, want) {
				t.Fatalf("row affine backend=%s H=%d L=%d got=%v want=%v", ngnn1KernelBackend, h, l, dst, want)
			}
			portable := make([]int32, l)
			ngnn4LinearRowsGo(portable, input, weights)
			if !reflect.DeepEqual(portable, want) {
				t.Fatalf("portable row affine H=%d L=%d", h, l)
			}
			if !reflect.DeepEqual(inputStore, beforeInput) || !reflect.DeepEqual(weightStore, beforeWeights) {
				t.Fatal("row affine changed input or weight guards")
			}
			for _, x := range append(append([]int32(nil), dstStore[:1]...), dstStore[1+l:]...) {
				if x != 0x12345678 {
					t.Fatal("row affine changed destination guard")
				}
			}
		}
	}
}
