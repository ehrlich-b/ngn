package engine

import "testing"

func TestLMRCaptures(t *testing.T) {
	old := T1104_LMR_CAPTURES
	defer func() { T1104_LMR_CAPTURES = old }()
	pos, err := ParseFEN("4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	move := firstCapture(t, "4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1")
	if staticExchangeEvaluation(pos, move) <= 0 {
		t.Fatal("fixture must be SEE-winning")
	}
	T1104_LMR_CAPTURES = 0
	if r := lmrCaptureReduction(pos, move, -1, 12, 20, false, true, false); r != 0 {
		t.Fatalf("off: %d", r)
	}
	T1104_LMR_CAPTURES = 1
	if r := lmrCaptureReduction(pos, move, -1, 12, 20, true, true, false); r != 0 {
		t.Fatalf("PV winner: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, 0, 12, 20, false, true, false); r != 0 {
		t.Fatalf("neutral history: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, -1, 12, 2, false, true, false); r != 0 {
		t.Fatalf("early move: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, -1, 20, 50, false, true, false); r <= 0 || r >= quietLMRReduction(20, 50, false, true, 100000, false, true) {
		t.Fatalf("late capture reduction: %d", r)
	}
	losingFEN := "4k3/8/2p5/3p4/4Q3/8/8/4K3 w - - 0 1"
	losingPos, err := ParseFEN(losingFEN)
	if err != nil {
		t.Fatal(err)
	}
	losing := firstCapture(t, losingFEN)
	if staticExchangeEvaluation(losingPos, losing) > 0 {
		t.Fatal("fixture must not be SEE-winning")
	}
	if r := lmrCaptureReduction(losingPos, losing, -1, 20, 50, true, true, false); r <= 0 {
		t.Fatalf("PV losing capture: %d", r)
	}

	// Compare against all quiet adjustments, including the depth cap and the
	// shallow late-move case that previously produced equal reductions.
	for depth := 3; depth <= 20; depth++ {
		for moves := 3; moves <= 64; moves++ {
			for _, pv := range []bool{false, true} {
				for _, improving := range []bool{false, true} {
					for _, ttPV := range []bool{false, true} {
						r := lmrCaptureReduction(losingPos, losing, -1, depth, moves, pv, improving, ttPV)
						for _, history := range []int{-100000, 0, 100000} {
							for _, killer := range []bool{false, true} {
								q := quietLMRReduction(depth, moves, pv, improving, history, ttPV, killer)
								if r > 0 && r >= q {
									t.Fatalf("depth=%d moves=%d capture=%d quiet=%d", depth, moves, r, q)
								}
							}
						}
					}
				}
			}
		}
	}

}

func TestLMRCapturesSearch(t *testing.T) {
	originalSearchOptions(t)
	old := T1104_LMR_CAPTURES
	defer func() { T1104_LMR_CAPTURES = old }()
	T1104_LMR_CAPTURES = 1
	pos, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	info := Search(pos, 9)
	if info.LMRCaptureReductions == 0 {
		t.Fatal("search did not reduce any captures")
	}
}
