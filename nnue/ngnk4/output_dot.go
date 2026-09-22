package ngnk4

func clippedSquaredWeightedFast(value int32, weight int16) int32 {
	if value < 0 {
		value = 0
	} else if value > InputScale {
		value = InputScale
	}
	return value * value * int32(weight)
}

// k4OutputDotPortable is the arithmetic oracle for the optional AVX2 kernel.
// It preserves the format's signed int32 sum for fast-safe output heads.
func k4OutputDotPortable(
	stm, nonSTM *[HiddenSize]int32,
	stmWeights, nonSTMWeights *[HiddenSize]int16,
) int32 {
	var sum int32
	for hidden := 0; hidden < HiddenSize; hidden += 4 {
		sum += clippedSquaredWeightedFast(stm[hidden], stmWeights[hidden])
		sum += clippedSquaredWeightedFast(nonSTM[hidden], nonSTMWeights[hidden])
		sum += clippedSquaredWeightedFast(stm[hidden+1], stmWeights[hidden+1])
		sum += clippedSquaredWeightedFast(nonSTM[hidden+1], nonSTMWeights[hidden+1])
		sum += clippedSquaredWeightedFast(stm[hidden+2], stmWeights[hidden+2])
		sum += clippedSquaredWeightedFast(nonSTM[hidden+2], nonSTMWeights[hidden+2])
		sum += clippedSquaredWeightedFast(stm[hidden+3], stmWeights[hidden+3])
		sum += clippedSquaredWeightedFast(nonSTM[hidden+3], nonSTMWeights[hidden+3])
	}
	return sum
}
