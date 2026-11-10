package engine

import (
	"math"
	"testing"
)

func TestHundredthLMRTunables(t *testing.T) {
	oldBase, oldDivisor := tuneLMRBase, tuneLMRDivisor
	t.Cleanup(func() {
		tuneLMRBase, tuneLMRDivisor = oldBase, oldDivisor
		rebuildLMRTable()
	})
	for _, tc := range []struct {
		base, divisor int
	}{{0, 200}, {1, 200}, {100, 200}, {0, 260}, {43, 259}} {
		tuneLMRBase, tuneLMRDivisor = tc.base, tc.divisor
		rebuildLMRTable()
		for d := 1; d < 64; d++ {
			for m := 1; m < 64; m++ {
				want := float64(tc.base)/100 + math.Log(float64(d))*math.Log(float64(m))/(float64(tc.divisor)/100)
				if lmrTable[d][m] != int(want) || lmrFractionalTable[d][m] != int(want*lmrPly) {
					t.Fatalf("base=%d divisor=%d depth=%d move=%d", tc.base, tc.divisor, d, m)
				}
			}
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
