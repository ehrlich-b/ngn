package engine

import (
	"os"
	"testing"

	"github.com/ehrlich-b/ngn/countereval"
)

func openCounterFixture(t *testing.T, outputBias float32) *os.File {
	t.Helper()
	path := writeUCICounterFixture(t, "counter-fixture.nn", outputBias)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	return file
}

func TestCounterTransitionAdapterSpecialMovesAndDirectBoards(t *testing.T) {
	tests := []struct {
		name, fen, notation        string
		capture, promotion, castle bool
	}{
		{"white normal", "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1", "e2e4", false, false, false},
		{"black normal", "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1", "e7e5", false, false, false},
		{"white capture", "4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1", "e4d5", true, false, false},
		{"black en passant", "4k3/8/8/8/3pP3/8/8/4K3 b - e3 0 1", "d4e3", true, false, false},
		{"white castle", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1g1", false, false, true},
		{"black castle", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8c8", false, false, true},
		{"white promotion", "4k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a7a8n", false, true, false},
		{"black promotion capture", "4k3/8/8/8/8/8/p7/1R2K3 b - - 0 1", "a2b1q", true, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pos, err := ParseFEN(test.fen)
			if err != nil {
				t.Fatal(err)
			}
			beforePosition := snapshotPVPosition(pos)
			beforeBoard, err := counterBoardFromPosition(pos)
			if err != nil {
				t.Fatal(err)
			}
			move := adapterLegalMove(t, pos, test.notation)
			delta, err := counterMoveDelta(pos, move)
			if err != nil {
				t.Fatal(err)
			}
			assertPVPositionRestored(t, pos, beforePosition)
			if delta.HasCapture != test.capture || delta.HasPromotion != test.promotion || delta.HasCastleRook != test.castle {
				t.Fatalf("flags=%+v", delta)
			}
			undoEP, undoTag, undoClock, legal := pos.MakeMove(move)
			if !legal {
				t.Fatal("generated move rejected")
			}
			expectedPost, err := validateCounterMoveAfter(pos, delta)
			if err != nil {
				t.Fatal(err)
			}
			model, _, loadErr := countereval.LoadCounter55Legacy(openCounterFixture(t, 0))
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			context, err := model.NewSearchContext(beforeBoard)
			if err != nil {
				t.Fatal(err)
			}
			if err := context.PushMove(delta, expectedPost); err != nil {
				t.Fatal(err)
			}
			if context.Board() != expectedPost {
				t.Fatal("Counter context board differs from direct post-position bitboards")
			}
			if err := context.Pop(); err != nil {
				t.Fatal(err)
			}
			if context.Board() != beforeBoard {
				t.Fatal("Counter pop did not restore direct pre-position bitboards")
			}
			pos.UnMakeMove(move, undoTag, undoEP, undoClock)
			assertPVPositionRestored(t, pos, beforePosition)
		})
	}
}

func TestCounterTransitionAdapterRejectsInvalidState(t *testing.T) {
	if _, err := counterBoardFromPosition(nil); err == nil {
		t.Fatal("nil position accepted")
	}
	pos, err := ParseFEN("4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := counterMoveDelta(pos, EmptyMove); err == nil {
		t.Fatal("empty move accepted")
	}
	delta := countereval.MoveDelta{MovingPlane: 0, From: 12, To: 28}
	if _, err := validateCounterMoveAfter(pos, delta); err == nil {
		t.Fatal("unchanged side accepted as post-move position")
	}
}
