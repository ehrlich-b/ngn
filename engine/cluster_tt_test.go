package engine

import "testing"

func TestClusterTTRetainsFourFullKeysIncludingZero(t *testing.T) {
	c := NewCache(1)
	move := NewMove(A2, A3, WhitePawn, NoPiece, NoType, 0)
	for i := 0; i < 4; i++ {
		c.Set(uint64(i)*(uint64(c.mask)+1), move, int16(i), int8(4+i*4), LowerBound, false)
	}
	for i := 0; i < 4; i++ {
		m, e, d, b, hit, pv := c.Get(uint64(i) * (uint64(c.mask) + 1))
		if !hit || m != move || e != int16(i) || d != int8(4+i*4) || b != LowerBound || pv {
			t.Fatalf("lost way %d", i)
		}
	}
	newHash := 4 * (uint64(c.mask) + 1)
	c.Set(newHash, move, 90, 0, LowerBound, false)
	if _, _, _, _, hit, _ := c.Get(newHash); hit {
		t.Fatal("fresh shallow bound evicted deep entry")
	}
	c.AdvanceAge()
	c.AdvanceAge()
	c.Set(newHash, move, 90, 0, LowerBound, false)
	if _, _, _, _, hit, _ := c.Get(newHash); !hit {
		t.Fatal("aged weak entry was not replaced")
	}
	if _, _, _, _, hit, _ := c.Get(0); hit {
		t.Fatal("wrong victim: weakest full key survived")
	}
	for i := 1; i < 4; i++ {
		if _, _, _, _, hit, _ := c.Get(uint64(i) * (uint64(c.mask) + 1)); !hit {
			t.Fatal("deeper entry was evicted")
		}
	}
	c.Clear()
	if _, _, _, _, hit, _ := c.Get(newHash); hit {
		t.Fatal("clear retained entry")
	}
	if c.Consumed() != 0 {
		t.Fatal("clear retained occupancy")
	}
}

func TestClusterTTMatchingKeyKeepsDepthAndRetainsMove(t *testing.T) {
	c := NewCache(1)
	move := NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle)
	c.Set(0, move, 70, 9, Exact, true)
	c.Set(0, EmptyMove, -50, 0, LowerBound, false)
	m, e, d, b, hit, pv := c.Get(0)
	if !hit || m != move || e != 70 || d != 9 || b != Exact || !pv {
		t.Fatal("shallow bound displaced stronger record")
	}
	c.Set(0, EmptyMove, 75, 10, Exact, true)
	m, e, d, b, hit, pv = c.Get(0)
	if !hit || m != move || e != 75 || d != 10 || b != Exact || !pv {
		t.Fatal("deeper update lost retained move")
	}
}

func TestClusterTTEmptyMoveZeroScoreIsOccupied(t *testing.T) {
	c := NewCache(1)
	if _, _, _, _, hit, _ := c.Get(0); hit {
		t.Fatal("empty table hits zero key")
	}
	c.Set(0, EmptyMove, 0, 0, Exact, false)
	m, e, d, b, hit, pv := c.Get(0)
	if !hit || m != EmptyMove || e != 0 || d != 0 || b != Exact || pv {
		t.Fatal("zero record confused with empty slot")
	}
}
