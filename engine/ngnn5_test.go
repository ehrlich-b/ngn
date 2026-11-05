package engine

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"math/rand"
	"testing"
)

func ngnn5TestNetwork(t *testing.T, threats bool) *NGNN1Network {
	t.Helper()
	n := ngnn3TestNetwork(t, 16, true)
	n.version, n.compactInput = 5, false
	n.threatWeights = make(map[uint32][]int8)
	if threats {
		rng := rand.New(rand.NewSource(51104))
		for row := 0; row < n.kingBuckets*ngnn5Features; row++ {
			w := make([]int8, n.hidden)
			for lane := range w {
				w[lane] = int8(rng.Intn(256) - 128)
			}
			n.threatWeights[uint32(row)] = w
		}
	}
	var out bytes.Buffer
	if err := writeNGNN5(&out, n); err != nil {
		t.Fatal(err)
	}
	got, err := ReadNGNN1(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got.version != 5 || got.backendName() != "ngnn5" || got.compactInput {
		t.Fatal("NGNN5 format identity lost")
	}
	return got
}

func TestNGNN5SparseValidation(t *testing.T) {
	n := ngnn3TestNetwork(t, 16, false)
	n.version = 5
	n.threatWeights = map[uint32][]int8{1: make([]int8, 16), 3: make([]int8, 16)}
	n.threatWeights[1][0], n.threatWeights[3][15] = -128, 127
	var out bytes.Buffer
	if err := writeNGNN5(&out, n); err != nil {
		t.Fatal(err)
	}
	good := out.Bytes()
	parsed, err := ReadNGNN1(bytes.NewReader(good))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.threatWeights) != 2 || parsed.threatWeights[1][0] != -128 || parsed.threatWeights[3][15] != 127 {
		t.Fatal("signed int8 sparse rows lost")
	}
	start := 112 + 2*n.kingBuckets*ngnn1Features*n.hidden
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], uint32(n.kingBuckets*ngnn5Features)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+20:], 1) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+20:], 0) },
		func(b []byte) { b[start+4] = 0 },
	} {
		bad := append([]byte(nil), good...)
		mutate(bad)
		binary.LittleEndian.PutUint32(bad[len(bad)-4:], crc32.ChecksumIEEE(bad[:len(bad)-4]))
		if got, err := ReadNGNN1(bytes.NewReader(bad)); got != nil || err == nil {
			t.Fatal("accepted noncanonical sparse rows")
		}
	}
}

func TestNGNN5ThreatIdentity(t *testing.T) {
	n := ngnn5TestNetwork(t, true)
	pos := newUCIStartingPosition()
	full, pieces := make([]int32, 2*n.hidden), make([]int32, 2*n.hidden)
	n.refresh(pos, full)
	copyNetwork := *n
	copyNetwork.version = 3
	copyNetwork.refresh(pos, pieces)
	equal := true
	for i := range full {
		if full[i] != pieces[i] {
			equal = false
		}
	}
	if equal {
		t.Fatal("NGNN5 nonzero threat block had no effect")
	}
	// Both colors normalize the same tuple and king context identically.
	a := ngnn5Feature(WhitePawn, D4, BlackKnight, E5, White, ngnn3Context{bucket: 2, mirror: 7})
	b := ngnn5Feature(BlackPawn, D5, WhiteKnight, E4, Black, ngnn3Context{bucket: 2, mirror: 7})
	if a < 0 || a != b {
		t.Fatal("threat perspective/mirror mismatch")
	}
	if ngnn5Feature(WhitePawn, D4, BlackKnight, D5, White, ngnn3Context{}) >= 0 || ngnn5Feature(WhiteKnight, D4, BlackKnight, D5, White, ngnn3Context{}) >= 0 {
		t.Fatal("impossible pair retained")
	}
}

func TestNGNN5ParserAndConverter(t *testing.T) {
	var out bytes.Buffer
	if err := ConvertNGNN3ToNGNN5(&out, bytes.NewReader(ngnn3TestBytes(16, 8, 8, false))); err != nil {
		t.Fatal(err)
	}
	good := out.Bytes()
	n, err := ReadNGNN1(bytes.NewReader(good))
	if err != nil || n.version != 5 || n.backendName() != "ngnn5" || len(n.threatWeights) != 0 {
		t.Fatalf("format/zero threats: %v", err)
	}
	for _, offset := range []int{0, 4, 8, 12, 16, 20, 24, 28, 32, 96, 100, 104, 108} {
		bad := append([]byte(nil), good...)
		bad[offset] ^= 255
		binary.LittleEndian.PutUint32(bad[len(bad)-4:], crc32.ChecksumIEEE(bad[:len(bad)-4]))
		if got, err := ReadNGNN1(bytes.NewReader(bad)); got != nil || err == nil {
			t.Fatalf("accepted header mutation %d", offset)
		}
	}
	for _, cut := range []int{0, 23, 24, 111, 112, len(good) - 1} {
		if got, err := ReadNGNN1(bytes.NewReader(good[:cut])); got != nil || err == nil {
			t.Fatalf("accepted truncation %d", cut)
		}
	}
	for _, bad := range [][]byte{append(append([]byte(nil), good...), 0), append([]byte(nil), good...)} {
		bad[len(bad)-1] ^= 1
		if got, err := ReadNGNN1(bytes.NewReader(bad)); got != nil || err == nil {
			t.Fatal("accepted CRC/trailing data")
		}
	}
	if err := ConvertNGNN3ToNGNN5(&bytes.Buffer{}, bytes.NewReader(good)); err == nil {
		t.Fatal("converted NGNN5 as NGNN3")
	}
}
