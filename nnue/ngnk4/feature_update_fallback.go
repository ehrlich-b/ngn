//go:build !amd64 || !amd64.v3

package ngnk4

const k4FeatureUpdateUsesAVX2 = false

func addFeatureRow(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	addFeatureRowPortable(destination, row)
}

func subtractFeatureRow(destination *[HiddenSize]int32, row *[HiddenSize]int16) {
	subtractFeatureRowPortable(destination, row)
}
