//go:build amd64 && amd64.v3 && !purego && !race

package engine

// Input bytes are unsigned clipped activations; weights are in [-64,64].
// Saturating int16 pairs are exact: |255*64*2| = 32640 < 32768.
// The complete dot is bounded by 4096*255*64 = 66,846,720 < 2^31.
// dst is disjoint from input and weights; L is a multiple of 16, 16..64.
//
//go:noescape
func ngnn4Linear(dst []int32, input []uint8, weights []int8)

// Four natural weight rows share a 32-byte input load. All input lengths are
// multiples of 32; L is 16..64 in multiples of 16. Operands are disjoint.
// The same unsigned-input/signed-weight pair and complete-dot bounds apply.
//
//go:noescape
func ngnn4LinearRows(dst []int32, input []uint8, weights []int8)

// Clamp int32 lanes before packing; arbitrary int32 endpoints are supported.
// Both perspectives have the validated H, a multiple of 16. dst is disjoint.
//
//go:noescape
func ngnn4PackInput(dst []uint8, us, them []int32)

// Fused signed division by 64, bias, clipping, square and wide-weight dot.
// All slices have the validated L (16..64, a multiple of 16).
//
//go:noescape
func ngnn4Output(affine []int32, bias, weights []int16) int64
