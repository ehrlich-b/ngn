//go:build amd64 && amd64.v3

package ngnk4

const k4OutputDotUsesAVX2 = true

var (
	_ [HiddenSize - 768]byte
	_ [768 - HiddenSize]byte
)

func k4OutputDot(
	stm, nonSTM *[HiddenSize]int32,
	stmWeights, nonSTMWeights *[HiddenSize]int16,
) int32 {
	return k4OutputDotAVX2(stm, nonSTM, stmWeights, nonSTMWeights)
}

//go:noescape
func k4OutputDotAVX2(
	stm, nonSTM *[HiddenSize]int32,
	stmWeights, nonSTMWeights *[HiddenSize]int16,
) int32
