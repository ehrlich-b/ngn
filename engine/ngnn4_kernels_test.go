package engine

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func TestNGNN4InputPackDifferential(t *testing.T) {
	values := []int32{-2147483648, -32768, -1, 0, 1, 127, 254, 255, 256, 32767, 2147483647}
	for h := 16; h <= 2048; h += 16 {
		a, b := make([]int32, h+7), make([]int32, h+9)
		us, them := a[3:3+h], b[5:5+h]
		for i := range us {
			us[i] = values[i%len(values)]
			them[i] = values[(i+4)%len(values)]
		}
		beforeA, beforeB := append([]int32(nil), a...), append([]int32(nil), b...)
		for swap := 0; swap < 2; swap++ {
			if swap == 1 {
				us, them = them, us
			}
			storage := make([]uint8, 2*h+11)
			for i := range storage {
				storage[i] = 173
			}
			dst := storage[5 : 5+2*h]
			want := make([]uint8, 2*h)
			for i, x := range append(append([]int32(nil), us...), them...) {
				if x > 255 {
					want[i] = 255
				} else if x > 0 {
					want[i] = uint8(x)
				}
			}
			ngnn4PackInput(dst, us, them)
			if !reflect.DeepEqual(dst, want) {
				t.Fatalf("AVX2 pack H=%d swap=%d", h, swap)
			}
			goOutput := make([]uint8, 2*h)
			ngnn4PackInputGo(goOutput, us, them)
			if !reflect.DeepEqual(goOutput, want) {
				t.Fatalf("Go pack H=%d swap=%d", h, swap)
			}
			for _, x := range append(append([]uint8(nil), storage[:5]...), storage[5+2*h:]...) {
				if x != 173 {
					t.Fatal("input pack changed guard")
				}
			}
		}
		if !reflect.DeepEqual(a, beforeA) || !reflect.DeepEqual(b, beforeB) {
			t.Fatal("input pack changed source")
		}
	}
}

func TestNGNN4LinearDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(41008))
	for h := 16; h <= 2048; h += 16 {
		for _, l := range []int{16, 32, 48, 64} {
			n := &NGNN1Network{hidden: h, hidden2: l, buckets: 1, layer2: make([]int8, l*2*h)}
			for i := range n.layer2 {
				n.layer2[i] = int8(rng.Intn(129) - 64)
				if h%64 == 0 {
					n.layer2[i] = 64
				}
				if h%64 == 16 {
					n.layer2[i] = -64
				}
			}
			n.packLayer2()
			// Deliberately misalign all three operands and guard their ends.
			inputStorage := make([]uint8, 2*h+7)
			weightStorage := make([]int8, len(n.packed2)+9)
			dstStorage := make([]int32, l+5)
			for i := range inputStorage {
				inputStorage[i] = 173
			}
			for i := range weightStorage {
				weightStorage[i] = 127
			}
			for i := range dstStorage {
				dstStorage[i] = 0x12345678
			}
			input := inputStorage[3 : 3+2*h]
			weights := weightStorage[5 : 5+len(n.packed2)]
			dst := dstStorage[1 : 1+l]
			copy(weights, n.packed2)
			for i := range input {
				input[i] = uint8(rng.Intn(256))
				if h%64 <= 16 {
					input[i] = 255
				}
			}
			beforeInput := append([]uint8(nil), inputStorage...)
			beforeWeights := append([]int8(nil), weightStorage...)
			want := make([]int32, l)
			for j := range want {
				var sum int64
				for i, x := range input {
					sum += int64(x) * int64(n.layer2[j*2*h+i])
				}
				want[j] = int32(sum)
			}
			ngnn4Linear(dst, input, weights)
			if !reflect.DeepEqual(dst, want) {
				t.Fatalf("backend %s H=%d L=%d", ngnn1KernelBackend, h, l)
			}
			goOutput := make([]int32, l)
			ngnn4LinearGo(goOutput, input, weights)
			if !reflect.DeepEqual(goOutput, want) {
				t.Fatalf("Go H=%d L=%d", h, l)
			}
			if !reflect.DeepEqual(beforeInput, inputStorage) || !reflect.DeepEqual(beforeWeights, weightStorage) {
				t.Fatal("kernel changed inputs")
			}
			if dstStorage[0] != 0x12345678 {
				t.Fatal("kernel changed prefix guard")
			}
			for _, x := range dstStorage[1+l:] {
				if x != 0x12345678 {
					t.Fatal("kernel changed suffix guard")
				}
			}
		}
	}
}

func TestNGNN4OutputDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(17008))
	values := []int32{-2147483648, -66846720, -16385, -129, -128, -127, -65, -64, -63, -1, 0, 1, 63, 64, 65, 16319, 16320, 16383, 66846720, 2147483647}
	biases := []int16{-32768, -255, -2, -1, 0, 1, 2, 254, 255, 256, 32767}
	weights := []int16{-32768, -32767, -129, -64, -1, 0, 1, 64, 129, 32767}
	for _, l := range []int{16, 32, 48, 64} {
		for trial := 0; trial < 200; trial++ {
			a, b, w := make([]int32, l+7), make([]int16, l+9), make([]int16, l+11)
			for i := range a {
				a[i] = 0x12345678
			}
			for i := range b {
				b[i] = 12345
			}
			for i := range w {
				w[i] = -12345
			}
			affine, bias, output := a[3:3+l], b[5:5+l], w[7:7+l]
			for j := range affine {
				affine[j] = values[(j+trial)%len(values)]
				bias[j] = biases[(j+trial/3)%len(biases)]
				output[j] = weights[(j+trial/7)%len(weights)]
				if trial%5 == 0 {
					affine[j] = int32(rng.Uint32())
					bias[j] = int16(rng.Uint32())
					output[j] = int16(rng.Uint32())
				}
				if trial == 1 || trial == 2 {
					affine[j], bias[j], output[j] = 16320, 0, 32767
					if trial == 2 {
						output[j] = -32768
					}
				}
				if trial == 3 {
					// Arithmetic shift alone maps -65/64 to -2 instead of -1.
					affine[j], bias[j], output[j] = -65, 2, 32767
				}
			}
			beforeA, beforeB, beforeW := append([]int32(nil), a...), append([]int16(nil), b...), append([]int16(nil), w...)
			var want int64
			for j, x := range affine {
				v := int64(x)/64 + int64(bias[j])
				if v < 0 {
					v = 0
				} else if v > 255 {
					v = 255
				}
				want += v * v * int64(output[j])
			}
			if got := ngnn4Output(affine, bias, output); got != want {
				t.Fatalf("backend %s L=%d trial=%d got=%d want=%d", ngnn1KernelBackend, l, trial, got, want)
			}
			if got := ngnn4OutputGo(affine, bias, output); got != want {
				t.Fatalf("Go L=%d trial=%d got=%d want=%d", l, trial, got, want)
			}
			if !reflect.DeepEqual(a, beforeA) || !reflect.DeepEqual(b, beforeB) || !reflect.DeepEqual(w, beforeW) {
				t.Fatal("output kernel changed source or guards")
			}
		}
	}
}

func BenchmarkNGNN4Output(b *testing.B) {
	for _, h := range []int{256, 512} {
		for _, l := range []int{16, 32} {
			b.Run(fmt.Sprintf("H%d/L%d", h, l), func(b *testing.B) {
				n, err := ReadNGNN1(bytes.NewReader(ngnn4TestBytes(h, l, 8, 8, false)))
				if err != nil {
					b.Fatal(err)
				}
				acc := make([]int32, 2*h)
				n.refresh(newUCIStartingPosition(), acc)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ngnn1BenchScore = n.evaluateBucket(acc, White, 7)
				}
			})
		}
	}
}
