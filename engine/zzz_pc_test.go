package engine

import "testing"

func TestZZZProbcutFireRate(t *testing.T) {
	if testing.Short() {
		t.Skip("probe only")
	}
	for _, tc := range []struct {
		name, fen string
		depth     int
	}{
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 12},
		{"mid", "r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10", 12},
		{"end", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 16},
	} {
		pos, _ := ParseFEN(tc.fen)
		ClearStop()
		ClearHistoryTable()
		TranspositionTable = NewCache(DEFAULT_CACHE_SIZE)
		zzPCBlocked, zzPCEligible = 0, 0
		info := SearchFixed(pos, tc.depth, nil)
		tot := zzPCBlocked + zzPCEligible
		pctBlocked := 0.0
		if tot > 0 {
			pctBlocked = float64(zzPCBlocked) * 100 / float64(tot)
		}
		fireOfNodes := 0.0
		if info != nil && info.Nodes > 0 {
			fireOfNodes = float64(zzPCBlocked) * 100 / float64(info.Nodes)
		}
		t.Logf("%-9s nodes=%-9d probcut-eligible=%-7d blocked-by-cutNode=%-7d (%.1f%% of eligible, %.3f%% of nodes)",
			tc.name, info.Nodes, tot, zzPCBlocked, pctBlocked, fireOfNodes)
	}
}
