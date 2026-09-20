package engine

import "testing"

func collectMainPickerMoves(t *testing.T, fen string, ttMove Move, ttHit bool) ([]Move, []Move) {
	t.Helper()
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	var generatedBuffer [256]Move
	generatedCount := GenerateMovesIntoBuffer(pos, generatedBuffer[:])
	generated := append([]Move(nil), generatedBuffer[:generatedCount]...)

	var frame searchFrame
	history := new(workerHistory)
	picker := newMainMovePicker(history, pos, &frame, nil, 4, EmptyMove, EmptyMove, ttMove, ttHit)
	got := make([]Move, 0, generatedCount)
	for {
		move, ok := picker.next()
		if !ok {
			break
		}
		got = append(got, move)
		if len(got) > len(generated)+1 {
			t.Fatal("picker emitted more moves than full generation")
		}
	}
	return generated, got
}

func requireSameMoveMultiset(t *testing.T, want, got []Move) {
	t.Helper()
	counts := make(map[Move]int, len(want))
	for _, move := range want {
		counts[move]++
	}
	for _, move := range got {
		counts[move]--
	}
	if len(want) != len(got) {
		t.Fatalf("move count changed: generated=%d picker=%d", len(want), len(got))
	}
	for move, count := range counts {
		if count != 0 {
			t.Fatalf("move multiplicity changed for %s: delta=%d", MoveToAlgebraic(move), count)
		}
	}
}

func moveByAlgebraic(t *testing.T, moves []Move, algebraic string) Move {
	t.Helper()
	for _, move := range moves {
		if MoveToAlgebraic(move) == algebraic {
			return move
		}
	}
	t.Fatalf("move %s not found", algebraic)
	return EmptyMove
}

func moveIndex(t *testing.T, moves []Move, target Move) int {
	t.Helper()
	for i, move := range moves {
		if move == target {
			return i
		}
	}
	t.Fatalf("move %s not emitted", MoveToAlgebraic(target))
	return -1
}

func TestMainMovePickerEmitsGeneratedMovesExactlyOnce(t *testing.T) {
	fixtures := []struct {
		name string
		fen  string
	}{
		{"start", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/2Pp4/1p2P3/2N2N2/PPQBBPPP/R3K2R w KQkq - 37 1"},
		{"castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"},
		{"en_passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1"},
		{"promotion", "4k3/P7/8/8/3p4/4Q3/8/4K3 w - - 0 1"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			generated, got := collectMainPickerMoves(t, fixture.fen, EmptyMove, false)
			requireSameMoveMultiset(t, generated, got)

			if len(generated) == 0 {
				t.Fatal("fixture generated no moves")
			}
			generated, got = collectMainPickerMoves(t, fixture.fen, generated[len(generated)/2], true)
			requireSameMoveMultiset(t, generated, got)
		})
	}
}

func TestMainMovePickerTTAndStageOrder(t *testing.T) {
	startFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	generated, _ := collectMainPickerMoves(t, startFEN, EmptyMove, false)
	ttMove := moveByAlgebraic(t, generated, "g1f3")
	_, got := collectMainPickerMoves(t, startFEN, ttMove, true)
	if got[0] != ttMove {
		t.Fatalf("ordinary TT move was not first: got %s", MoveToAlgebraic(got[0]))
	}

	stale := NewMove(A1, A8, WhiteQueen, NoPiece, NoType, 0)
	generated, got = collectMainPickerMoves(t, startFEN, stale, true)
	requireSameMoveMultiset(t, generated, got)
	for _, move := range got {
		if move == stale {
			t.Fatal("stale TT move was emitted")
		}
	}

	promotionFEN := "4k3/P7/8/8/3p4/4Q3/8/4K3 w - - 0 1"
	generated, got = collectMainPickerMoves(t, promotionFEN, EmptyMove, false)
	queenPromotion := moveByAlgebraic(t, generated, "a7a8q")
	goodCapture := moveByAlgebraic(t, generated, "e3d4")
	quiet := moveByAlgebraic(t, generated, "e3e4")
	if !(moveIndex(t, got, queenPromotion) < moveIndex(t, got, goodCapture) &&
		moveIndex(t, got, goodCapture) < moveIndex(t, got, quiet)) {
		t.Fatalf("tactical stage order changed: promotion=%d capture=%d quiet=%d",
			moveIndex(t, got, queenPromotion), moveIndex(t, got, goodCapture), moveIndex(t, got, quiet))
	}

	losingFEN := "3rk3/3p4/8/8/8/8/8/3QK3 w - - 0 1"
	generated, got = collectMainPickerMoves(t, losingFEN, EmptyMove, false)
	losingCapture := moveByAlgebraic(t, generated, "d1d7")
	quiet = moveByAlgebraic(t, generated, "d1a4")
	pos, err := ParseFEN(losingFEN)
	if err != nil {
		t.Fatal(err)
	}
	if score := new(workerHistory).captureOrderScore(pos, losingCapture); score >= 0 {
		t.Fatalf("fixture capture is not losing: score=%d", score)
	}
	if moveIndex(t, got, quiet) >= moveIndex(t, got, losingCapture) {
		t.Fatalf("losing capture was not deferred: quiet=%d capture=%d", moveIndex(t, got, quiet), moveIndex(t, got, losingCapture))
	}
}
