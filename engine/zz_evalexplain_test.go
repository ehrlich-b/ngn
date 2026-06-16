package engine

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestZZEvalExplain (throwaway diagnostic, lever #1): decomposes NGN's static
// white-POV eval into its individual terms, mirroring evalCoreWhite+evalExtrasWhite
// EXACTLY (the term sum is asserted == evaluateUnsafe, so there is no drift).
//
//   ZZ_FEN=<fen>           -> dump one position's per-term contributions.
//   BIAS_PGN=<lossxray pgn> -> walk loss positions (BIAS_SKIP/BIAS_EVERY like the
//                              bias extractor) and report each term's MEAN OWN-SIDE
//                              contribution (white-POV flipped to NGN's side). The
//                              terms that credit NGN most in games it is LOSING are
//                              the over-optimism culprits behind the +113 zzbias gap.
type termVal struct {
	name  string
	white int
}

func evalTermsWhite(board *Bitboard) []termVal {
	var t []termVal
	add := func(n string, v int) { t = append(t, termVal{n, v}) }

	// --- evalCoreWhite ---
	mgScore, egScore, ph := evaluatePeSTO(board)
	phase := ph
	add("pesto", (mgScore*phase+egScore*(totalPhase-phase))/totalPhase)

	whitePawns := board.GetBitboardOf(WhitePawn)
	blackPawns := board.GetBitboardOf(BlackPawn)
	_, _, wPassed, bPassed, wDoubled, bDoubled, wIso, bIso, chainW, chainB :=
		pawnStructureCounts(board, whitePawns, blackPawns)
	add("passed", (wPassed-bPassed)*taperW(passedPawnBonus, passedPawnBonusEG, phase))
	blackNonPawns := board.GetBlackPieces() &^ blackPawns
	whiteNonPawns := board.GetWhitePieces() &^ whitePawns
	wPassers := passersOf(whitePawns, blackPawns, White)
	bPassers := passersOf(blackPawns, whitePawns, Black)
	add("passerBlocked", -(passedBlockedCount(wPassers, blackNonPawns, White)-passedBlockedCount(bPassers, whiteNonPawns, Black))*taperW(blockedPasserPenalty, blockedPasserPenaltyEG, phase))
	wKsq := trailingZeros(board.GetBitboardOf(WhiteKing))
	bKsq := trailingZeros(board.GetBitboardOf(BlackKing))
	add("passerKingDisc", -(passedKingDiscount(wPassers, bKsq, White)-passedKingDiscount(bPassers, wKsq, Black))*taperW(0, kingPasserBlockEG, phase))
	add("passerRookBehind", -(passedRookBehindCount(wPassers, board.GetBitboardOf(BlackRook), White)-passedRookBehindCount(bPassers, board.GetBitboardOf(WhiteRook), Black))*taperW(0, rookBehindPasserEG, phase))
	add("doubled", -(wDoubled-bDoubled)*taperW(doubledPawnPenalty, doubledPawnPenaltyEG, phase))
	add("isolated", -(wIso-bIso)*taperW(isolatedPawnPenalty, isolatedPawnPenaltyEG, phase))
	add("chains", (chainW-chainB)*taperW(pawnChainWeight, pawnChainWeightEG, phase)/100)

	// --- evalExtrasWhite ---
	var sliderAtt [64]uint64
	fillSliderAttacks(board, &sliderAtt)
	mgMob, egMob := evaluateMobilityDelta(board, &sliderAtt)
	add("mobility", taperW(mgMob, egMob, phase))
	add("rookOpen", (evaluateRookOnOpenFile(board, White)-evaluateRookOnOpenFile(board, Black))*taperW(rookOpenWeight, rookOpenWeightEG, phase)/100)
	wN, wB := board.GetBitboardOf(WhiteKnight), board.GetBitboardOf(WhiteBishop)
	bN, bB := board.GetBitboardOf(BlackKnight), board.GetBitboardOf(BlackBishop)
	add("outposts", (evaluateOutpostSquares(wN, wB, whitePawns, blackPawns, White)-evaluateOutpostSquares(bN, bB, blackPawns, whitePawns, Black))*taperW(outpostWeight, outpostWeightEG, phase)/100)
	if phase >= 6 {
		add("kingSafety", (evaluateKingSafety(board, White, &sliderAtt)-evaluateKingSafety(board, Black, &sliderAtt))*taperW(kingSafetyWeight, kingSafetyWeightEG, phase)/100)
	} else {
		add("kingActivity", evaluateKingActivity(board)*taperW(kingActivityWeight, kingActivityWeightEG, phase)/100)
	}
	add("threats", evaluateThreatsDelta(board, &sliderAtt, phase))
	return t
}

func TestZZEvalExplain(t *testing.T) {
	pawnCacheDisabled = true
	defer func() { pawnCacheDisabled = false }()

	if fen := os.Getenv("ZZ_FEN"); fen != "" {
		p, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		terms := evalTermsWhite(&p.Board)
		sum := 0
		fmt.Printf("FEN %s\n", fen)
		for _, tv := range terms {
			if tv.white != 0 {
				fmt.Printf("  %-16s %+6d\n", tv.name, tv.white)
			}
			sum += tv.white
		}
		want := evaluateUnsafe(&p.Board)
		fmt.Printf("  %-16s %+6d   (evaluateUnsafe=%+d)\n", "SUM", sum, want)
		if sum != want {
			t.Fatalf("term sum %d != evaluateUnsafe %d — mirror drift", sum, want)
		}
		return
	}

	in := os.Getenv("BIAS_PGN")
	if in == "" {
		t.Skip("set ZZ_FEN or BIAS_PGN")
	}
	every, skip := 4, 12
	fmt.Sscanf(os.Getenv("BIAS_EVERY"), "%d", &every)
	fmt.Sscanf(os.Getenv("BIAS_SKIP"), "%d", &skip)
	if every < 1 {
		every = 4
	}

	f, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sums := map[string]int{}
	var order []string
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<22)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(ln, "GAME ") {
			continue
		}
		bar := strings.Index(ln, " | ")
		if bar < 0 {
			continue
		}
		head := strings.Fields(ln[:bar])
		if len(head) < 5 || head[2] != "L" || strings.Contains(ln, "time-forfeit") {
			continue
		}
		ngnWhite := head[3] == "w"
		moves := strings.Fields(ln[bar+3:])
		pos := &Position{Board: StartingBoard(), Tag: WhiteCanCastleKingSide | WhiteCanCastleQueenSide | BlackCanCastleKingSide | BlackCanCastleQueenSide | WhiteToMove, EnPassant: NoSquare}
		for ply, mv := range moves {
			pm, err := ParseUCIMove(pos, mv)
			if err != nil {
				break
			}
			pos.MakeMove(pm)
			if ply < skip || ply%every != 0 {
				continue
			}
			terms := evalTermsWhite(&pos.Board)
			sum := 0
			for _, tv := range terms {
				own := tv.white
				if !ngnWhite {
					own = -own
				}
				if _, ok := sums[tv.name]; !ok {
					order = append(order, tv.name)
				}
				sums[tv.name] += own
				sum += tv.white
			}
			if sum != evaluateUnsafe(&pos.Board) {
				t.Fatalf("term sum %d != evaluateUnsafe %d at ply %d — mirror drift", sum, evaluateUnsafe(&pos.Board), ply)
			}
			n++
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	type row struct {
		name string
		mean float64
	}
	var rows []row
	for _, k := range order {
		rows = append(rows, row{k, float64(sums[k]) / float64(n)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].mean > rows[j].mean })
	fmt.Printf("\n=== MEAN OWN-SIDE term contribution over %d loss positions (+ = credits NGN, the losing side) ===\n", n)
	for _, r := range rows {
		fmt.Printf("  %-16s %+8.1f\n", r.name, r.mean)
	}
}
