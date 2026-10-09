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
func ngnn4LinearRows(dst []int32, input []uint8, weights []int8) {
	if ngnn4UseVNNI {
		ngnn4LinearRowsVNNI(dst, input, weights)
	} else {
		ngnn4LinearRowsAVX2(dst, input, weights)
	}
}

//go:noescape
func ngnn4LinearRowsAVX2(dst []int32, input []uint8, weights []int8)

// Four unsigned bytes times four signed weights accumulate directly into
// each int32 lane. The full-dot bound above also bounds every partial sum.
//
//go:noescape
func ngnn4LinearRowsVNNI(dst []int32, input []uint8, weights []int8)

var ngnn4UseVNNI = ngnn4HasVNNI()

func ngnn4HasVNNI() bool {
	maxLeaf, _, _, _ := ngnn4CPUID(0, 0)
	if maxLeaf < 7 {
		return false
	}
	_, _, features, _ := ngnn4CPUID(1, 0)
	if features&(1<<27) == 0 { // OSXSAVE permits XGETBV.
		return false
	}
	_, hardware, vector, _ := ngnn4CPUID(7, 0)
	return ngnn4VNNIState(hardware, vector, ngnn4XCR0())
}

// The EVEX 256-bit dot requires AVX512F, AVX512VL and AVX512VNNI.
// XCR0 must enable XMM, YMM, opmask and both ZMM state components.
func ngnn4VNNIState(hardware, vector uint32, xcr0 uint64) bool {
	const required = uint32(1<<16 | 1<<31)
	return hardware&required == required && vector&(1<<11) != 0 && xcr0&0xe6 == 0xe6
}

//go:noescape
func ngnn4CPUID(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)

//go:noescape
func ngnn4XCR0() uint64

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
