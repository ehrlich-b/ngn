//go:build amd64 && amd64.v3 && !purego && !race

package engine

// Input bytes are unsigned clipped activations; weights are in [-64,64].
// Saturating int16 pairs are exact: |255*64*2| = 32640 < 32768.
// The complete dot is bounded by 4096*255*64 = 66,846,720 < 2^31.
// dst is disjoint from input and weights; L is a multiple of 16, 16..64.
//
//go:noescape
func ngnn4Linear(dst []int32, input []uint8, weights []int8)

// Clamp int32 lanes before packing; arbitrary int32 endpoints are supported.
// Both perspectives have the validated H, a multiple of 16. dst is disjoint.
//
//go:noescape
func ngnn4PackInput(dst []uint8, us, them []int32)
