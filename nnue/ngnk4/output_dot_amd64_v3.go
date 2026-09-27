//go:build amd64 && amd64.v3

package ngnk4

const k4OutputDotUsesAVX2 = true

// These pairs fail compilation if the fixed K4 hidden width changes without
// a corresponding kernel review.
var (
	_ [HiddenSize - 768]byte
	_ [768 - HiddenSize]byte
)

func k4OutputDot(
	stm, nonSTM, stmWeights, nonSTMWeights *[HiddenSize]int16,
) int32 {
	return k4OutputDotAVX2(stm, nonSTM, stmWeights, nonSTMWeights)
}

//go:noescape
func k4OutputDotAVX2(
	stm, nonSTM, stmWeights, nonSTMWeights *[HiddenSize]int16,
) int32
