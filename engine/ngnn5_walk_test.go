package engine

import (
	"math/rand"
	"reflect"
	"testing"
)

// Mailbox ray oracle, independent of engine attack tables and update selection.
func ngnn5Oracle(n *NGNN1Network, pos *Position) []int32 {
	acc, _ := ngnn3Oracle(n, pos)
	for from, attacker := range pos.Board.mailbox {
		if !validPiece(attacker) {
			continue
		}
		color := attacker.Color()
		p := 0
		if color == Black {
			p = 1
		}
		king := 0
		for sq, piece := range pos.Board.mailbox {
			if piece == GetPiece(King, color) {
				king = sq
				break
			}
		}
		flip := 0
		if color == Black {
			flip = 56
		}
		king ^= flip
		mirror := 0
		if king%8 >= 4 {
			mirror = 7
			king ^= 7
		}
		bucket := int(n.kingMap[king])
		for to, victim := range pos.Board.mailbox {
			if !validPiece(victim) || to == from {
				continue
			}
			dr, df := to/8-from/8, to%8-from%8
			ar, af := abs(dr), abs(df)
			attack := false
			switch attacker.Type() {
			case Pawn:
				forward := 1
				if color == Black {
					forward = -1
				}
				attack = dr == forward && af == 1 && from/8 > 0 && from/8 < 7
			case Knight:
				attack = ar*af == 2
			case King:
				attack = ar <= 1 && af <= 1 && victim.Type() != King
			case Bishop:
				attack = ar == af
			case Rook:
				attack = dr == 0 || df == 0
			case Queen:
				attack = ar == af || dr == 0 || df == 0
			}
			if !attack {
				continue
			}
			if attacker.Type() == Bishop || attacker.Type() == Rook || attacker.Type() == Queen {
				rstep, fstep := 0, 0
				if dr > 0 {
					rstep = 1
				} else if dr < 0 {
					rstep = -1
				}
				if df > 0 {
					fstep = 1
				} else if df < 0 {
					fstep = -1
				}
				for r, f := from/8+rstep, from%8+fstep; r != to/8 || f != to%8; r, f = r+rstep, f+fstep {
					if pos.Board.mailbox[r*8+f] != NoPiece {
						attack = false
						break
					}
				}
			}
			if !attack {
				continue
			}
			v := int(victim.Type()) - 1
			if victim.Color() != color {
				v += 6
			}
			index := ngnn5Layout[int(attacker.Type())-1][from^flip^mirror][v][to^flip^mirror]
			if index < 0 {
				continue
			}
			for i, w := range n.threatWeights[uint32(bucket*ngnn5Features+int(index))] {
				acc[p*n.hidden+i] += int32(w)
			}
		}
	}
	return acc
}

func ngnn5Assert(t *testing.T, e *workerEvaluator, pos *Position) {
	t.Helper()
	n := e.nnue.network
	if n.version != 5 || e.Identity().backendName() != "ngnn5" || len(n.threatWeights) == 0 {
		t.Fatal("walk requires NGNN5 with nonzero threat rows")
	}
	full := make([]int32, 2*n.hidden)
	n.refresh(pos, full)
	oracle := ngnn5Oracle(n, pos)
	if !reflect.DeepEqual(full, oracle) || !reflect.DeepEqual(full, e.nnue.at(e.depth)) {
		t.Fatalf("NGNN5 refresh/oracle/incremental mismatch depth %d FEN %s", e.depth, GenerateFEN(pos))
	}
	if e.SearchSTM(pos) != n.evaluateBucket(oracle, pos.Turn(), n.bucket(pos)) {
		t.Fatal("NGNN5 incremental score mismatch")
	}
}

func TestNGNN5LongWalk(t *testing.T) {
	n := ngnn5TestNetwork(t, true)
	s, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigureThreads(1); err != nil {
		t.Fatal(err)
	}
	if err := s.SelectNGNN1Evaluator(n); err != nil {
		t.Fatal(err)
	}
	if s.ThreadCount() != 1 || s.SelectedEvaluatorBackend() != "ngnn5" {
		t.Fatal("walk format/Threads mismatch")
	}
	rng := rand.New(rand.NewSource(1104))
	type undo struct {
		move  Move
		ep    Square
		tag   PositionState
		clock uint8
	}
	seeds := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
		"r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
		"4k3/8/8/3pP3/8/8/PP6/4K3 w - d6 0 1",
		"4k3/pp6/8/8/3Pp3/8/8/4K3 b - d3 0 1",
		"1r2k3/P7/8/8/8/8/PP6/4K3 w - - 0 1",
		"4k3/pp6/8/8/8/8/p7/1R2K3 b - - 0 1",
	}
	positions, captures, promotions, castles, epMoves, refreshes, nulls := 0, 0, 0, 0, 0, 0, 0
	checkMove := func(e *workerEvaluator, pos *Position, move Move) undo {
		tr := e.mustPrepareMove(pos, move)
		if tr.threatBefore == nil {
			t.Fatal("NGNN3 piece-only transition substituted")
		}
		if tr.delta.refreshMask != 0 {
			refreshes++
		}
		ep, tag, clock, _ := pos.MakeMove(move)
		e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
		ngnn5Assert(t, e, pos)
		if move.CapturedPiece() != NoPiece {
			captures++
		}
		if move.PromoType() != NoType {
			promotions++
		}
		if move.IsCastle() {
			castles++
		}
		if move.IsEnPassant() {
			epMoves++
		}
		return undo{move, ep, tag, clock}
	}
	unmake := func(e *workerEvaluator, pos *Position, u undo) {
		e.mustPop()
		pos.UnMakeMove(u.move, u.tag, u.ep, u.clock)
		ngnn5Assert(t, e, pos)
	}
	// Enumerate all legal seed moves first to guarantee every special case,
	// including underpromotions of either color and all four castles.
	for _, fen := range seeds {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		e := s.mustPreparePrimaryEvaluator(pos, mustAcquireHCEModelUse())
		releaseHCEModelUse()
		ngnn5Assert(t, e, pos)
		for _, move := range GenerateLegalMoves(pos) {
			for visit := 0; visit < 2; visit++ {
				u := checkMove(e, pos, move)
				unmake(e, pos, u)
			}
		}
	}
	for game := 0; positions < 20000; game++ {
		pos, err := ParseFEN(seeds[game%len(seeds)])
		if err != nil {
			t.Fatal(err)
		}
		e := s.mustPreparePrimaryEvaluator(pos, mustAcquireHCEModelUse())
		releaseHCEModelUse()
		ngnn5Assert(t, e, pos)
		var history []undo
		for ply := 0; ply < 100 && positions < 20000; ply++ {
			moves := GenerateLegalMoves(pos)
			if len(moves) == 0 {
				break
			}
			if ply%13 == 0 && !pos.IsInCheck() {
				tr := e.mustPrepareNull(pos)
				ep := pos.MakeNullMove()
				e.mustPushMadeNull(pos, tr, ep)
				ngnn5Assert(t, e, pos)
				// A real child after a null alias must not overwrite its ancestor.
				children := GenerateLegalMoves(pos)
				if len(children) > 0 {
					u := checkMove(e, pos, children[rng.Intn(len(children))])
					unmake(e, pos, u)
				}
				e.mustPop()
				pos.UnMakeNullMove(ep)
				ngnn5Assert(t, e, pos)
				nulls++
			}
			history = append(history, checkMove(e, pos, moves[rng.Intn(len(moves))]))
			positions++
		}
		for i := len(history) - 1; i >= 0; i-- {
			unmake(e, pos, history[i])
		}
	}
	if captures == 0 || promotions == 0 || castles == 0 || epMoves == 0 || refreshes == 0 || nulls == 0 {
		t.Fatal("missing transition coverage")
	}
	t.Logf("format=NGNN5 Threads=1 positions=%d captures=%d promotions=%d castles=%d en-passant=%d context-refresh=%d nulls=%d; refresh+mailbox oracle after every make/unmake", positions, captures, promotions, castles, epMoves, refreshes, nulls)
}
