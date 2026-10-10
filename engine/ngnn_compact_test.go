package engine

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"math/rand"
	"reflect"
	"testing"
)

func TestNGNNCompactLoaderBound(t *testing.T) {
	for _, tc := range []struct {
		name         string
		bias, weight int16
		want         bool
	}{
		{"upper_exact", 32703, 1, true}, {"upper_over", 32704, 1, false},
		{"lower_exact", -32704, -1, true}, {"lower_over", -32705, -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := ngnn3TestBytes(16, 8, 8, false)
			// All rows in all buckets: one unit per occupied square in lane0.
			for row := 0; row < 8*768; row++ {
				binary.LittleEndian.PutUint16(data[96+2*row*16:], uint16(tc.weight))
			}
			binary.LittleEndian.PutUint16(data[96+2*8*768*16:], uint16(tc.bias))
			binary.LittleEndian.PutUint32(data[len(data)-4:], crc32.ChecksumIEEE(data[:len(data)-4]))
			n, err := ReadNGNN1(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if n.compactInput != tc.want {
				t.Fatalf("loader compact=%v, want %v", n.compactInput, tc.want)
			}
		})
	}
	if ngnn3TestNetwork(t, 16, true).compactInput {
		t.Fatal("extreme network chose compact accumulators")
	}
}

func TestNGNNCompactKernels(t *testing.T) {
	rng := rand.New(rand.NewSource(2026101006))
	for _, h := range []int{16, 32, 512, 2048} {
		rows := make([][]int16, 6)
		for k := range rows {
			back := make([]int16, h+2)
			rows[k] = back[1 : h+1] // intentionally unaligned
			for i := range rows[k] {
				rows[k][i] = int16(rng.Uint32())
			}
		}
		src := make([]int16, 2*h+2)[1 : 2*h+1]
		for i := range src {
			src[i] = int16(rng.Uint32())
		}
		for _, alias := range []bool{false, true} {
			for _, kind := range []string{"add", "sub", "move", "pair", "capture", "castle"} {
				back := make([]int16, 2*h+2)
				back[0], back[len(back)-1] = 12345, -12345
				dst := back[1 : len(back)-1]
				copy(dst, src)
				input := append([]int16(nil), src...)
				if alias {
					input = dst
				}
				want := append([]int16(nil), src...)
				for i := 0; i < h; i++ {
					x := int64(src[i])
					switch kind {
					case "add":
						x += int64(rows[0][i])
					case "sub":
						x -= int64(rows[0][i])
					case "move", "pair":
						x += -int64(rows[0][i]) + int64(rows[1][i])
					case "capture":
						x += -int64(rows[0][i]) - int64(rows[1][i]) + int64(rows[2][i])
					case "castle":
						x += -int64(rows[0][i]) - int64(rows[1][i]) + int64(rows[2][i]) + int64(rows[3][i])
					}
					want[i] = int16(x)
					if kind == "pair" {
						want[h+i] = int16(int64(src[h+i]) - int64(rows[2][i]) + int64(rows[3][i]))
					}
					if kind == "capture" {
						want[h+i] = int16(int64(src[h+i]) - int64(rows[3][i]) - int64(rows[4][i]) + int64(rows[5][i]))
					}
				}
				switch kind {
				case "add":
					ngnn16AddRow(dst[:h], rows[0])
				case "sub":
					ngnn16SubRow(dst[:h], rows[0])
				case "move":
					ngnn16MoveRows(dst[:h], input[:h], rows[0], rows[1])
				case "pair":
					ngnn16MovePair(dst, input, rows[0], rows[1], rows[2], rows[3])
				case "capture":
					ngnn16CapturePair(dst, input, rows[0], rows[1], rows[2], rows[3], rows[4], rows[5])
				case "castle":
					ngnn16CastleRows(dst[:h], input[:h], rows[0], rows[1], rows[2], rows[3])
				}
				if !reflect.DeepEqual(dst, want) || back[0] != 12345 || back[len(back)-1] != -12345 {
					t.Fatalf("H%d alias%v %s mismatch or guard overwrite", h, alias, kind)
				}
			}
		}
		weights := make([]int16, 2*h)
		var want int64
		for i := range weights {
			weights[i] = int16(rng.Intn(257) - 128)
			x := int64(min(255, max(0, src[i])))
			want += x * x * int64(weights[i])
		}
		if got := ngnn16OutputSmall(src[:h], src[h:], weights); got != want {
			t.Fatalf("H%d output=%d want%d", h, got, want)
		}
	}
}

func TestNGNNCompactNullGrowth(t *testing.T) {
	n := ngnn3TestNetwork(t, 512, false)
	if !n.compactInput {
		t.Fatal("bounded fixture did not choose compact path")
	}
	pos := newUCIStartingPosition()
	e := ngnn1NewWorker(t, n, pos)
	root := append([]int16(nil), e.nnue.frames[0].compact...)
	var eps []Square
	for ply := 1; ply <= 261; ply++ {
		tr := e.mustPrepareNull(pos)
		ep := pos.MakeNullMove()
		e.mustPushMadeNull(pos, tr, ep)
		eps = append(eps, ep)
		if &e.nnue.frames[ply].compact[0] != &e.nnue.frames[0].compact[0] {
			t.Fatal("null growth lost root alias")
		}
	}
	for _, move := range GenerateLegalMoves(pos) {
		tr := e.mustPrepareMove(pos, move)
		ep, tag, clock, _ := pos.MakeMove(move)
		e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
		ngnn3Assert(t, e, pos)
		if !reflect.DeepEqual(root, e.nnue.frames[0].compact) || &e.nnue.frames[e.depth].compact[0] == &e.nnue.frames[0].compact[0] {
			t.Fatal("real child changed or aliased root")
		}
		e.mustPop()
		pos.UnMakeMove(move, tag, ep, clock)
	}
	for i := len(eps) - 1; i >= 0; i-- {
		e.mustPop()
		pos.UnMakeNullMove(eps[i])
		ngnn3Assert(t, e, pos)
	}
	if len(e.nnue.stack) != 0 || len(e.nnue.compactStack) == 0 {
		t.Fatal("compact path retained wide accumulator stack")
	}
}

func TestNGNNCompactOutputExtremes(t *testing.T) {
	for _, h := range []int{16, 512, 2048} {
		for _, weight := range []int16{-128, 128} {
			for _, activation := range []int16{-32768, 0, 255, 32767} {
				a := make([]int16, 2*h+2)[1 : 2*h+1]
				w := make([]int16, 2*h+2)[1 : 2*h+1]
				for i := range a {
					a[i], w[i] = activation, weight
				}
				x := int64(min(255, max(0, activation)))
				want := int64(2*h) * x * x * int64(weight)
				if got := ngnn16OutputSmall(a[:h], a[h:], w); got != want {
					t.Fatalf("H%d activation%d weight%d got%d want%d", h, activation, weight, got, want)
				}
			}
		}
	}
}
