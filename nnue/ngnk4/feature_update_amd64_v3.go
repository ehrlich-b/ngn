//go:build amd64 && amd64.v3

package ngnk4

const k4FeatureUpdateUsesAVX2 = true

var (
	_ [HiddenSize - 768]byte
	_ [768 - HiddenSize]byte
)

func addFeatureRow(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	k4AddFeatureRowAVX2(destination, row)
}

func subtractFeatureRow(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	k4SubtractFeatureRowAVX2(destination, row)
}

//go:noescape
func k4AddFeatureRowAVX2(destination *[HiddenSize]int32, row *[HiddenSize]int16)

//go:noescape
func k4SubtractFeatureRowAVX2(destination *[HiddenSize]int32, row *[HiddenSize]int16)
