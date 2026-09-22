package ngnk4

import (
	"fmt"
	"math/bits"
)

var kingBucketTable = [64]int{
	1, 1, 0, 0, 0, 0, 1, 1,
	2, 2, 2, 2, 2, 2, 2, 2,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
}

type accumulator [2][HiddenSize]int32

func (model *Model) validate() error {
	if model == nil || !model.loaded || model.metadata.ArchitectureID != ArchitectureID ||
		model.metadata.FileBytes != FileSize || model.metadata.PayloadBytes != PayloadSize {
		return fmt.Errorf("%w: model is not strict-loader validated", ErrFormat)
	}
	return nil
}

func validatePosition(position Position) error {
	if position.SideToMove != White && position.SideToMove != Black {
		return fmt.Errorf("NGN K4 position: invalid side to move %d", position.SideToMove)
	}
	var occupied uint64
	for plane, pieces := range position.Board {
		if occupied&pieces != 0 {
			return fmt.Errorf("NGN K4 position: overlapping piece plane %d", plane)
		}
		occupied |= pieces
	}
	if count := bits.OnesCount64(position.Board[WhiteKing]); count != 1 {
		return fmt.Errorf("NGN K4 position: white king count %d, want 1", count)
	}
	if count := bits.OnesCount64(position.Board[BlackKing]); count != 1 {
		return fmt.Errorf("NGN K4 position: black king count %d, want 1", count)
	}
	if count := bits.OnesCount64(occupied); count > 32 {
		return fmt.Errorf("NGN K4 position: piece count %d exceeds legal-board bound 32", count)
	}
	return nil
}

func featureIndex(color Color, pieceType, square, kingSquare int, perspective Color) int {
	orientedSquare := square
	orientedKing := kingSquare
	if perspective == Black {
		orientedSquare ^= 56
		orientedKing ^= 56
	}
	if kingSquare%8 > 3 {
		orientedSquare ^= 7
	}
	bucket := kingBucketTable[orientedKing]
	return bucket*InputSize + int(color^perspective)*384 + pieceType*64 + orientedSquare
}

func (model *Model) fullRefresh(board Board) accumulator {
	var result accumulator
	for hidden, bias := range model.inputBiases {
		result[White][hidden] = int32(bias)
		result[Black][hidden] = int32(bias)
	}
	whiteKingSquare := bits.TrailingZeros64(board[WhiteKing])
	blackKingSquare := bits.TrailingZeros64(board[BlackKing])
	for plane, pieceBits := range board {
		color := Color(plane / 6)
		pieceType := plane % 6
		for pieceBits != 0 {
			square := bits.TrailingZeros64(pieceBits)
			pieceBits &= pieceBits - 1
			whiteRow := &model.inputWeights[featureIndex(color, pieceType, square, whiteKingSquare, White)]
			blackRow := &model.inputWeights[featureIndex(color, pieceType, square, blackKingSquare, Black)]
			addFeatureRow(&result[White], whiteRow)
			addFeatureRow(&result[Black], blackRow)
		}
	}
	return result
}

// These fixed-width kernels expose independent accumulator lanes to the Go
// compiler. The network width is a format constant and a multiple of eight.
// Keeping the bounds explicit avoids a branch and dependent induction update
// for every one of the 768 scalar lanes on the search hot path.
func addFeatureRowPortable(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	for hidden := 0; hidden < HiddenSize; hidden += 8 {
		destination[hidden] += int32(row[hidden])
		destination[hidden+1] += int32(row[hidden+1])
		destination[hidden+2] += int32(row[hidden+2])
		destination[hidden+3] += int32(row[hidden+3])
		destination[hidden+4] += int32(row[hidden+4])
		destination[hidden+5] += int32(row[hidden+5])
		destination[hidden+6] += int32(row[hidden+6])
		destination[hidden+7] += int32(row[hidden+7])
	}
}

func subtractFeatureRowPortable(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	for hidden := 0; hidden < HiddenSize; hidden += 8 {
		destination[hidden] -= int32(row[hidden])
		destination[hidden+1] -= int32(row[hidden+1])
		destination[hidden+2] -= int32(row[hidden+2])
		destination[hidden+3] -= int32(row[hidden+3])
		destination[hidden+4] -= int32(row[hidden+4])
		destination[hidden+5] -= int32(row[hidden+5])
		destination[hidden+6] -= int32(row[hidden+6])
		destination[hidden+7] -= int32(row[hidden+7])
	}
}

func clippedSquaredWeighted(value int32, weight int16) int64 {
	if value < 0 {
		value = 0
	} else if value > InputScale {
		value = InputScale
	}
	return int64(value) * int64(value) * int64(weight)
}

func outputBucket(board Board) int {
	pieceCount := bits.OnesCount64(occupiedBoard(board))
	bucket := (pieceCount - 2) / 4
	if bucket < 0 {
		return 0
	}
	if bucket >= OutputBuckets {
		return OutputBuckets - 1
	}
	return bucket
}

func (model *Model) evaluateAccumulator(accumulator *accumulator, sideToMove Color, bucket int) int64 {
	stm := &accumulator[sideToMove]
	nonSTM := &accumulator[sideToMove^1]
	stmWeights := &model.outputWeights[bucket][0]
	nonSTMWeights := &model.outputWeights[bucket][1]
	if model.metadata.FastOutputSafe[bucket] {
		sum := k4OutputDot(stm, nonSTM, stmWeights, nonSTMWeights)
		q := int64(sum)/InputScale + int64(model.outputBiases[bucket])
		return q * OutputScale / (InputScale * LayerScale)
	}
	var sum int64
	for hidden := 0; hidden < HiddenSize; hidden += 4 {
		sum += clippedSquaredWeighted(stm[hidden], stmWeights[hidden])
		sum += clippedSquaredWeighted(nonSTM[hidden], nonSTMWeights[hidden])
		sum += clippedSquaredWeighted(stm[hidden+1], stmWeights[hidden+1])
		sum += clippedSquaredWeighted(nonSTM[hidden+1], nonSTMWeights[hidden+1])
		sum += clippedSquaredWeighted(stm[hidden+2], stmWeights[hidden+2])
		sum += clippedSquaredWeighted(nonSTM[hidden+2], nonSTMWeights[hidden+2])
		sum += clippedSquaredWeighted(stm[hidden+3], stmWeights[hidden+3])
		sum += clippedSquaredWeighted(nonSTM[hidden+3], nonSTMWeights[hidden+3])
	}
	q := sum/InputScale + int64(model.outputBiases[bucket])
	return q * OutputScale / (InputScale * LayerScale)
}

// EvaluateRaw returns the side-to-move score under the frozen 400*z integer
// contract. Rule-50 attenuation and mate-band reservation belong to the engine.
func (model *Model) EvaluateRaw(position Position) (int64, error) {
	if err := model.validate(); err != nil {
		return 0, err
	}
	if err := validatePosition(position); err != nil {
		return 0, err
	}
	accumulator := model.fullRefresh(position.Board)
	return model.evaluateAccumulator(&accumulator, position.SideToMove, outputBucket(position.Board)), nil
}
