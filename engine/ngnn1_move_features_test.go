package engine

import "testing"

func TestNGNN1MoveFeaturePairs(t *testing.T) {
	// Cover every int8 piece value, including compatibility low nibbles;
	// squares are the six-bit fields of a packed move.
	for value := -128; value < 128; value++ {
		piece := Piece(value)
		for square := A1; square <= H8; square++ {
			white, black := ngnn1MoveFeatures(piece, square)
			if white != uint16(ngnn1Feature(piece, square, White)) || black != uint16(ngnn1Feature(piece, square, Black)) {
				t.Fatalf("piece=%d square=%d feature pair differs", piece, square)
			}
		}
	}
}
