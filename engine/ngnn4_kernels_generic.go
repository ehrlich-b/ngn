//go:build !amd64 || !amd64.v3 || purego || race

package engine

func ngnn4Linear(dst []int32, input []uint8, weights []int8) {
	ngnn4LinearGo(dst, input, weights)
}

func ngnn4PackInput(dst []uint8, us, them []int32) {
	ngnn4PackInputGo(dst, us, them)
}
