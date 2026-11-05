package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	wave6H          = 32
	wave6Linear     = 120 + 2*(6144+1)*wave6H
	wave6LinearBias = wave6Linear + 2*8*2*wave6H
	wave6Hidden     = wave6LinearBias + 4*8
	wave6HiddenBias = wave6Hidden + 2*16*2*wave6H
	wave6Output     = wave6HiddenBias + 4*16
	wave6OutputBias = wave6Output + 2*8*16
)

func wave6Fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "experiments", "wave2-1104", "residual-format", "synthetic-h32.ngn6"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func wave6CRC(raw []byte) {
	binary.LittleEndian.PutUint32(raw[len(raw)-4:], crc32.ChecksumIEEE(raw[:len(raw)-4]))
}

func wave6Read(t *testing.T, raw []byte) *NGNN6Network {
	t.Helper()
	wave6CRC(raw)
	n, err := ReadNGNN6(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWave2ResidualFormatOracle(t *testing.T) {
	raw := wave6Fixture(t)
	n, err := ReadNGNN6(bytes.NewReader(raw))
	if err != nil || n == nil || !n.linear.valid() || n.linear.hidden != wave6H {
		t.Fatalf("H32 layout: %v", err)
	}
	var oracle struct {
		Format string `json:"format"`
		Hidden int    `json:"hidden"`
		SHA256 string `json:"network_sha256"`
		Cases  []struct {
			FEN        string  `json:"fen"`
			Bucket     int     `json:"bucket"`
			Accum      []int32 `json:"accum"`
			LinearCP   int     `json:"linear_cp"`
			ResidualCP int     `json:"residual_cp"`
			CP         int     `json:"cp"`
		} `json:"cases"`
	}
	data, err := os.ReadFile(filepath.Join("..", "experiments", "wave2-1104", "residual-format", "oracle-h32.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if oracle.Format != "NGNN6" || oracle.Hidden != wave6H || oracle.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) || len(oracle.Cases) != 16 {
		t.Fatal("fixture metadata mismatch")
	}
	var seen [8][2]bool
	for _, c := range oracle.Cases {
		pos, err := ParseFEN(c.FEN)
		if err != nil {
			t.Fatal(err)
		}
		acc := make([]int32, 2*wave6H)
		n.linear.refresh(pos, acc)
		if !reflect.DeepEqual(acc, c.Accum) || n.linear.bucket(pos) != c.Bucket {
			t.Fatalf("accumulator/bucket mismatch: %s", c.FEN)
		}
		if got := n.evaluateBucket(acc, pos.Turn(), c.Bucket); got != c.CP {
			t.Fatalf("%s: score %d, oracle %d", c.FEN, got, c.CP)
		}
		linear := n.linear.evaluateBucket(acc, pos.Turn(), c.Bucket)
		if linear != c.LinearCP || c.CP != c.LinearCP+c.ResidualCP {
			t.Fatalf("fixture branch mismatch: %s", c.FEN)
		}
		turn := 0
		if pos.Turn() == Black {
			turn = 1
		}
		seen[c.Bucket][turn] = true
	}
	for b, turns := range seen {
		if !turns[0] || !turns[1] {
			t.Fatalf("missing bucket %d turn", b)
		}
	}
	if model, err := ReadNGNN1(bytes.NewReader(raw)); model != nil || err == nil {
		t.Fatal("default loader accepted NGNN6")
	}
	// The prefix really is the original NGNN3 payload, not a reinterpretation.
	legacy := append([]byte(nil), raw[:96]...)
	copy(legacy, "NGN3")
	binary.LittleEndian.PutUint32(legacy[4:], 3)
	legacy = append(legacy, raw[120:wave6Hidden]...)
	legacy = append(legacy, 0, 0, 0, 0)
	wave6CRC(legacy)
	base, err := ReadNGNN1(bytes.NewReader(legacy))
	if err != nil || !reflect.DeepEqual(base.weights, n.linear.weights) || !reflect.DeepEqual(base.output, n.linear.output) || !reflect.DeepEqual(base.bias, n.linear.bias) || !reflect.DeepEqual(base.otherBiases, n.linear.otherBiases) || base.outputBias != n.linear.outputBias || base.kingMap != n.linear.kingMap {
		t.Fatalf("NGNN3 prefix mismatch: %v", err)
	}
	t.Logf("16 independent H32 oracle cases; both turns and all eight rows; fixture bytes=%d", len(raw))
}

func TestWave2ResidualFormatZeroProjection(t *testing.T) {
	raw := wave6Fixture(t)
	// Full int16 bottleneck weights, with arbitrary nonzero biases, remain
	// admissible at H32. They must be irrelevant when R and RB are both zero.
	for offset := wave6Hidden; offset < wave6HiddenBias; offset += 2 {
		w := uint16(0x8000)
		if offset%4 == 0 {
			w = 0x7fff
		}
		binary.LittleEndian.PutUint16(raw[offset:], w)
	}
	for j := 0; j < 16; j++ {
		binary.LittleEndian.PutUint32(raw[wave6HiddenBias+j*4:], uint32(j*17777))
	}
	for i := wave6Output; i < len(raw)-4; i++ {
		raw[i] = 0
	}
	for _, extreme := range []bool{false, true} {
		if extreme {
			for i := wave6Linear; i < wave6LinearBias; i += 2 {
				w := uint16(0x8000)
				if i%4 == 0 {
					w = 0x7fff
				}
				binary.LittleEndian.PutUint16(raw[i:], w)
			}
		}
		n := wave6Read(t, raw)
		for sample := 0; sample < 40; sample++ {
			acc := make([]int32, 2*wave6H)
			for i := range acc {
				acc[i] = int32((sample*173+i*67)%1024 - 512)
				if sample == 0 {
					acc[i] = -1 << 31
				} else if sample == 1 {
					acc[i] = 1<<31 - 1
				}
			}
			for _, turn := range []Color{White, Black} {
				for b := 0; b < 8; b++ {
					if got, want := n.evaluateBucket(acc, turn, b), n.linear.evaluateBucket(acc, turn, b); got != want {
						t.Fatalf("zero projection extreme=%v sample=%d turn=%d bucket=%d: %d != %d", extreme, sample, turn, b, got, want)
					}
				}
			}
		}
	}
}

func TestWave2ResidualFormatArithmetic(t *testing.T) {
	empty := wave6Fixture(t)
	for i := 120; i < len(empty)-4; i++ {
		empty[i] = 0
	}
	for _, tc := range []struct {
		name                                     string
		lb, rb, hb                               int32
		linearWeight, hiddenWeight, outputWeight int16
		x                                        int32
		want                                     int
	}{
		{name: "linear inner positive truncation", x: 1, linearWeight: 32767, lb: -128, want: 0},
		{name: "linear inner negative truncation", x: 1, linearWeight: -32768, lb: 128, want: 0},
		{name: "linear outer negative truncation", lb: -40, want: 0},
		{name: "linear outer negative cp", lb: -41, want: -1},
		{name: "residual negative truncation", rb: -40, want: 0},
		{name: "residual negative cp", rb: -41, want: -1},
		{name: "independent branch truncations", lb: 40, rb: 40, want: 0},
		{name: "bottleneck truncate below two", hb: 127, outputWeight: 64, want: 1},
		{name: "bottleneck exact two", hb: 128, outputWeight: 64, want: 3},
		{name: "unsquared bottleneck input", x: 2, hiddenWeight: 32, outputWeight: 64, want: 1},
		{name: "bottleneck lower clip", hb: -64000, outputWeight: 64, want: 0},
		{name: "bottleneck upper clip", hb: 64000, outputWeight: 64, want: 400},
		{name: "positive before final clamp", lb: 1224000, rb: -408000, want: 20000},
		{name: "negative before final clamp", lb: -1224000, rb: 408000, want: -20000},
		{name: "positive final clamp", lb: 1224000, want: 24999},
		{name: "negative final clamp", lb: -1224000, want: -24999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := bytes.Clone(empty)
			binary.LittleEndian.PutUint16(raw[wave6Linear:], uint16(tc.linearWeight))
			binary.LittleEndian.PutUint32(raw[wave6LinearBias:], uint32(tc.lb))
			binary.LittleEndian.PutUint16(raw[wave6Hidden:], uint16(tc.hiddenWeight))
			binary.LittleEndian.PutUint32(raw[wave6HiddenBias:], uint32(tc.hb))
			binary.LittleEndian.PutUint16(raw[wave6Output:], uint16(tc.outputWeight))
			binary.LittleEndian.PutUint32(raw[wave6OutputBias:], uint32(tc.rb))
			n := wave6Read(t, raw)
			acc := make([]int32, 2*wave6H)
			acc[0] = tc.x
			if got := n.evaluateBucket(acc, White, 0); got != tc.want {
				t.Fatalf("score %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWave2ResidualFormatMalformed(t *testing.T) {
	good := wave6Fixture(t)
	reject := func(name string, raw []byte, message string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			n, err := ReadNGNN6(bytes.NewReader(raw))
			if n != nil || err == nil || !strings.Contains(err.Error(), message) {
				t.Fatalf("expected %q rejection, got layout=%v err=%v", message, n != nil, err)
			}
		})
	}
	for _, field := range []struct {
		offset  int
		values  []uint32
		message string
	}{
		{0, []uint32{0}, "magic/version"},
		{4, []uint32{0, 3, 4, 5, 7, 0xffffffff}, "magic/version"},
		{8, []uint32{0, 16, 31, 33, 48, 256, 2048, 0xffffffff}, "hidden width"},
		{12, []uint32{0, 1, 7, 9, 0xffffffff}, "dimensions"},
		{16, []uint32{0, 1, 4, 9, 0xffffffff}, "dimensions"},
		{96, []uint32{0, 8, 15, 17, 0xffffffff}, "dimensions"},
		{116, []uint32{1, 0xffffffff}, "reserved"},
	} {
		for _, value := range field.values {
			raw := bytes.Clone(good)
			binary.LittleEndian.PutUint32(raw[field.offset:], value)
			wave6CRC(raw)
			reject(fmt.Sprintf("header_%d_%d", field.offset, value), raw, field.message)
		}
	}
	for _, offset := range []int{20, 24, 28, 100, 104, 108, 112} {
		for _, value := range []uint32{0, 1, binary.LittleEndian.Uint32(good[offset:]) + 1, 0xffffffff} {
			raw := bytes.Clone(good)
			binary.LittleEndian.PutUint32(raw[offset:], value)
			wave6CRC(raw)
			reject(fmt.Sprintf("scale_%d_%d", offset, value), raw, "scales")
		}
	}
	for _, offset := range []int{32, 63, 95} {
		raw := bytes.Clone(good)
		raw[offset] = 8
		wave6CRC(raw)
		reject(fmt.Sprintf("map_%d", offset), raw, "map")
	}
	for _, cut := range []int{0, 4, 96, 119, 120, wave6Linear, wave6LinearBias, wave6Hidden, wave6HiddenBias, wave6Output, wave6OutputBias, len(good) - 4, len(good) - 1} {
		reject(fmt.Sprintf("truncated_%d", cut), good[:cut], "NGNN6")
	}
	// Corrupt header/map, each payload section and CRC without repairing CRC.
	for _, offset := range []int{32, 120, wave6Linear - 2, wave6Linear, wave6LinearBias, wave6Hidden, wave6HiddenBias, wave6Output, wave6OutputBias, len(good) - 1} {
		raw := bytes.Clone(good)
		raw[offset] ^= 1
		reject(fmt.Sprintf("crc_%d", offset), raw, "CRC")
	}
	reject("trailing_byte", append(bytes.Clone(good), 0), "trailing")
	reject("concatenated_files", append(bytes.Clone(good), good...), "trailing")

	// Max/min int32 endpoints with zero rows are accepted. Adding just one
	// signed weight at either endpoint exceeds the reserved arithmetic box.
	for _, stage := range []struct{ weight, bias, rows, width int }{
		{wave6Hidden, wave6HiddenBias, 16, 2 * wave6H},
		{wave6Output, wave6OutputBias, 8, 16},
	} {
		for _, sign := range []int{1, -1} {
			for _, row := range []int{0, stage.rows - 1} {
				raw := bytes.Clone(good)
				start := stage.weight + row*stage.width*2
				for i := start; i < start+stage.width*2; i++ {
					raw[i] = 0
				}
				bound := int64(1<<31 - 1)
				if sign < 0 {
					bound = -1 << 31
				}
				biasAt := stage.bias + row*4
				binary.LittleEndian.PutUint32(raw[biasAt:], uint32(bound))
				wave6Read(t, raw)
				binary.LittleEndian.PutUint16(raw[start:], uint16(int16(sign)))
				wave6CRC(raw)
				reject(fmt.Sprintf("arithmetic_%d_%d_%d", stage.bias, row, sign), bytes.Clone(raw), "arithmetic bound")
				binary.LittleEndian.PutUint32(raw[biasAt:], uint32(bound-int64(sign)*255))
				wave6Read(t, raw) // exact full-input endpoint is admissible
			}
		}
	}
}

type wave6Probe struct {
	header []byte
	reads  int
}

func (r *wave6Probe) Read(p []byte) (int, error) {
	r.reads++
	if len(r.header) == 0 {
		return 0, fmt.Errorf("unexpected payload read")
	}
	n := copy(p, r.header)
	r.header = r.header[n:]
	return n, nil
}

type wave6ReadError struct{}

func (wave6ReadError) Read([]byte) (int, error) { return 0, fmt.Errorf("source failed") }

func TestWave2ResidualFormatReaderBounds(t *testing.T) {
	raw := wave6Fixture(t)
	// Huge untrusted H is rejected without requesting any payload bytes.
	header := bytes.Clone(raw[:120])
	binary.LittleEndian.PutUint32(header[8:], 0xffffffff)
	probe := &wave6Probe{header: header}
	if n, err := ReadNGNN6(probe); n != nil || err == nil || !strings.Contains(err.Error(), "hidden width") || probe.reads != 1 {
		t.Fatalf("unbounded header handling: reads=%d err=%v", probe.reads, err)
	}
	// A source error after a complete file is not silently treated as EOF.
	reader := io.MultiReader(bytes.NewReader(raw), wave6ReadError{})
	if n, err := ReadNGNN6(reader); n != nil || err == nil || !strings.Contains(err.Error(), "source failed") {
		t.Fatalf("lost trailing read error: %v", err)
	}
	if n, err := ReadNGNN6(wave6ReadError{}); n != nil || err == nil || !strings.Contains(err.Error(), "source failed") {
		t.Fatalf("lost header read error: %v", err)
	}
}
