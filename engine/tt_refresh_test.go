package engine

import "testing"

func TestTTMatchingRefreshWindow(t *testing.T) {
	for _, mode := range []TTMode{TTDirect, TTSynchronized} {
		oldMove := NewMove(A2, A3, WhitePawn, NoPiece, NoType, 0)
		newMove := NewMove(B2, B4, WhitePawn, NoPiece, NoType, 0)
		for _, depth := range []int8{7, 8, 9, 10} {
			c := newCacheWithMode(1, mode)
			c.Set(0, oldMove, 70, 10, LowerBound, false)
			c.Set(0, newMove, -30, depth, LowerBound, false)
			m, e, d, b, hit, pv := c.Get(0)
			if depth >= 8 {
				if !hit || m != newMove || e != -30 || d != depth || b != LowerBound || pv {
					t.Fatalf("mode=%d depth=%d: refresh lost coherent tuple: %v %d %d %v %v %v", mode, depth, m, e, d, b, hit, pv)
				}
			} else if !hit || m != oldMove || e != 70 || d != 10 || b != LowerBound || pv {
				t.Fatalf("mode=%d: substantially shallower bound displaced deeper tuple", mode)
			}
		}
	}
}

func TestTTMatchingRefreshKeepsBoundQualityAndEmptyMove(t *testing.T) {
	c := NewCache(1)
	move := NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle)
	c.Set(9, move, 90, 10, Exact, true)
	// Two plies plus the exact/PV bonuses exceed the refresh allowance.
	c.Set(9, EmptyMove, -50, 8, LowerBound, false)
	if m, e, d, b, hit, pv := c.Get(9); !hit || m != move || e != 90 || d != 10 || b != Exact || !pv {
		t.Fatal("refresh erased exact/PV preference")
	}
	// Matching quality permits the refreshed score/depth and retains the move.
	c.Set(9, EmptyMove, 80, 8, Exact, true)
	if m, e, d, b, hit, pv := c.Get(9); !hit || m != move || e != 80 || d != 8 || b != Exact || !pv {
		t.Fatal("refresh lost empty-move fallback or updated bound")
	}
}

func TestTTRefreshAllowanceDoesNotApplyToCollisions(t *testing.T) {
	c := NewCache(1)
	stride := uint64(c.mask) + 1
	for i := uint64(0); i < ttWays; i++ {
		c.Set(i*stride, EmptyMove, int16(i), 10, LowerBound, false)
	}
	c.Set(ttWays*stride, EmptyMove, -30, 8, LowerBound, false)
	if _, _, _, _, hit, _ := c.Get(ttWays * stride); hit {
		t.Fatal("matching refresh allowance changed collision victim policy")
	}
	for i := uint64(0); i < ttWays; i++ {
		if _, _, d, _, hit, _ := c.Get(i * stride); !hit || d != 10 {
			t.Fatal("near-depth colliding key displaced an existing record")
		}
	}
}
