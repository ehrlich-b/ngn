package sf18big

import (
	"errors"
	"slices"
	"testing"

	base "github.com/ehrlich-b/ngn/nnue"
)

func TestActiveThreatsMatchesPinnedStockfishStartPosition(t *testing.T) {
	position := bigReferenceStartPosition()
	want := []uint32{
		506, 525, 4550, 4551, 4571, 4572, 10143, 10241, 11032, 11136,
		19187, 19188, 19189, 22096, 26463, 36583, 36584, 36585, 37421,
		42762, 42781, 47787, 47788, 47808, 47809, 55334, 55432, 56231,
		56335, 69143, 69144, 69145, 72062, 76429, 78573, 78574, 78575,
		79416,
	}
	for perspective := base.White; perspective <= base.Black; perspective++ {
		got, err := ActiveThreats(position, perspective)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got.Indices[:got.Count], want) {
			t.Fatalf("perspective %d active threats = %v, want %v", perspective, got.Indices[:got.Count], want)
		}
	}
}

func TestDiffThreatsMatchesPinnedB2B3Transition(t *testing.T) {
	before := bigReferenceStartPosition()
	after := bigReferenceStartPosition()
	after.SideToMove = base.Black
	for index := range after.Pieces {
		piece := &after.Pieces[index]
		if piece.Color == base.White && piece.Piece == base.Pawn && piece.Square == 9 {
			piece.Square = 17
			break
		}
	}
	tests := []struct {
		perspective base.Color
		removed     []uint32
		added       []uint32
	}{
		{base.White, []uint32{4572}, []uint32{10, 13}},
		{base.Black, []uint32{47809}, []uint32{40260, 40263}},
	}
	for _, test := range tests {
		diff, err := DiffThreats(before, after, test.perspective)
		if err != nil {
			t.Fatal(err)
		}
		if diff.RequiresRefresh ||
			!slices.Equal(diff.Removed.Indices[:diff.Removed.Count], test.removed) ||
			!slices.Equal(diff.Added.Indices[:diff.Added.Count], test.added) {
			t.Fatalf("perspective %d diff = %+v, want removed %v added %v", test.perspective, diff, test.removed, test.added)
		}
	}
}

func TestDiffThreatsKingFileHalfRefresh(t *testing.T) {
	before := base.Position{Pieces: []base.PieceOnSquare{
		{Piece: base.King, Color: base.White, Square: 3},
		{Piece: base.King, Color: base.Black, Square: 60},
	}}
	after := base.Position{SideToMove: base.Black, Pieces: []base.PieceOnSquare{
		{Piece: base.King, Color: base.White, Square: 4},
		{Piece: base.King, Color: base.Black, Square: 60},
	}}
	white, err := DiffThreats(before, after, base.White)
	if err != nil {
		t.Fatal(err)
	}
	black, err := DiffThreats(before, after, base.Black)
	if err != nil {
		t.Fatal(err)
	}
	if !white.RequiresRefresh || black.RequiresRefresh {
		t.Fatalf("refresh decisions white=%v black=%v", white.RequiresRefresh, black.RequiresRefresh)
	}
}

func TestBigReferenceRejectsInvalidInputs(t *testing.T) {
	if _, err := ActiveThreats(base.Position{}, base.White); !errors.Is(err, ErrEvaluation) {
		t.Fatalf("empty position error = %v", err)
	}
	if got, err := (*Model)(nil).EvaluateSelected(bigReferenceStartPosition()); got != (SelectedEvaluation{}) || !errors.Is(err, ErrEvaluation) {
		t.Fatalf("nil model result = %+v, error %v", got, err)
	}
}

func bigReferenceStartPosition() base.Position {
	position := base.Position{SideToMove: base.White, Pieces: make([]base.PieceOnSquare, 0, 32)}
	backRank := [...]base.PieceType{
		base.Rook, base.Knight, base.Bishop, base.Queen,
		base.King, base.Bishop, base.Knight, base.Rook,
	}
	for file, piece := range backRank {
		position.Pieces = append(position.Pieces,
			base.PieceOnSquare{Piece: piece, Color: base.White, Square: base.Square(file)},
			base.PieceOnSquare{Piece: base.Pawn, Color: base.White, Square: base.Square(8 + file)},
			base.PieceOnSquare{Piece: base.Pawn, Color: base.Black, Square: base.Square(48 + file)},
			base.PieceOnSquare{Piece: piece, Color: base.Black, Square: base.Square(56 + file)},
		)
	}
	return position
}
