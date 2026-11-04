package engine

import "testing"

func directQuietPromotions(pos *Position) map[Move]bool {
	var got [32]Move
	n := appendQuietPromotions(pos, got[:])
	out := make(map[Move]bool, n)
	for _, move := range got[:n] {
		out[move] = true
	}
	return out
}

func fullQuietPromotions(pos *Position, legal bool) map[Move]bool {
	var all [256]Move
	n := GenerateMovesIntoBuffer(pos, all[:])
	out := map[Move]bool{}
	for _, move := range all[:n] {
		if !move.IsCapture() && !move.IsEnPassant() && move.PromoType() != NoType && (!legal || IsLegalMove(pos, move)) {
			out[move] = true
		}
	}
	return out
}

func equalMoveSets(a, b map[Move]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for move := range a {
		if !b[move] {
			return false
		}
	}
	return true
}

func actualNoisyMoves(pos *Position, legal bool) map[Move]bool {
	var captures [64]Move
	n := GenerateCapturesIntoBuffer(pos, captures[:])
	out := map[Move]bool{}
	for _, move := range captures[:n] {
		if !legal || IsLegalMove(pos, move) {
			out[move] = true
		}
	}
	for move := range directQuietPromotions(pos) {
		if !legal || IsLegalMove(pos, move) {
			out[move] = true
		}
	}
	return out
}

func fullNoisyMoves(pos *Position, legal bool) map[Move]bool {
	var all [256]Move
	n := GenerateMovesIntoBuffer(pos, all[:])
	out := map[Move]bool{}
	for _, move := range all[:n] {
		noisy := move.IsCapture() || (!move.IsEnPassant() && move.PromoType() != NoType)
		if noisy && (!legal || IsLegalMove(pos, move)) {
			out[move] = true
		}
	}
	return out
}

func assertQuietPromotionClass(t *testing.T, pos *Position) {
	t.Helper()
	before := pos.Copy()
	direct := directQuietPromotions(pos)
	if !PositionEqual(before, pos) {
		t.Fatal("direct generator changed position")
	}
	fullPseudo := fullQuietPromotions(pos, false)
	if !equalMoveSets(direct, fullPseudo) {
		t.Fatalf("pseudo quiet promotions direct=%v full=%v", direct, fullPseudo)
	}
	directLegal := map[Move]bool{}
	for move := range direct {
		if IsLegalMove(pos, move) {
			directLegal[move] = true
		}
	}
	fullLegal := fullQuietPromotions(pos, true)
	if !equalMoveSets(directLegal, fullLegal) {
		t.Fatalf("legal quiet promotions direct=%v full=%v", directLegal, fullLegal)
	}
	if !equalMoveSets(actualNoisyMoves(pos, false), fullNoisyMoves(pos, false)) {
		t.Fatalf("pseudo noisy class differs")
	}
	if !equalMoveSets(actualNoisyMoves(pos, true), fullNoisyMoves(pos, true)) {
		t.Fatalf("legal noisy class differs")
	}
	if !PositionEqual(before, pos) {
		t.Fatal("legality checks changed position")
	}
}

func TestQuietPromotionGeneratorMatchesFullMoveClass(t *testing.T) {
	cases := []string{
		"7k/5P2/6K1/8/8/8/8/8 w - - 0 1",     // white quiet mate promotions
		"8/8/8/8/8/6k1/5p2/7K b - - 0 1",     // black mirror
		"r5k1/1P6/8/8/8/8/8/6K1 w - - 0 1",   // b8 quiet and bxa8 capture promotions
		"6k1/8/8/8/8/8/1p6/R5K1 b - - 0 1",   // black quiet and bxa1 capture promotions
		"1b5k/2P5/3K4/8/8/8/8/8 w - - 0 1",   // diagonal-pinned white c7 pawn
		"8/8/8/8/8/3k4/2p5/1B5K b - - 0 1",   // diagonal-pinned black c2 pawn mirror
		"4r1k1/4P3/8/8/8/8/8/4K3 w - - 0 1",  // blocked white pawn
		"4k3/8/8/8/8/8/4p3/4R1K1 b - - 0 1",  // blocked black pawn
		"8/PPPPPPPP/7k/8/8/8/8/K7 w - - 0 1", // maximum eight quiet-promotion pawns
	}
	for _, fen := range cases {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		assertQuietPromotionClass(t, pos)
		if fen == "1b5k/2P5/3K4/8/8/8/8/8 w - - 0 1" || fen == "8/8/8/8/8/3k4/2p5/1B5K b - - 0 1" {
			if got := len(directQuietPromotions(pos)); got != 4 {
				t.Fatalf("diagonal pin pseudo promotions=%d, want 4", got)
			}
			if got := len(fullQuietPromotions(pos, true)); got != 0 {
				t.Fatalf("diagonal pin legal promotions=%d, want 0", got)
			}
		}
		if fen == "8/PPPPPPPP/7k/8/8/8/8/K7 w - - 0 1" && len(directQuietPromotions(pos)) != 32 {
			t.Fatalf("maximum quiet promotions=%d, want 32", len(directQuietPromotions(pos)))
		}
	}
}

func TestQuietPromotionGeneratorDeterministicLegalWalk(t *testing.T) {
	pos, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	for ply := 0; ply < 160; ply++ {
		assertQuietPromotionClass(t, pos)
		var all [256]Move
		n := GenerateMovesIntoBuffer(pos, all[:])
		legal := make([]Move, 0, n)
		for _, move := range all[:n] {
			if IsLegalMove(pos, move) {
				legal = append(legal, move)
			}
		}
		if len(legal) == 0 {
			t.Fatalf("walk ended at ply %d", ply)
		}
		move := legal[(ply*17+3)%len(legal)]
		before := pos.Copy()
		ep, tag, hc, _ := pos.MakeMove(move)
		pos.UnMakeMove(move, tag, ep, hc)
		if !PositionEqual(before, pos) {
			t.Fatalf("make/unmake changed walk state at ply %d", ply)
		}
		ep, tag, hc, _ = pos.MakeMove(move)
		_ = ep
		_ = tag
		_ = hc
	}
}

func TestQSearchQuietPromotionMates(t *testing.T) {
	for _, fen := range []string{
		"7k/5P2/6K1/8/8/8/8/8 w - - 0 1",
		"8/8/8/8/8/6k1/5p2/7K b - - 0 1",
	} {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		before := pos.Copy()
		ClearStop()
		ClearHistoryTable()
		defaultSearchEngine, _ = NewSearchEngineWithHash(DEFAULT_CACHE_SIZE)
		info := newQSearchInfo()
		info.maxNodes = 100 // Bound the historical horizon witness.
		if got := quiescence(pos, -INFINITY, INFINITY, 0, info); got != MATE_VALUE-1 {
			t.Fatalf("qsearch(%s)=%d, want exact mate score %d", fen, got, MATE_VALUE-1)
		}
		if !PositionEqual(before, pos) || info.FrameDepth != 0 || info.RepStackLen != 0 {
			t.Fatalf("qsearch did not restore state: pos=%t frames=%d reps=%d", PositionEqual(before, pos), info.FrameDepth, info.RepStackLen)
		}
	}
}

func TestQSearchQuietPromotionEvaluatorTransitions(t *testing.T) {
	for _, tc := range []struct {
		fen      string
		pawn     Piece
		from, to Square
	}{
		{"7k/5P2/6K1/8/8/8/8/8 w - - 0 1", WhitePawn, F7, F8},
		{"8/8/8/8/8/6k1/5p2/7K b - - 0 1", BlackPawn, F2, F1},
	} {
		pos, err := ParseFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}
		before := pos.Copy()
		searcher := NewSearchEngine()
		evaluator := ngnn1NewWorker(t, ngnn3TestNetwork(t, 32, true), pos)
		ngnn3Assert(t, evaluator, pos)
		info := newN3CDirectSearchInfo(searcher, evaluator, NewCache(1), 0)
		pushes := 0
		evaluator.transitionObserver = func(p *Position, e *workerEvaluator) {
			ngnn3Assert(t, e, p)
			if p.Board.PieceOccupancy(tc.pawn)&SquareMask[tc.from] == 0 && p.Board.PieceAt(tc.to) != NoPiece {
				pushes++
			}
		}
		if got := quiescence(pos, -INFINITY, INFINITY, 0, info); got != MATE_VALUE-1 {
			t.Errorf("qsearch(%s)=%d", tc.fen, got)
		}
		if pushes == 0 || !PositionEqual(before, pos) || info.FrameDepth != 0 || info.RepStackLen != 0 || evaluator.evaluationDepth() != 0 {
			t.Fatalf("promotion pushes=%d, position restored=%t, frames=%d reps=%d evaluator=%d", pushes, PositionEqual(before, pos), info.FrameDepth, info.RepStackLen, evaluator.evaluationDepth())
		}
		ngnn3Assert(t, evaluator, pos)
	}
}
