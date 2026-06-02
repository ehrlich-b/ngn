package engine

// Gradient Texel tuning. The static eval is LINEAR in its tunable parameters for
// a fixed position: the PeSTO core is Σ_pieces sign·(material[pt] + rawPST[pt][sq])
// tapered by phase, and every other term is feature·weight (optionally /100). So
// for each position we can emit a sparse feature-coefficient vector once — the
// trace — and thereafter evaluate the eval (and its exact gradient w.r.t. every
// weight) as cheap linear algebra over the cached trace, with no re-evaluation.
// That caching is what makes gradient descent 100-1000x cheaper than coordinate
// descent's per-parameter line search.
//
// Parameter layout (the global index order, matching cmd/texel's "full" target —
// TexelMaterialParams ++ TexelPSTParams ++ TexelWeights):
//
//	[0..4]    pestoMGMaterial[Pawn..Queen]
//	[5..9]    pestoEGMaterial[Pawn..Queen]
//	[10..777] 768 raw PST cells: 12 tables (mgPawn..mgKing, egPawn..egKing) × 64
//	[778..786] the 9 hand-set scalar weights (TexelWeights order)
//
// King has no material parameter (its value is fixed at 0, conventional since both
// sides always have exactly one), but it does have a PST.

const (
	texelNumMaterial = 10  // mg Pawn..Queen (0..4), eg Pawn..Queen (5..9)
	texelNumPST      = 768 // 12 tables × 64
	texelNumScalars  = 9   // == len(TexelWeights())
	texelPSTBase     = texelNumMaterial
	texelScalarBase  = texelNumMaterial + texelNumPST
	texelNumParams   = texelNumMaterial + texelNumPST + texelNumScalars // 787
)

// Global-index helpers. King (pt==6) is out of material range — callers must skip
// material for the king; its PST indices are valid.
func texelMatMGIndex(pt PieceType) int          { return int(pt) - 1 }     // Pawn..Queen -> 0..4
func texelMatEGIndex(pt PieceType) int          { return 5 + int(pt) - 1 } // -> 5..9
func texelPSTMGIndex(pt PieceType, c uint8) int { return texelPSTBase + (int(pt)-1)*64 + int(c) }
func texelPSTEGIndex(pt PieceType, c uint8) int { return texelPSTBase + (6+int(pt)-1)*64 + int(c) }

// pieceTerm is one piece's contribution to the PeSTO core: its type, the raw-table
// cell it indexes (sq^56 for white, sq for black — matching RebuildPST), and its
// sign (+1 white, −1 black, so summing yields the white-perspective score).
type pieceTerm struct {
	Pt   uint8
	Cell uint8
	Sign int8
}

// evalTrace is the cached linear decomposition of one position's eval: the PeSTO
// occupancy (per-piece terms) plus the integer feature values of the non-PeSTO
// terms (exactly as evaluateUnsafe computes them, with the phase gates folded in
// so an inactive term's feature is 0). Phase and the game result ride along.
type evalTrace struct {
	Phase  int
	Result float64
	Pieces []pieceTerm
	// Non-PeSTO term features (white-perspective deltas, integers).
	Passed       int
	Doubled      int
	Isolated     int
	Chain        int
	Mobility     int
	RookOpen     int
	Outpost      int
	KingSafety   int // 0 unless phase >= 7
	KingActivity int // 0 unless phase < 6
}

// texelRawMG/EG cache the (fixed) addresses of the raw white-POV PST tables so the
// reconstruction need not rebuild the slice each call. The CONTENTS change during
// tuning; the pointers do not.
var texelRawMG, texelRawEG [7]*[64]int

func texelRawTables() {
	if texelRawMG[Pawn] == nil {
		texelRawMG = [7]*[64]int{nil, &mgPawnTable, &mgKnightTable, &mgBishopTable, &mgRookTable, &mgQueenTable, &mgKingTable}
		texelRawEG = [7]*[64]int{nil, &egPawnTable, &egKnightTable, &egBishopTable, &egRookTable, &egQueenTable, &egKingTable}
	}
}

// buildEvalTrace decomposes a position's eval into a cached trace. It mirrors
// evaluateUnsafe term-for-term: the per-piece occupancy reproduces the PeSTO
// accumulator (accMG/accEG), and the feature integers are captured from the same
// helper calls evaluateUnsafe makes — so reconstructEvalInt(trace) is bit-exact
// with evaluateUnsafe(board), which the gate test asserts.
func buildEvalTrace(board *Bitboard) evalTrace {
	var t evalTrace
	phase := board.accPhase
	if phase > totalPhase {
		phase = totalPhase
	}
	t.Phase = phase

	// PeSTO core occupancy. White pieces index the raw table at sq^56 (display
	// order) with +1; black at sq with −1 — exactly RebuildPST's convention.
	pieces := make([]pieceTerm, 0, 32)
	for p := WhitePawn; p <= WhiteKing; p++ {
		pt := PieceType(p) // WhitePawn=1=Pawn .. WhiteKing=6=King
		for bb := board.GetBitboardOf(p); bb != 0; bb &= bb - 1 {
			sq := bitScanForward(bb)
			pieces = append(pieces, pieceTerm{uint8(pt), sq ^ 56, 1})
		}
	}
	for p := BlackPawn; p <= BlackKing; p++ {
		pt := PieceType(p - 6) // BlackPawn=7 -> Pawn ..
		for bb := board.GetBitboardOf(p); bb != 0; bb &= bb - 1 {
			sq := bitScanForward(bb)
			pieces = append(pieces, pieceTerm{uint8(pt), sq, -1})
		}
	}
	t.Pieces = pieces

	// Non-PeSTO terms — identical calls/order to evaluateUnsafe.
	whitePawns := board.GetBitboardOf(WhitePawn)
	blackPawns := board.GetBitboardOf(BlackPawn)
	wPassed, bPassed, wDoubled, bDoubled, wIso, bIso, chainW, chainB := pawnStructureCounts(board, whitePawns, blackPawns)
	t.Passed = wPassed - bPassed
	t.Doubled = wDoubled - bDoubled
	t.Isolated = wIso - bIso
	t.Chain = chainW - chainB
	t.Mobility = evaluateMobilityDelta(board)
	t.RookOpen = evaluateRookOnOpenFile(board, White) - evaluateRookOnOpenFile(board, Black)

	whiteKnights := board.GetBitboardOf(WhiteKnight)
	whiteBishops := board.GetBitboardOf(WhiteBishop)
	blackKnights := board.GetBitboardOf(BlackKnight)
	blackBishops := board.GetBitboardOf(BlackBishop)
	t.Outpost = evaluateOutpostSquares(whiteKnights, whiteBishops, whitePawns, blackPawns, White) -
		evaluateOutpostSquares(blackKnights, blackBishops, blackPawns, whitePawns, Black)

	if phase >= 7 {
		t.KingSafety = evaluateKingSafety(board, White) - evaluateKingSafety(board, Black)
	} else if phase < 6 {
		t.KingActivity = evaluateKingActivity(board)
	}
	return t
}

// reconstructEvalInt rebuilds the integer eval from a trace using the SAME integer
// arithmetic as evaluateUnsafe (one taper division, per-term /100 truncation). It
// reads the current parameter values from the package globals, so at trace time it
// is bit-exact with evaluateUnsafe — the gate that proves the trace is faithful.
func reconstructEvalInt(t *evalTrace) int {
	texelRawTables()
	mg, eg := 0, 0
	for _, pc := range t.Pieces {
		pt := PieceType(pc.Pt)
		s := int(pc.Sign)
		mg += s * (pestoMGMaterial[pt] + texelRawMG[pt][pc.Cell])
		eg += s * (pestoEGMaterial[pt] + texelRawEG[pt][pc.Cell])
	}
	eval := (mg*t.Phase + eg*(totalPhase-t.Phase)) / totalPhase
	eval += t.Passed * passedPawnBonus
	eval -= t.Doubled * doubledPawnPenalty
	eval -= t.Isolated * isolatedPawnPenalty
	eval += t.Chain * pawnChainWeight / 100
	eval += t.Mobility * mobilityWeight / 100
	eval += t.RookOpen * rookOpenWeight / 100
	eval += t.Outpost * outpostWeight / 100
	eval += t.KingSafety * kingSafetyWeight / 100
	eval += t.KingActivity * kingActivityWeight / 100
	return eval
}

// evalTraceFloat is the linear eval model used for gradients: s = Σ wⱼ·cⱼ over the
// float weight vector w (length texelNumParams). It is the float relaxation of
// reconstructEvalInt — the only difference is that the /24 taper and the six /100
// term scalings are exact here rather than integer-truncated, a sub-centipawn
// residual the gradient ignores. PST and material share each piece's occupancy
// coefficient (the eval sums material+table into one PST lookup).
func evalTraceFloat(t *evalTrace, w []float64) float64 {
	mgF := float64(t.Phase) / float64(totalPhase)
	egF := float64(totalPhase-t.Phase) / float64(totalPhase)
	var s float64
	for _, pc := range t.Pieces {
		pt := PieceType(pc.Pt)
		sg := float64(pc.Sign)
		s += sg * mgF * w[texelPSTMGIndex(pt, pc.Cell)]
		s += sg * egF * w[texelPSTEGIndex(pt, pc.Cell)]
		if pt != King {
			s += sg * mgF * w[texelMatMGIndex(pt)]
			s += sg * egF * w[texelMatEGIndex(pt)]
		}
	}
	s += float64(t.Passed) * w[texelScalarBase+0]
	s -= float64(t.Doubled) * w[texelScalarBase+1]
	s -= float64(t.Isolated) * w[texelScalarBase+2]
	s += float64(t.Chain) / 100.0 * w[texelScalarBase+3]
	s += float64(t.Mobility) / 100.0 * w[texelScalarBase+4]
	s += float64(t.RookOpen) / 100.0 * w[texelScalarBase+5]
	s += float64(t.Outpost) / 100.0 * w[texelScalarBase+6]
	s += float64(t.KingSafety) / 100.0 * w[texelScalarBase+7]
	s += float64(t.KingActivity) / 100.0 * w[texelScalarBase+8]
	return s
}

// TexelFullParams returns pointers to all 787 tunable parameters in the global
// index order (material, PST, scalar weights) — the order the trace's index
// helpers assume. The gradient optimizer reads current values from these and
// writes tuned values back (RebuildPST after, since PST/material feed it).
func TexelFullParams() []*int {
	ptrs := make([]*int, 0, texelNumParams)
	ptrs = append(ptrs, TexelMaterialParams()...)
	ptrs = append(ptrs, TexelPSTParams()...)
	for _, w := range TexelWeights() {
		ptrs = append(ptrs, w.Ptr)
	}
	return ptrs
}
