package engine

import "testing"

func TestExtraSearchTunables(t *testing.T) {
	if len(TunableSearchParams) < 44 {
		t.Fatal("missing search parameters")
	}
	for _, p := range TunableSearchParams {
		if *p.Ptr != p.Def || p.Def < p.Min || p.Def > p.Max {
			t.Fatalf("invalid default for %s", p.Name)
		}
	}
	old := lmrTable[20][20]
	if !setTunableParam("LMRBase", "100") || lmrTable[20][20] != old+1 {
		t.Fatal("LMR option did not rebuild table")
	}
	setTunableParam("LMRBase", "0")
	if lmrTable[20][20] != old {
		t.Fatal("LMR table did not restore")
	}
	before := lmpThresholdFor(4, false)
	setTunableParam("LMPQuadratic", "2")
	if lmpThresholdFor(4, false) <= before {
		t.Fatal("LMP option has no effect")
	}
	setTunableParam("LMPQuadratic", "1")
	setTunableParam("HistoryLMRDivisor", "500")
	if h1NormalizedLMRTerm(1000) != -2 {
		t.Fatal("history LMR option has no effect")
	}
	setTunableParam("HistoryLMRDivisor", "2500")
	var h workerHistory
	h.pawnCorrectionHistory[White][0] = 640
	setTunableParam("PawnCorrectionWeight", "4")
	if h.correctionValue(White, 0) != 40 {
		t.Fatal("correction option has no effect")
	}
	setTunableParam("PawnCorrectionWeight", "3")
}
