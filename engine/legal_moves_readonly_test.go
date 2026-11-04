package engine

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

// serialLegalOracle follows the original serial filtering path on a separately
// parsed position. The explicit fixtures below independently constrain its set.
func serialLegalOracle(pos *Position) []Move {
	if !validatePosition(pos) {
		return []Move{}
	}
	pseudo := generateMovesUnsafe(pos)
	legal := make([]Move, 0, len(pseudo))
	for _, move := range pseudo {
		if IsLegalMove(pos, move) {
			legal = append(legal, move)
		}
	}
	return legal
}

func TestLegalMovesReadOnly(t *testing.T) {
	fixtures := []struct {
		name, fen, legal string
	}{
		{"initial", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "a2a3 a2a4 b1a3 b1c3 b2b3 b2b4 c2c3 c2c4 d2d3 d2d4 e2e3 e2e4 f2f3 f2f4 g1f3 g1h3 g2g3 g2g4 h2h3 h2h4"},
		{"castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 7 1", "a1a2 a1a3 a1a4 a1a5 a1a6 a1a7 a1a8 a1b1 a1c1 a1d1 e1c1 e1d1 e1d2 e1e2 e1f1 e1f2 e1g1 h1f1 h1g1 h1h2 h1h3 h1h4 h1h5 h1h6 h1h7 h1h8"},
		{"black_to_move", "4k3/8/8/8/8/8/8/4K3 b - - 0 1", "e8d7 e8d8 e8e7 e8f7 e8f8"},
		{"en_passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e1d1 e1d2 e1e2 e1f1 e1f2 e5d6 e5e6"},
		{"promotion", "4k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a7a8b a7a8n a7a8q a7a8r e1d1 e1d2 e1e2 e1f1 e1f2"},
		{"check_evasion", "k3r3/8/8/8/8/8/8/4K3 w - - 0 1", "e1d1 e1d2 e1f1 e1f2"},
		{"invalid", "8/8/8/8/8/8/8/4K3 w - - 0 1", ""},
		{"mate", "7k/6Q1/6K1/8/8/8/8/8 b - - 0 1", ""},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			pos, err := ParseFEN(fixture.fen)
			if err != nil {
				t.Fatal(err)
			}
			oraclePos, err := ParseFEN(fixture.fen)
			if err != nil {
				t.Fatal(err)
			}
			want := serialLegalOracle(oraclePos)
			var names []string
			for _, move := range want {
				names = append(names, move.ToString())
			}
			sort.Strings(names)
			if strings.Join(names, " ") != fixture.legal {
				t.Fatalf("oracle legal set = %q; want %q", strings.Join(names, " "), fixture.legal)
			}
			// Seed history and a lazy hash, and snapshot all state, including the
			// accumulator and mailbox contained in Board.
			pos.Positions = map[uint64]int{1: 3, 42: 2}
			pos.hash = 0
			board, side, ep, tag, hash, clock := pos.Board, pos.Side, pos.EnPassant, pos.Tag, pos.hash, pos.HalfMoveClock
			history := map[uint64]int{1: 3, 42: 2}
			const workers, rounds = 12, 24
			var ready, done sync.WaitGroup
			ready.Add(workers)
			done.Add(workers)
			start := make(chan struct{})
			failures := make(chan []Move, workers*rounds)
			for i := 0; i < workers; i++ {
				go func() {
					defer done.Done()
					ready.Done()
					<-start
					for j := 0; j < rounds; j++ {
						got := GenerateLegalMoves(pos)
						if !reflect.DeepEqual(got, want) {
							failures <- got
						}
					}
				}()
			}
			ready.Wait()
			close(start)
			done.Wait()
			close(failures)
			for got := range failures {
				t.Errorf("ordered moves = %v; want %v", got, want)
				break
			}
			if pos.Board != board || pos.Side != side || pos.EnPassant != ep || pos.Tag != tag || pos.hash != hash || pos.HalfMoveClock != clock || !reflect.DeepEqual(pos.Positions, history) {
				t.Fatal("query changed input position or history")
			}
		})
	}
}
