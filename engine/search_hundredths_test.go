package engine

import "testing"

func TestHundredthLMRTunables(t *testing.T) {
	oldBase, oldDivisor := tuneLMRBase, tuneLMRDivisor
	t.Cleanup(func() {
		tuneLMRBase, tuneLMRDivisor = oldBase, oldDivisor
		rebuildLMRTable()
	})
	for _, tc := range []struct {
		base, divisor, depth, move, whole, fractional int
	}{
		{0, 200, 20, 20, 4, 4594},  // Original whole-number default.
		{1, 200, 1, 1, 0, 10},      // One hundredth survives in the fractional table.
		{100, 200, 1, 1, 1, 1024},  // One full ply.
		{0, 260, 20, 20, 3, 3534},  // Divisor 2.60 changes the whole reduction too.
		{43, 259, 20, 20, 3, 3988}, // Both inputs have fractional parts.
	} {
		tuneLMRBase, tuneLMRDivisor = tc.base, tc.divisor
		rebuildLMRTable()
		if lmrTable[tc.depth][tc.move] != tc.whole || lmrFractionalTable[tc.depth][tc.move] != tc.fractional {
			t.Fatalf("base=%d divisor=%d: whole=%d fractional=%d, want %d/%d", tc.base, tc.divisor,
				lmrTable[tc.depth][tc.move], lmrFractionalTable[tc.depth][tc.move], tc.whole, tc.fractional)
		}
	}
	setTunableParam("LMRBase", "0")
	setTunableParam("LMRDivisor", "200")
	before := lmrFractionalTable[20][20]
	if !setTunableParam("LMRDivisor", "260") || lmrFractionalTable[20][20] >= before {
		t.Fatal("UCI divisor 2.60 must reduce the LMR table relative to 2.00")
	}
	setTunableParam("LMRBase", "1")
	if lmrFractionalTable[1][1] != 10 {
		t.Fatal("UCI LMRBase=1 must represent 0.01 ply")
	}
}

func TestHundredthNullMoveDepthDivisor(t *testing.T) {
	old := tuneNullMoveDepthDivisor
	t.Cleanup(func() { tuneNullMoveDepthDivisor = old })
	for _, tc := range []struct {
		divisor     string
		depth, want int
	}{{"600", 12, 5}, {"601", 12, 6}, {"569", 29, 19}, {"650", 12, 6}} {
		t.Run(tc.divisor, func(t *testing.T) {
			p, info := wave2NullState(t)
			W2NullVerify = 0
			if !setTunableParam("NullMoveDepthDivisor", tc.divisor) {
				t.Fatal("missing UCI divisor")
			}
			calls := 0
			info.nullSearch = func(_ *Position, depth, _, _, _ int, _, _, _ bool, _ *SearchInfo) int {
				calls++
				if depth != tc.want {
					t.Fatalf("child depth=%d want=%d", depth, tc.want)
				}
				return -100
			}
			alphaBetaPV(p, tc.depth, 2, 99, 100, false, true, true, info)
			if calls != 1 {
				t.Fatalf("null child calls=%d", calls)
			}
		})
	}
}
