package engine

import (
	"math"
	"testing"
)

// texelGateFENs span every eval term and the full phase range (0..24): startpos,
// a castled middlegame (king safety), rook/pawn and K+P endgames (king activity),
// doubled/isolated/passed pawns, an open-file rook, a knight outpost, and a
// reduced-material position at phase 9 to exercise the lowered king-safety gate.
var texelGateFENs = []string{
	"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	"r1bq1rk1/pp2bppp/2n1pn2/2pp4/3P4/2N1PN2/PPP1BPPP/R1BQ1RK1 w - - 0 8",
	"8/5pk1/6p1/8/8/1R6/5PK1/8 w - - 0 1",
	"8/8/8/4k3/8/4P3/4K3/8 w - - 0 1",
	"4k3/8/8/8/8/2P1P3/2P5/4K3 w - - 0 1",
	"4k3/8/8/3P4/8/8/8/4K3 w - - 0 1",
	"4k3/8/8/8/8/8/4P3/R3K3 w - - 0 1",
	"4k3/1p6/8/2Np4/8/8/8/4K3 w - - 0 1",
	"1q2k3/8/8/8/8/8/8/1Q2K2N w - - 0 1",
	"r2q1rk1/1b1nbppp/p2ppn2/1p6/3NPP2/2N1B3/PPPQB1PP/R4RK1 w - - 0 12",
}

// TestEvalTraceReproducesEval is the Phase (a) gate: the cached trace must
// reproduce evaluateUnsafe bit-for-bit (and its PeSTO occupancy must reproduce the
// accumulator), which makes texelMSE-from-traces identical to texelMSE. The tuner
// always runs with the pawn cache disabled, so we match that regime here.
func TestEvalTraceReproducesEval(t *testing.T) {
	saved := pawnCacheDisabled
	pawnCacheDisabled = true
	defer func() { pawnCacheDisabled = saved }()

	for _, fen := range texelGateFENs {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q): %v", fen, err)
		}
		b := pos.Board
		b.recomputeAccumulator()

		want := evaluateUnsafe(&b)
		tr := buildEvalTrace(&b)
		got := reconstructEvalInt(&tr)
		if got != want {
			t.Errorf("reconstructEvalInt=%d, evaluateUnsafe=%d (phase %d) for %q", got, want, tr.Phase, fen)
		}

		// The per-piece occupancy must rebuild the accumulator exactly (guards the
		// white sq^56 / black sq flip convention independently of term cancellation).
		texelRawTables()
		mg, eg := 0, 0
		for _, pc := range tr.Pieces {
			pt := PieceType(pc.Pt)
			s := int(pc.Sign)
			mg += s * (pestoMGMaterial[pt] + texelRawMG[pt][pc.Cell])
			eg += s * (pestoEGMaterial[pt] + texelRawEG[pt][pc.Cell])
		}
		if mg != b.accMG || eg != b.accEG {
			t.Errorf("decomposed (mg=%d,eg=%d) != accumulator (mg=%d,eg=%d) for %q", mg, eg, b.accMG, b.accEG, fen)
		}
	}
}

// TestEvalTraceFloatFidelity confirms the float linear model (used for gradients)
// matches the integer eval to within the expected truncation residual: one /24
// taper floor plus six /100 term floors, < ~8cp. A large gap would mean the float
// model and the engine eval have structurally diverged.
func TestEvalTraceFloatFidelity(t *testing.T) {
	saved := pawnCacheDisabled
	pawnCacheDisabled = true
	defer func() { pawnCacheDisabled = saved }()

	ptrs := TexelFullParams()
	if len(ptrs) != texelNumParams {
		t.Fatalf("TexelFullParams len=%d, want %d", len(ptrs), texelNumParams)
	}
	w := make([]float64, len(ptrs))
	for i, p := range ptrs {
		w[i] = float64(*p)
	}

	maxResidual := 0.0
	for _, fen := range texelGateFENs {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q): %v", fen, err)
		}
		b := pos.Board
		b.recomputeAccumulator()
		tr := buildEvalTrace(&b)
		residual := math.Abs(evalTraceFloat(&tr, w) - float64(reconstructEvalInt(&tr)))
		if residual > maxResidual {
			maxResidual = residual
		}
	}
	if maxResidual > 8.0 {
		t.Errorf("float/int eval residual %.3f exceeds expected truncation bound (~8cp)", maxResidual)
	}
	t.Logf("max float/int residual: %.3fcp", maxResidual)
}
