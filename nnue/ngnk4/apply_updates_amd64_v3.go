//go:build amd64 && amd64.v3

package ngnk4

const k4ApplyUpdatesUsesAVX2 = true

// These pairs fail compilation if the fixed K4 hidden width changes without
// a corresponding kernel review.
var (
	_ [HiddenSize - 768]byte
	_ [768 - HiddenSize]byte
)

func k4ApplyUpdates2(
	destination, add0, subtract1 *[HiddenSize]int16,
) {
	k4ApplyUpdates2AVX2(destination, add0, subtract1)
}

func k4ApplyUpdates3(
	destination, add0, subtract1, subtract2 *[HiddenSize]int16,
) {
	k4ApplyUpdates3AVX2(destination, add0, subtract1, subtract2)
}

func k4ApplyUpdates4(
	destination, add0, subtract1, add2, subtract3 *[HiddenSize]int16,
) {
	k4ApplyUpdates4AVX2(destination, add0, subtract1, add2, subtract3)
}

//go:noescape
func k4ApplyUpdates2AVX2(
	destination, add0, subtract1 *[HiddenSize]int16,
)

//go:noescape
func k4ApplyUpdates3AVX2(
	destination, add0, subtract1, subtract2 *[HiddenSize]int16,
)

//go:noescape
func k4ApplyUpdates4AVX2(
	destination, add0, subtract1, add2, subtract3 *[HiddenSize]int16,
)
