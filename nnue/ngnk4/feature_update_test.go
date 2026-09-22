package ngnk4

import (
	"math/rand"
	"testing"
)

func TestK4FeatureUpdatesMatchPortableOracle(t *testing.T) {
	random := rand.New(rand.NewSource(26092002))
	for trial := 0; trial < 128; trial++ {
		var start [HiddenSize]int32
		var row [HiddenSize]int16
		for index := 0; index < HiddenSize; index++ {
			start[index] = int32(random.Uint32())
			row[index] = int16(random.Uint32())
		}
		got, want := start, start
		addFeatureRow(&got, &row)
		addFeatureRowPortable(&want, &row)
		if got != want {
			t.Fatalf("trial %d: AVX2/add differed from portable oracle", trial)
		}
		subtractFeatureRow(&got, &row)
		subtractFeatureRowPortable(&want, &row)
		if got != want || got != start {
			t.Fatalf("trial %d: AVX2/subtract differed from portable oracle", trial)
		}
	}
}
