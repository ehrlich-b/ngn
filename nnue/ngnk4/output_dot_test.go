package ngnk4

import (
	"math/rand"
	"testing"
)

func TestK4OutputDotMatchesPortableOracle(t *testing.T) {
	random := rand.New(rand.NewSource(26092001))
	for trial := 0; trial < 128; trial++ {
		var stm, nonSTM [HiddenSize]int32
		var stmWeights, nonSTMWeights [HiddenSize]int16
		for index := 0; index < HiddenSize; index++ {
			switch trial % 4 {
			case 0:
				stm[index] = int32(random.Intn(1024) - 512)
				nonSTM[index] = int32(random.Intn(1024) - 512)
			case 1:
				stm[index] = int32(random.Uint32())
				nonSTM[index] = int32(random.Uint32())
			case 2:
				stm[index] = -1 << 31
				nonSTM[index] = 1<<31 - 1
			case 3:
				stm[index] = int32(index - 256)
				nonSTM[index] = int32(512 - index)
			}
			stmWeights[index] = int16(random.Uint32())
			nonSTMWeights[index] = int16(random.Uint32())
		}
		got := k4OutputDot(&stm, &nonSTM, &stmWeights, &nonSTMWeights)
		want := k4OutputDotPortable(&stm, &nonSTM, &stmWeights, &nonSTMWeights)
		if got != want {
			t.Fatalf("trial %d: dot=%d, portable=%d", trial, got, want)
		}
	}
}
