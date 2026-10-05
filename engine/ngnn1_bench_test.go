package engine

import (
	"bytes"
	"testing"
)

func ngnn1BenchNetwork(b *testing.B, h int) *NGNN1Network {
	b.Helper()
	n, err := ReadNGNN1(bytes.NewReader(ngnn1TestBytes(h, 19, false)))
	if err != nil {
		b.Fatal(err)
	}
	return n
}

func BenchmarkNGNN1Push(b *testing.B) {
	n := ngnn1BenchNetwork(b, 256)
	pos := newUCIStartingPosition()
	s := ngnn1State{network: n}
	s.reset(pos)
	s.ensure(1)
	delta := ngnn1MoveDelta(NewMove(E2, E4, WhitePawn, NoPiece, NoType, 0))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.push(1, delta)
	}
}

var ngnn1BenchScore int

func BenchmarkNGNN1Output(b *testing.B) {
	n := ngnn1BenchNetwork(b, 256)
	a := make([]int32, 2*n.hidden)
	n.refresh(newUCIStartingPosition(), a)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ngnn1BenchScore = n.evaluate(a, White)
	}
}
