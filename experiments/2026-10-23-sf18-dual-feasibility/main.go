// Diagnostic operation accounting only; no tensors or search evaluator.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ehrlich-b/ngn/engine"
)

type sequence struct {
	ID      string   `json:"id"`
	FEN     string   `json:"fen"`
	Actions []string `json:"actions"`
}

type suite struct {
	CaseCount int        `json:"case_count"`
	RowCount  int        `json:"row_count"`
	Cases     []sequence `json:"cases"`
}

type corpus struct {
	Suites map[string]suite `json:"suites"`
}

type board [64]engine.Piece
type edge struct {
	From, To         int
	Attacker, Target engine.Piece
}
type edgeSet map[edge]bool

type row struct {
	Case                string `json:"case"`
	Action              string `json:"action"`
	Depth               int    `json:"depth"`
	Pieces              int    `json:"pieces"`
	Stack               int    `json:"stack"`
	OccupiedAttackEdges int    `json:"occupied_attack_edges_upper"`
	BaseColumns         int    `json:"base_columns"`
	ThreatColumnsUpper  int    `json:"threat_columns_upper"`
	BaseLaneAdds        int    `json:"dual_base_lane_adds"`
	ThreatLaneAddsUpper int    `json:"big_threat_lane_adds_upper"`
	PSQTAddsUpper       int    `json:"dual_psqt_adds_upper"`
	EagerCopyBytes      int    `json:"eager_numeric_copy_bytes"`
	SimpleEval          int    `json:"simple_eval_stm"`
	First               string `json:"first_network"`
	DenseMACsLower      int    `json:"selected_dense_macs_lower"`
	DenseMACsUpper      int    `json:"selected_dense_macs_upper"`
	PairProductsLower   int    `json:"pair_products_lower"`
	PairProductsUpper   int    `json:"pair_products_upper"`
}

func must(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func snapshot(p *engine.Position) (b board) {
	for sq := range b {
		b[sq] = p.Board.PieceAt(engine.Square(sq))
	}
	return
}

func king(b board, c engine.Color) int {
	for sq, pc := range b {
		if pc.Type() == engine.King && pc.Color() == c {
			return sq
		}
	}
	panic("king missing")
}

// A strict superset of FullThreats: every occupied geometric attack target,
// including friendly defenders and kings, with sliders stopped at first blocker.
// No release-specific piece-pair exclusions or feature-index map is asserted.
func edges(b board) edgeSet {
	r := edgeSet{}
	for from, pc := range b {
		if pc == engine.NoPiece {
			continue
		}
		x, y := from%8, from/8
		for to, target := range b {
			if to == from || target == engine.NoPiece {
				continue
			}
			dx, dy := to%8-x, to/8-y
			ax, ay := abs(dx), abs(dy)
			attack, slide := false, false
			switch pc.Type() {
			case engine.Pawn:
				step := 1
				if pc.Color() == engine.Black {
					step = -1
				}
				attack = ax == 1 && dy == step
			case engine.Knight:
				attack = ax*ay == 2
			case engine.King:
				attack = ax <= 1 && ay <= 1
			case engine.Bishop:
				slide = ax == ay
			case engine.Rook:
				slide = dx == 0 || dy == 0
			case engine.Queen:
				slide = ax == ay || dx == 0 || dy == 0
			}
			if slide {
				attack = true
				for a, z := x+sign(dx), y+sign(dy); a != to%8 || z != to/8; a, z = a+sign(dx), z+sign(dy) {
					if b[z*8+a] != engine.NoPiece {
						attack = false
						break
					}
				}
			}
			if attack {
				r[edge{from, to, pc, target}] = true
			}
		}
	}
	return r
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
func diff(a, b edgeSet) (n int) {
	for k := range a {
		if !b[k] {
			n++
		}
	}
	for k := range b {
		if !a[k] {
			n++
		}
	}
	return
}

func measure(id, action string, depth int, p *engine.Position, before board) row {
	b := snapshot(p)
	e := edges(b)
	r := row{Case: id, Action: action, Depth: depth, OccupiedAttackEdges: len(e)}
	values := [7]int{0, 208, 781, 825, 1276, 2538, 0}
	for _, pc := range b {
		if pc == engine.NoPiece {
			continue
		}
		r.Pieces++
		v := values[pc.Type()]
		if pc.Color() != p.Turn() {
			v = -v
		}
		r.SimpleEval += v
	}
	r.Stack = (r.Pieces - 1) / 4
	if action == "root" {
		r.BaseColumns = 2 * r.Pieces
		r.ThreatColumnsUpper = 2 * len(e)
	} else if action != "pop" {
		r.EagerCopyBytes = 8896
		changed := 0
		for sq, pc := range b {
			if pc != before[sq] {
				if pc != engine.NoPiece {
					changed++
				}
				if before[sq] != engine.NoPiece {
					changed++
				}
			}
		}
		oldEdges := edges(before)
		for _, c := range []engine.Color{engine.White, engine.Black} {
			oldKing, newKing := king(before, c), king(b, c)
			if oldKing != newKing {
				r.BaseColumns += r.Pieces
			} else {
				r.BaseColumns += changed
			}
			if (oldKing%8 < 4) != (newKing%8 < 4) {
				r.ThreatColumnsUpper += len(e)
			} else {
				r.ThreatColumnsUpper += diff(oldEdges, e)
			}
		}
	}
	r.BaseLaneAdds = r.BaseColumns * (1024 + 128)
	r.ThreatLaneAddsUpper = r.ThreatColumnsUpper * 1024
	r.PSQTAddsUpper = (2*r.BaseColumns + r.ThreatColumnsUpper) * 8
	r.First = "BIG"
	r.DenseMACsLower = 17440
	r.DenseMACsUpper = 17440
	r.PairProductsLower = 1024
	r.PairProductsUpper = 1024
	if abs(r.SimpleEval) > 962 {
		r.First = "SMALL"
		r.DenseMACsLower = 3104
		r.DenseMACsUpper = 3104 + 17440
		r.PairProductsLower = 128
		r.PairProductsUpper = 128 + 1024
	}
	// Each row quotes the cost if queried. Pop itself evaluates no head.
	return r
}

func replay(s sequence) []row {
	p, err := engine.ParseFEN(s.FEN)
	if err != nil {
		panic(err)
	}
	root := snapshot(p)
	rows := []row{measure(s.ID, "root", 0, p, board{})}
	type undo struct {
		move   engine.Move
		tag    engine.PositionTag
		ep     engine.Square
		clock  uint8
		before board
		turn   engine.Color
		hash   uint64
		fen    string
	}
	var stack []undo
	for i, action := range s.Actions {
		u := undo{before: snapshot(p), turn: p.Turn(), hash: p.Hash(), fen: engine.GenerateFEN(p)}
		if action == "null" {
			must(!p.IsInCheck(), "null while in check")
			u.ep = p.MakeNullMove()
		} else {
			for _, m := range engine.GenerateLegalMoves(p) {
				if engine.MoveToAlgebraic(m) == action {
					u.move = m
					break
				}
			}
			must(u.move != engine.EmptyMove, s.ID+": illegal "+action)
			var made bool
			u.ep, u.tag, u.clock, made = p.MakeMove(u.move)
			must(made, s.ID+": make failed")
		}
		stack = append(stack, u)
		rows = append(rows, measure(s.ID, action, i+1, p, u.before))
	}
	for i := len(stack) - 1; i >= 0; i-- {
		u := stack[i]
		if u.move == engine.EmptyMove {
			p.UnMakeNullMove(u.ep)
		} else {
			p.UnMakeMove(u.move, u.tag, u.ep, u.clock)
		}
		must(snapshot(p) == u.before && p.Turn() == u.turn && p.Hash() == u.hash, s.ID+": unwind mismatch")
		must(engine.GenerateFEN(p) == u.fen, s.ID+": full state unwind mismatch")
		rows = append(rows, measure(s.ID, "pop", i, p, board{}))
	}
	must(root == snapshot(p), "root mismatch")
	return rows
}

func main() {
	must(len(os.Args) == 3, "usage: cost-study upstream-sequences.json additional-sequences.json")
	// Independent, literal attack-geometry witnesses: friendly rook target,
	// blocked enemy rook target, both pawn directions, knight, bishop and king.
	var b board
	b[0] = engine.WhiteRook
	b[8] = engine.WhitePawn
	b[16] = engine.BlackRook
	e := edges(b)
	must(e[edge{0, 8, engine.WhiteRook, engine.WhitePawn}] && !e[edge{0, 16, engine.WhiteRook, engine.BlackRook}], "blocker witness")
	b = board{}
	b[27] = engine.WhitePawn
	b[36] = engine.BlackPawn
	e = edges(b)
	must(len(e) == 2, "pawn directions witness")
	b = board{}
	b[0] = engine.WhiteBishop
	b[18] = engine.BlackKnight
	b[35] = engine.WhiteKing
	e = edges(b)
	must(len(e) == 2, "bishop/knight witness")
	b = board{}
	b[0] = engine.WhiteKing
	b[1] = engine.WhiteRook
	e = edges(b)
	must(len(e) == 2, "king witness")
	var all []row
	cases := 0
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		var c corpus
		if err = json.Unmarshal(data, &c); err != nil {
			panic(err)
		}
		names := []string{}
		for name := range c.Suites {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			s := c.Suites[name]
			must(len(s.Cases) == s.CaseCount, "case count")
			count := 0
			for _, seq := range s.Cases {
				r := replay(seq)
				count += len(r)
				all = append(all, r...)
				cases++
			}
			must(count == s.RowCount, fmt.Sprintf("%s rows=%d expected=%d", name, count, s.RowCount))
		}
	}
	stacks := map[int]bool{}
	kinds := map[string]bool{}
	for _, r := range all {
		stacks[r.Stack] = true
		kinds[r.First] = true
	}
	must(len(stacks) == 8 && len(kinds) == 2, "material stack/selector coverage")
	must(len(all) > 0, "empty study")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(struct {
		Schema string `json:"schema"`
		Cases  int    `json:"cases"`
		Rows   []row  `json:"rows"`
	}{"sf18-dual-cost-bounds/v1", cases, all}); err != nil {
		panic(err)
	}
}
