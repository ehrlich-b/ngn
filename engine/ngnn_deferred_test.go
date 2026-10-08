package engine

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func TestNGNNDeferredSequences(t *testing.T) {
	for version := 1; version <= 3; version++ {
		for _, extreme := range []bool{false, true} {
			t.Run(fmt.Sprintf("V%d/extreme=%t", version, extreme), func(t *testing.T) {
				data := ngnn1TestBytes(48, 19, extreme)
				if version == 2 {
					data = ngnn2TestBytes(48, 8, extreme)
				} else if version == 3 {
					data = ngnn3TestBytes(48, 8, 8, extreme)
				}
				n, err := ReadNGNN1(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				rng := rand.New(rand.NewSource(80113))
				for _, fen := range []string{
					"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
					"r3k2r/pp6/8/8/8/8/PP6/R3K2R w KQkq - 0 1",
					"4k3/8/8/3pP3/8/8/PP6/4K3 w - d6 0 1",
					"1r2k3/P7/8/8/8/8/PP6/4K3 w - - 0 1",
				} {
					pos, err := ParseFEN(fen)
					if err != nil {
						t.Fatal(err)
					}
					e := ngnn1NewWorker(t, n, pos)
					assert := func() {
						t.Helper()
						want := make([]int32, 2*n.hidden)
						n.refresh(pos, want)
						cp := ngnn2Oracle(n, pos)
						if version == 3 {
							want, cp = ngnn3Oracle(n, pos)
						}
						if got := e.SearchSTM(pos); got != cp || !reflect.DeepEqual(e.nnue.at(e.depth), want) {
							t.Fatalf("delayed accumulator/score mismatch at ply %d", e.depth)
						}
					}
					type undo struct {
						move  Move
						ep    Square
						tag   PositionTag
						clock uint8
					}
					var undos []undo
					pop := func(u undo) {
						e.mustPop()
						if u.move == 0 {
							pos.UnMakeNullMove(u.ep)
						} else {
							pos.UnMakeMove(u.move, u.tag, u.ep, u.clock)
						}
					}
					for ply := 0; ply < 180; ply++ {
						if ply%7 == 6 {
							assert()
						}
						if ply%9 == 4 && !pos.IsInCheck() {
							tr := e.mustPrepareNull(pos)
							ep := pos.MakeNullMove()
							e.mustPushMadeNull(pos, tr, ep)
							undos = append(undos, undo{ep: ep})
						} else {
							moves := GenerateLegalMoves(pos)
							if len(moves) == 0 {
								break
							}
							move := moves[rng.Intn(len(moves))]
							for _, candidate := range moves {
								if candidate.IsCastle() || candidate.IsEnPassant() || candidate.PromoType() != NoType || (ply%3 != 0 && candidate.MovingPiece().Type() == King) {
									move = candidate
									break
								}
							}
							tr := e.mustPrepareMove(pos, move)
							ep, tag, clock, _ := pos.MakeMove(move)
							e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
							undos = append(undos, undo{move, ep, tag, clock})
						}
						if e.nnue.frames[e.depth].ready != 0 {
							t.Fatal("made move computed an unused accumulator")
						}
						// Reuse a previously occupied frame without evaluating it.
						if ply%11 == 10 {
							pop(undos[len(undos)-1])
							undos = undos[:len(undos)-1]
							pop(undos[len(undos)-1])
							undos = undos[:len(undos)-1]
						}
					}
					assert()
					for i := len(undos) - 1; i >= 0; i-- {
						pop(undos[i])
						if i%5 == 0 {
							assert()
						}
					}
					assert()
					undos = nil
					// Force stack growth while every intermediate null frame is
					// unused, then evaluate and unwind across the allocation.
					for i := 0; i < 260; i++ {
						tr := e.mustPrepareNull(pos)
						ep := pos.MakeNullMove()
						e.mustPushMadeNull(pos, tr, ep)
						undos = append(undos, undo{ep: ep})
					}
					assert()
					for i := len(undos) - 1; i >= 0; i-- {
						pop(undos[i])
					}
					assert()
					if err := e.Reset(newUCIStartingPosition()); err != nil {
						t.Fatal(err)
					}
					pos = newUCIStartingPosition()
					assert()
				}
			})
		}
	}
}
