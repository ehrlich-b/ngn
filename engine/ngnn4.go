package engine

func (n *NGNN1Network) validOutputLayout() bool {
	if n.version != 4 {
		return n.hidden2 == 0 && len(n.layer2) == 0 && len(n.bias2) == 0 && len(n.output) == n.bucketCount()*2*n.hidden
	}
	if n.hidden2 < 16 || n.hidden2 > 64 || n.hidden2%16 != 0 ||
		len(n.layer2) != n.bucketCount()*n.hidden2*2*n.hidden || len(n.packed2) != len(n.layer2) ||
		len(n.bias2) != n.bucketCount()*n.hidden2 || len(n.output) != n.bucketCount()*n.hidden2 {
		return false
	}
	for _, w := range n.layer2 {
		if w < -64 || w > 64 {
			return false
		}
	}
	return true
}

// Four adjacent inputs for eight outputs fill one AVX2 register. Serialized
// W2 remains ordinary [NB,L,2H] row order; this immutable layout is loader-only.
func (n *NGNN1Network) packLayer2() {
	n.packed2 = make([]int8, len(n.layer2))
	width := 2 * n.hidden
	for b := 0; b < n.bucketCount(); b++ {
		for i := 0; i < width; i++ {
			for j := 0; j < n.hidden2; j++ {
				n.packed2[b*n.hidden2*width+(i/4*n.hidden2+j)*4+i%4] = n.layer2[(b*n.hidden2+j)*width+i]
			}
		}
	}
}

// NGNN4 uses clipped linear accumulator activations, a narrow learned affine
// layer, then squared clipped activations and the ordinary centipawn scale.
// W2 is quantized by QB; B2 by QA. Signed division truncates toward zero.
func (n *NGNN1Network) evaluateLayer2(acc []int32, turn Color, bucket int) int {
	us, them := acc[:n.hidden], acc[n.hidden:2*n.hidden]
	if turn == Black {
		us, them = them, us
	}
	var hidden [64]int32
	var input [2 * ngnn1MaxHidden]uint8
	ngnn4PackInput(input[:2*n.hidden], us, them)
	start := bucket * n.hidden2
	ngnn4Linear(hidden[:n.hidden2], input[:2*n.hidden], n.packed2[start*2*n.hidden:(start+n.hidden2)*2*n.hidden])
	for j := 0; j < n.hidden2; j++ {
		hidden[j] = hidden[j]/ngnn1QB + int32(n.bias2[start+j])
	}
	sum := ngnn1Dot(hidden[:n.hidden2], n.output[start:start+n.hidden2])
	bias := n.outputBias
	if bucket > 0 {
		bias = n.otherBiases[bucket-1]
	}
	cp := (sum/ngnn1QA + int64(bias)) * ngnn1Scale / (ngnn1QA * ngnn1QB)
	return int(min(ngnn1ScoreLimit, max(-ngnn1ScoreLimit, cp)))
}

func ngnn4PackInputGo(dst []uint8, us, them []int32) {
	for i := range us {
		dst[i] = uint8(min(max(us[i], 0), ngnn1QA))
		dst[len(us)+i] = uint8(min(max(them[i], 0), ngnn1QA))
	}
}

func ngnn4LinearGo(dst []int32, input []uint8, weights []int8) {
	for j := range dst {
		var sum int32
		for i, x := range input {
			sum += int32(x) * int32(weights[(i/4*len(dst)+j)*4+i%4])
		}
		dst[j] = sum
	}
}
