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
	if r := lmrCaptureReduction(pos, move, -1, 12, 20, false); r != 0 {
		t.Fatalf("off: %d", r)
	}
	T1104_LMR_CAPTURES = 1
	if r := lmrCaptureReduction(pos, move, -1, 12, 20, true); r != 0 {
		t.Fatalf("PV winner: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, 0, 12, 20, false); r != 0 {
		t.Fatalf("neutral history: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, -1, 12, 2, false); r != 0 {
		t.Fatalf("early move: %d", r)
	}
	if r := lmrCaptureReduction(pos, move, -1, 12, 20, false); r <= 0 || r >= lmrTable[12][20] {
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
	if r := lmrCaptureReduction(losingPos, losing, -1, 12, 20, true); r <= 0 {
		t.Fatalf("PV losing capture: %d", r)
	}

}

func TestLMRCapturesSearch(t *testing.T) {
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
