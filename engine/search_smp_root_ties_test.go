package engine

import (
	"reflect"
	"testing"
)

func TestSMPHelperRootTiesPreserveScoresAndTactics(t *testing.T) {
	for _, fen := range []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/1P3ppp/8/8/3p4/2N5/PPP2PPP/R3K2R w KQkq - 0 1",
	} {
		p, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		moves := GenerateMoves(p)
		var h workerHistory
		var baseline [256]Move
		n := h.orderMovesIntoBufferWithDepth(moves, 0, baseline[:], p)
		var scoreMoves [256]Move
		var scores [256]int
		h.scoreMovesIntoBuffer(moves, 0, EmptyMove, EmptyMove, EmptyMove, false, scoreMoves[:], scores[:], p, false)
		score := map[Move]int{}
		for i, m := range moves {
			score[m] = scores[i]
		}
		for _, worker := range []int{0, 1, 2, 3} {
			var out [256]Move
			if h.orderSMPHelperRootMoves(moves, out[:], p, worker) != n {
				t.Fatal("count changed")
			}
			got := map[Move]bool{}
			for i := 0; i < n; i++ {
				m := out[i]
				if got[m] {
					t.Fatal("duplicate")
				}
				got[m] = true
				if score[m] != score[baseline[i]] {
					t.Fatal("score band changed")
				}
				base := baseline[i]
				if (base.IsCapture() || base.IsEnPassant() || base.PromoType() != NoType || base.IsCastle()) && m != base {
					t.Fatal("tactical tie moved")
				}
			}
			var again [256]Move
			h.orderSMPHelperRootMoves(moves, again[:], p, worker)
			if !reflect.DeepEqual(out, again) {
				t.Fatal("nondeterministic tie order")
			}
			if worker == 0 && !reflect.DeepEqual(out, baseline) {
				t.Fatal("worker0 changed")
			}
		}
	}
}
func TestSMPHelperRootTiesSeparateColdHelpers(t *testing.T) {
	p := controlTestPosition(t)
	moves := GenerateMoves(p)
	var h workerHistory
	var orders [4][256]Move
	for worker := 0; worker < 4; worker++ {
		h.orderSMPHelperRootMoves(moves, orders[worker][:], p, worker)
		for earlier := 0; earlier < worker; earlier++ {
			if reflect.DeepEqual(orders[worker], orders[earlier]) {
				t.Fatal("cold helpers share order")
			}
		}
	}
	if !reflect.DeepEqual(h, workerHistory{}) {
		t.Fatal("ordering changed learning")
	}
}
