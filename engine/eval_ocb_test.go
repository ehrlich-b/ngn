package engine

import "testing"

// TestOCBEndgameDrawScaling locks the opposite-colored-bishop draw-scale (eval-gap
// rank 1): it must (a) detect a pure OCB ending and pull its eval toward the draw,
// and (b) leave every non-OCB position byte-identical regardless of ocbScaleEG.
func TestOCBEndgameDrawScaling(t *testing.T) {
	saved := ocbScaleEG
	defer func() { ocbScaleEG = saved }()
	mag := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}

	// Pure OCB ending, White up a pawn: Bd3 (light) vs Be7 (dark), pawns both sides.
	// A textbook draw NGN would otherwise score at ~full pawn-up.
	ocb := "4k3/4bp2/8/8/8/3B4/P4P2/4K3 w - - 0 1"
	pos, err := ParseFEN(ocb)
	if err != nil {
		t.Fatal(err)
	}
	ocbScaleEG = 32
	if got := ocbScale(&pos.Board); got != ocbScaleEG {
		t.Fatalf("ocbScale failed to detect the OCB ending: got %d, want %d (check the FEN's bishop colors)", got, ocbScaleEG)
	}
	ocbScaleEG = 64
	full := Evaluate(&pos.Board)
	ocbScaleEG = 32
	scaled := Evaluate(&pos.Board)
	if mag(scaled) >= mag(full) {
		t.Errorf("OCB eval not pulled toward draw: full=%d scaled=%d", full, scaled)
	}

	// Non-OCB position: ocbScale must be identity and the eval invariant to ocbScaleEG.
	for _, fen := range []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", // startpos
		"4k3/4bp2/8/8/8/4B3/P4P2/4K3 w - - 0 1",                    // same-colored bishops (Be3/Be7 both dark)
		"4k3/5p2/8/8/8/3B4/P4P2/2N1K3 w - - 0 1",                   // has a knight -> not pure OCB
	} {
		p, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("bad FEN %q: %v", fen, err)
		}
		if got := ocbScale(&p.Board); got != 64 {
			t.Errorf("ocbScale fired on a non-OCB position %q (got %d)", fen, got)
		}
		ocbScaleEG = 64
		a := Evaluate(&p.Board)
		ocbScaleEG = 32
		b := Evaluate(&p.Board)
		if a != b {
			t.Errorf("non-OCB eval changed with ocbScaleEG for %q: %d vs %d", fen, a, b)
		}
	}
}
