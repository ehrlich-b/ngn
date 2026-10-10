package engine

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNGNNNullFrameOwnership(t *testing.T) {
	for _, version := range []int{1, 2, 3} {
		for _, hidden := range []int{16, 512, 2048} {
			n := ngnn1TestNetwork(t, hidden, true)
			if version == 2 {
				var err error
				n, err = ReadNGNN1(bytes.NewReader(ngnn2TestBytes(hidden, 8, true)))
				if err != nil {
					t.Fatal(err)
				}
			} else if version == 3 {
				n = ngnn3TestNetwork(t, hidden, true)
			}
			pos := newUCIStartingPosition()
			e := ngnn1NewWorker(t, n, pos)
			assert := func() {
				if version == 3 {
					ngnn3Assert(t, e, pos)
				} else if version == 2 {
					ngnn2AssertRefresh(t, e, pos)
				} else {
					ngnn1AssertRefresh(t, e, pos)
				}
			}
			root := append([]int32(nil), e.nnue.at(0)...)
			first := GenerateLegalMoves(pos)[0]
			trFirst := e.mustPrepareMove(pos, first)
			epFirst, tagFirst, clockFirst, _ := pos.MakeMove(first)
			e.mustPushMadeMove(pos, trFirst, first, tagFirst, epFirst, clockFirst)
			anchor := append([]int32(nil), e.nnue.at(1)...)
			// Cross two reallocations while every null ply shares a non-root slot.
			var nullEP []Square
			for ply := 2; ply <= 261; ply++ {
				tr := e.mustPrepareNull(pos)
				ep := pos.MakeNullMove()
				e.mustPushMadeNull(pos, tr, ep)
				nullEP = append(nullEP, ep)
				if &e.nnue.at(ply)[0] != &e.nnue.at(1)[0] {
					t.Fatalf("NGNN%d H%d ply%d copied a null accumulator", version, hidden, ply)
				}
				if !reflect.DeepEqual(anchor, e.nnue.at(ply)) {
					t.Fatal("growth changed an aliased accumulator")
				}
			}
			// A real descendant must materialize its own frame; revisiting the
			// same logical ply after a null must do the same for every sibling.
			for _, move := range GenerateLegalMoves(pos) {
				tr := e.mustPrepareMove(pos, move)
				ep, tag, clock, _ := pos.MakeMove(move)
				e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
				if &e.nnue.at(e.depth)[0] == &e.nnue.at(1)[0] {
					t.Fatal("real descendant shares writable ancestor storage")
				}
				assert()
				if !reflect.DeepEqual(root, e.nnue.at(0)) || !reflect.DeepEqual(anchor, e.nnue.at(1)) {
					t.Fatal("real descendant changed root accumulator")
				}
				e.mustPop()
				pos.UnMakeMove(move, tag, ep, clock)
				trNull := e.mustPrepareNull(pos)
				epNull := pos.MakeNullMove()
				e.mustPushMadeNull(pos, trNull, epNull)
				e.mustPop()
				pos.UnMakeNullMove(epNull)
			}
			for i := len(nullEP) - 1; i >= 0; i-- {
				e.mustPop()
				pos.UnMakeNullMove(nullEP[i])
				assert()
			}
			e.mustPop()
			pos.UnMakeMove(first, tagFirst, epFirst, clockFirst)
			if err := e.Reset(pos); err != nil {
				t.Fatal(err)
			}
			assert()
			for _, move := range GenerateLegalMoves(pos) {
				tr := e.mustPrepareMove(pos, move)
				ep, tag, clock, _ := pos.MakeMove(move)
				e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
				assert()
				if &e.nnue.at(1)[0] == &e.nnue.at(0)[0] {
					t.Fatal("reset left a writable sibling aliased to the root")
				}
				e.mustPop()
				pos.UnMakeMove(move, tag, ep, clock)
			}
		}
	}
}
