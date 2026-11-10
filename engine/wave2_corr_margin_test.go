package engine

import (
	"encoding/json"
	"os"
	"testing"
)

func TestWave2CorrMarginOffDepth4(t *testing.T) {
	originalSearchOptions(t)
	old := W2CorrMargin
	W2CorrMargin = 0
	defer func() { W2CorrMargin = old }()
	data, err := os.ReadFile("../experiments/wave2-1104/corr-margin/off-depth4.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, FEN, Move string
		Score           int
		Nodes, QNodes   uint64
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	n, err := LoadNGNN1("nets/ngn-h768-2dcb6d93.nnue")
	if err != nil {
		t.Fatal(err)
	}
	if n.hidden != 768 {
		t.Fatal("fixture requires H768")
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			pos, err := ParseFEN(c.FEN)
			if err != nil {
				t.Fatal(err)
			}
			s, err := NewSearchEngineWithHash(1)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.SelectNGNN1Evaluator(n); err != nil {
				t.Fatal(err)
			}
			info := s.SearchFixed(pos, 4, nil)
			t.Logf("BASELINE %s %s %d %d %d", c.Name, info.BestMove.ToString(), info.BestScore, info.Nodes, info.QNodes)
			if info.BestMove.ToString() != c.Move || info.BestScore != c.Score || info.Nodes != c.Nodes || info.QNodes != c.QNodes {
				t.Fatalf("off mismatch: move=%s score=%d nodes=%d qnodes=%d", info.BestMove.ToString(), info.BestScore, info.Nodes, info.QNodes)
			}
		})
	}
}

func TestWave2CorrMarginFormula(t *testing.T) {
	old := W2CorrMargin
	defer func() { W2CorrMargin = old }()
	maxInt := int(^uint(0) >> 1)
	for _, c := range []struct {
		name                 string
		raw, corrected, want int
	}{
		{"zero", 100, 100, 0}, {"positive", 100, 201, 50}, {"negative", 201, 100, 50},
		{"below-cap", 0, 255, 127}, {"cap", 0, 256, 128}, {"above-cap", 0, 257, 128},
		{"negative-cap", 0, -256, 128}, {"largest-valid", -MATE_IN_MAX + 1, MATE_IN_MAX - 1, 128},
		{"unknown-raw", unknownStaticEval, 0, 0}, {"unknown-corrected", 0, unknownStaticEval, 0},
		{"mate-raw", MATE_IN_MAX, 0, 0}, {"mate-corrected", 0, MATE_IN_MAX, 0},
		{"negative-mate-raw", -MATE_IN_MAX, 0, 0}, {"negative-mate-corrected", 0, -MATE_IN_MAX, 0},
		{"max-int", maxInt, -maxInt - 1, 0}, {"min-int", -maxInt - 1, maxInt, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			W2CorrMargin = 0
			if got := wave2CorrMargin(c.raw, c.corrected); got != 0 {
				t.Fatalf("off=%d", got)
			}
			W2CorrMargin = 1
			if got := wave2CorrMargin(c.raw, c.corrected); got != c.want {
				t.Fatalf("got=%d want=%d", got, c.want)
			}
		})
	}
}

func TestWave2CorrMarginSearch(t *testing.T) {
	old, oldMore := W2CorrMargin, T1104CorrMore
	defer func() { W2CorrMargin, T1104CorrMore = old, oldMore }()
	for _, c := range []struct {
		name                                    string
		enabled, correction, more, raw, ttScore int
		tt                                      bool
		bound                                   NodeType
		inCheck, prune                          bool
		slack                                   int
	}{
		{name: "off-positive", correction: 1024, raw: 300, prune: true, slack: 1},
		{name: "off-negative", correction: -1024, raw: 300, prune: true, slack: 1},
		{name: "zero", enabled: 1, raw: 300, prune: true, slack: 1},
		{name: "positive", enabled: 1, correction: 1024, raw: 300, slack: 1},
		{name: "negative", enabled: 1, correction: -1024, raw: 300, slack: 1},
		{name: "tt-exact-positive", enabled: 1, correction: 1024, raw: 300, tt: true, ttScore: 310, bound: Exact, slack: 10},
		{name: "tt-exact-negative", enabled: 1, correction: -1024, raw: 300, tt: true, ttScore: 290, bound: Exact, slack: 10},
		{name: "tt-lower-positive", enabled: 1, correction: 1024, raw: 300, tt: true, ttScore: 360, bound: LowerBound, slack: 10},
		{name: "tt-upper-negative", enabled: 1, correction: -1024, raw: 300, tt: true, ttScore: 240, bound: UpperBound, slack: 10},
		{name: "tt-ignored-mate", enabled: 1, correction: 1024, raw: 300, tt: true, ttScore: MATE_IN_MAX, bound: Exact, slack: 1},
		{name: "tt-ignored-negative-mate", enabled: 1, correction: -1024, raw: 300, tt: true, ttScore: -MATE_IN_MAX, bound: Exact, slack: 1},
		{name: "tt-zero-delta", enabled: 1, raw: 300, tt: true, ttScore: 500, bound: Exact, prune: true, slack: 1},
		{name: "corr-more", enabled: 1, more: 1, raw: 300, slack: 1},
		{name: "in-check", enabled: 1, correction: 1024, raw: 300, inCheck: true, slack: 1},
		{name: "mate-raw", enabled: 1, correction: -1024, raw: MATE_IN_MAX, prune: true, slack: 1},
		{name: "mate-corrected", enabled: 1, correction: 1024, raw: MATE_IN_MAX - 40, prune: true, slack: 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			W2CorrMargin, T1104CorrMore = c.enabled, c.more
			fen := "4k3/p7/8/8/8/8/8/R3K3 w - - 0 1"
			if c.inCheck {
				fen = "4r1k1/8/8/8/8/8/P7/4K3 w - - 0 1"
			}
			pos, info := newProbcutContractStateForFEN(t, fen, 1)
			hash := pos.Hash()
			rootEvals := 0
			info.evaluator.staticOracle = func(p *Position) int {
				if p.Hash() == hash {
					rootEvals++
					return c.raw
				}
				return 0
			}
			idx := pawnCorrectionIndex(pos.Board.pieces[WhitePawn], pos.Board.pieces[BlackPawn])
			info.history.pawnCorrectionHistory[pos.Turn()][idx] = c.correction
			if c.more != 0 {
				info.history.majorCorrectionHistory[pos.Turn()][majorCorrectionIndex(pos)] = 1024
			}
			corrected := c.raw + info.history.correctionValue(pos.Turn(), idx) + info.history.moreCorrection(pos, EmptyMove)
			static := corrected
			if c.tt {
				info.tt.Set(hash, EmptyMove, int16(c.ttScore), 0, c.bound, false)
				if c.ttScore > -MATE_IN_MAX && c.ttScore < MATE_IN_MAX {
					static = c.ttScore
				}
			}
			beta := static - REVERSE_FUTILITY_MARGIN - c.slack
			got := alphaBetaPV(pos, 1, 0, beta-1, beta, false, false, false, info)
			if info.Stopped {
				t.Fatal("search stopped")
			}
			if c.prune {
				if info.RFPPrunes != 1 || info.MoveLoopNodes != 0 || got != static-REVERSE_FUTILITY_MARGIN {
					t.Fatalf("expected root cutoff: score=%d RFP=%d loops=%d", got, info.RFPPrunes, info.MoveLoopNodes)
				}
			} else if info.MoveLoopNodes == 0 || info.RFPPrunes != 0 {
				t.Fatalf("expected continued root search: score=%d RFP=%d loops=%d", got, info.RFPPrunes, info.MoveLoopNodes)
			}
			if c.inCheck {
				if rootEvals != 0 {
					t.Fatal("evaluated in-check root")
				}
			} else if rootEvals != 1 || info.StaticEvalStack[0] != static {
				t.Fatalf("raw eval calls=%d refined=%d want=%d", rootEvals, info.StaticEvalStack[0], static)
			}
			t.Logf("raw=%d corrected=%d refined=%d beta=%d score=%d RFP=%d loops=%d", c.raw, corrected, static, beta, got, info.RFPPrunes, info.MoveLoopNodes)
		})
	}
}

func TestWave2CorrMarginDefaultOption(t *testing.T) {
	if W2CorrMargin != acceptedOptionDefaults["W2CorrMargin"] {
		t.Fatal("must use the accepted default")
	}
	for _, p := range TunableSearchParams {
		if p.Name == "W2CorrMargin" {
			if p.Ptr != &W2CorrMargin || p.Def != acceptedOptionDefaults["W2CorrMargin"] || p.Min != 0 || p.Max != 1 {
				t.Fatalf("bad registry entry: %+v", p)
			}
			return
		}
	}
	t.Fatal("missing UCI option")
}
