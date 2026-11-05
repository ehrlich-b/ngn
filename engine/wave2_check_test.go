package engine

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

type wave2CheckWitness struct {
	Move    string
	Check   bool
	Illegal bool // Giving check must not imply own-king legality.
}

type wave2CheckFixture struct {
	Name, FEN string
	Witnesses []wave2CheckWitness
}

// Keep all position fields except the synchronization primitive, including
// mailbox/material phase in Board and the raw (possibly cold) cached hash.
type wave2CheckSnapshot struct {
	Board     Bitboard
	Side      Color
	EP        Square
	Tag       PositionState
	Hash      uint64
	Clock     uint8
	Positions map[uint64]int
}

func wave2Snapshot(pos *Position) wave2CheckSnapshot {
	history := make(map[uint64]int, len(pos.Positions))
	for k, v := range pos.Positions {
		history[k] = v
	}
	return wave2CheckSnapshot{pos.Board, pos.Side, pos.EnPassant, pos.Tag, pos.hash, pos.HalfMoveClock, history}
}

// Every raw generated move is classified, even if it exposes the mover's king.
// The oracle owns a separate position and makes/unmakes the complete move.
func wave2CompareMoves(t *testing.T, pos *Position, evaluator, savedEvaluator *hceEvaluator, witnesses []wave2CheckWitness) []Move {
	t.Helper()
	before := wave2Snapshot(pos)
	oracle := pos.Copy()
	oracle.Hash() // Make/unmake's incremental key requires a warm oracle root.
	oracleBefore := wave2Snapshot(oracle)
	score := evaluator.SearchSTM(oracle)
	*savedEvaluator = *evaluator

	var buffer [256]Move
	n := GenerateMovesIntoBuffer(pos, buffer[:])
	moves := buffer[:n]
	sort.Slice(moves, func(i, j int) bool { return moves[i].ToString() < moves[j].ToString() })
	seen := make(map[string]bool, len(witnesses))
	var legal []Move
	checking, nonchecking := 0, 0
	for _, move := range moves {
		got := wave2GivesCheck(&pos.Board, move)
		if !reflect.DeepEqual(wave2Snapshot(pos), before) {
			t.Fatalf("predicate mutated caller after %s", move.ToString())
		}
		ep, tag, clock, _ := oracle.MakeMove(move)
		want := isInCheck(oracle, move.MovingPiece().Color().Other())
		ownKingSafe := !isInCheck(oracle, move.MovingPiece().Color())
		oracle.UnMakeMove(move, tag, ep, clock)
		if !reflect.DeepEqual(wave2Snapshot(oracle), oracleBefore) {
			t.Fatalf("oracle failed to restore %s", move.ToString())
		}
		if got != want {
			t.Errorf("%s: predicate=%v, make/isInCheck/unmake=%v; FEN %s", move.ToString(), got, want, GenerateFEN(pos))
		}
		for _, witness := range witnesses {
			if witness.Move == move.ToString() {
				seen[witness.Move] = true
				if want != witness.Check || (witness.Illegal && ownKingSafe) {
					t.Errorf("witness %s: check=%v ownKingSafe=%v; want check=%v illegal=%v", witness.Move, want, ownKingSafe, witness.Check, witness.Illegal)
				}
			}
		}
		if want {
			checking++
		} else {
			nonchecking++
		}
		if ownKingSafe && (!move.IsCastle() || IsLegalMove(oracle, move)) {
			legal = append(legal, move)
		}
	}
	for _, witness := range witnesses {
		if !seen[witness.Move] {
			t.Errorf("witness %s was not generated", witness.Move)
		}
	}
	if !reflect.DeepEqual(wave2Snapshot(pos), before) || !reflect.DeepEqual(wave2Snapshot(oracle), oracleBefore) {
		t.Fatal("move comparison changed caller or oracle state")
	}
	if *evaluator != *savedEvaluator || evaluator.SearchSTM(oracle) != score {
		t.Fatal("query changed evaluator cache/generation or root score")
	}
	t.Logf("pseudo=%d checking=%d nonchecking=%d legal=%d coldHash=%v; caller/hash/phase/history/evaluator restored", n, checking, nonchecking, len(legal), before.Hash == 0)
	return legal
}

func TestWave2GivesCheckCorpus(t *testing.T) {
	_ = mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	data, err := os.ReadFile("../experiments/wave2-1104/check-classify/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []wave2CheckFixture
	if err := json.Unmarshal(data, &fixtures); err != nil || len(fixtures) == 0 {
		t.Fatalf("invalid corpus: %v", err)
	}
	evaluator, savedEvaluator := &hceEvaluator{}, &hceEvaluator{}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			for _, cold := range []bool{false, true} {
				pos, err := ParseFEN(fixture.FEN)
				if err != nil {
					t.Fatal(err)
				}
				pos.Positions[42] = 3 // Nontrivial caller history must survive.
				if cold {
					pos.hash = 0
				}
				wave2CompareMoves(t, pos, evaluator, savedEvaluator, fixture.Witnesses)
			}
		})
	}
}

func TestWave2GivesCheckWalk(t *testing.T) {
	_ = mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	pos, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	evaluator, savedEvaluator := &hceEvaluator{}, &hceEvaluator{}
	const maxPlies = 128
	state := uint64(1104)
	for ply := 0; ply < maxPlies; ply++ {
		moves := wave2CompareMoves(t, pos, evaluator, savedEvaluator, nil)
		if len(moves) == 0 {
			t.Logf("seed=1104 ended at ply=%d (cap=%d)", ply, maxPlies)
			return
		}
		// Fixed xorshift and sorted legal moves make the walk reproducible
		// independently of math/rand and map iteration implementations.
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		move := moves[state%uint64(len(moves))]
		pos.GameMakeMove(move)
	}
	t.Logf("seed=1104 completed %d plies", maxPlies)
}
