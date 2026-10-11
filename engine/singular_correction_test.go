package engine

import "testing"

func TestSingularCorrectionBounds(t *testing.T) {
	for _, side := range []struct{ name, fen, excluded string }{
		{"white", "7k/8/8/8/8/8/8/KR6 w - - 0 1", "b1b8"},
		{"black", "kr6/8/8/8/8/8/8/7K b - - 0 1", "b8b1"},
	} {
		for _, c := range []struct {
			name                string
			restricted          bool
			oracle, alpha, beta int
			wantWrite           bool
		}{
			{"restricted fail low", true, 100, 49, 50, false},
			{"ordinary fail low", false, 100, 49, 50, true},
			{"restricted fail high", true, -100, -51, -50, true},
			{"ordinary fail high", false, -100, -51, -50, true},
			{"excluded other ply fail low", false, 100, 49, 50, true},
		} {
			t.Run(side.name+"/"+c.name, func(t *testing.T) {
				pos, info := newProbcutContractStateForFEN(t, side.fen, 1)
				excluded := adapterLegalMove(t, pos, side.excluded)
				rootHash := pos.Hash()
				info.evaluator.staticOracle = func(p *Position) int {
					if p.Hash() == rootHash {
						return c.oracle
					}
					return 0
				}
				if c.restricted {
					info.ExcludedMove, info.ExcludedPly = excluded, 0
				}
				if c.name == "excluded other ply fail low" {
					info.ExcludedMove, info.ExcludedPly = excluded, 2
				}
				stm := pos.Turn()
				p := pawnCorrectionIndex(pos.Board.pieces[WhitePawn], pos.Board.pieces[BlackPawn])
				n := nonPawnCorrectionIndex(pos.Board.WhiteOccupancy()&^pos.Board.pieces[WhitePawn], pos.Board.BlackOccupancy()&^pos.Board.pieces[BlackPawn])
				m := minorCorrectionIndex(pos.Board.pieces[WhiteKnight]|pos.Board.pieces[WhiteBishop], pos.Board.pieces[BlackKnight]|pos.Board.pieces[BlackBishop])
				got := alphaBetaPV(pos, 1, 0, c.alpha, c.beta, true, true, false, info)
				if info.Stopped {
					t.Fatal("search stopped")
				}
				if c.oracle > 0 && got >= c.beta {
					t.Fatalf("expected fail low, score=%d", got)
				}
				if c.oracle < 0 && got < c.beta {
					t.Fatalf("expected fail high, score=%d", got)
				}
				values := []int{info.history.pawnCorrectionHistory[stm][p], info.history.nonPawnCorrectionHistory[stm][n], info.history.minorCorrectionHistory[stm][m]}
				t.Logf("score=%d static=%d restricted=%v corrections=%v", got, c.oracle, c.restricted, values)
				for i, value := range values {
					if (value != 0) != c.wantWrite {
						t.Errorf("corrector %d=%d, want write=%v", i, value, c.wantWrite)
					}
				}
			})
		}
	}
}
