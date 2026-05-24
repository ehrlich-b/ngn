package engine

import "testing"

// TestSEECorrectness checks staticExchangeEvaluation against hand-computed
// exchange values on crafted positions. Piece weights: P=100 N=320 B=330 R=525
// Q=1000. SEE feeds capture ordering AND pruning (SEE-prune, probcut, qsearch
// delta), so a bug here corrupts the search quietly across the board.
func TestSEECorrectness(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		uci  string // move in UCI (matches Move.ToString())
		want int
	}{
		{
			name: "pawn takes undefended knight",
			fen:  "4k3/8/8/3n4/4P3/8/8/4K3 w - - 0 1",
			uci:  "e4d5", want: 320,
		},
		{
			name: "pawn takes pawn defended by pawn (equal)",
			fen:  "4k3/8/2p5/3p4/4P3/8/8/4K3 w - - 0 1",
			uci:  "e4d5", want: 0,
		},
		{
			name: "queen takes pawn defended by pawn (lose queen for pawn)",
			fen:  "4k3/8/2p5/3p4/8/8/8/3QK3 w - - 0 1",
			uci:  "d1d5", want: 100 - 1000,
		},
		{
			name: "knight takes pawn defended by pawn",
			fen:  "4k3/8/4p3/3p4/8/2N5/8/4K3 w - - 0 1",
			uci:  "c3d5", want: 100 - 320,
		},
		{
			name: "rook battery x-ray recapture (win pawn, trade rooks)",
			fen:  "4r1k1/8/8/4p3/8/4R3/8/4R1K1 w - - 0 1",
			uci:  "e3e5", want: 100,
		},
		{
			name: "pawn promotes capturing undefended rook",
			fen:  "r5k1/1P6/8/8/8/8/8/6K1 w - - 0 1",
			uci:  "b7a8q", want: 525 + (1000 - 100),
		},
		{
			name: "en passant captures undefended pawn",
			fen:  "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1",
			uci:  "e5d6", want: 100,
		},
	}

	for _, c := range cases {
		pos, err := ParseFEN(c.fen)
		if err != nil {
			t.Errorf("%s: bad FEN %q: %v", c.name, c.fen, err)
			continue
		}
		var move Move = EmptyMove
		for _, m := range GenerateLegalMoves(pos) {
			if m.ToString() == c.uci {
				move = m
				break
			}
		}
		if move == EmptyMove {
			t.Errorf("%s: move %s not legal in %q", c.name, c.uci, c.fen)
			continue
		}
		got := staticExchangeEvaluation(pos, move)
		if got != c.want {
			t.Errorf("%s: SEE(%s)=%d, want %d   [%s]", c.name, c.uci, got, c.want, c.fen)
		}
	}
}
