package engine

import (
	"math/rand"
	"testing"
)

// TestNGNN3LongWalk exercises the existing baseline over many make/unmake
// transitions. It is also a regression baseline for future input formats.
func TestNGNN3LongWalk(t *testing.T) {
	rng := rand.New(rand.NewSource(1104))
	n := ngnn3TestNetwork(t, 32, false)
	type undo struct {
		move  Move
		ep    Square
		tag   PositionState
		clock uint8
	}
	positions := 0
	for positions < 20000 {
		pos := newUCIStartingPosition()
		e := ngnn1NewWorker(t, n, pos)
		var history []undo
		for ply := 0; ply < 100 && positions < 20000; ply++ {
			moves := GenerateLegalMoves(pos)
			if len(moves) == 0 {
				break
			}
			move := moves[rng.Intn(len(moves))]
			tr := e.mustPrepareMove(pos, move)
			ep, tag, clock, _ := pos.MakeMove(move)
			e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
			history = append(history, undo{move, ep, tag, clock})
			ngnn3Assert(t, e, pos)
			positions++
		}
		for i := len(history) - 1; i >= 0; i-- {
			u := history[i]
			e.mustPop()
			pos.UnMakeMove(u.move, u.tag, u.ep, u.clock)
			ngnn3Assert(t, e, pos)
		}
	}
}
