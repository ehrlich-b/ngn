package main

import "testing"

func TestCampaignChunkBoundaries(t *testing.T) {
	for _, tc := range []struct{ start, total, chunk, report, want int }{
		{1, 1200, 100, 1, 100}, {101, 1200, 100, 1, 200},
		{1101, 1200, 100, 1, 1200}, {1199, 1200, 100, 25, 1200},
		{1, 1200, 0, 25, 1200}, {1, 1200, 2, 1, 2}, {3, 1200, 1, 1, 3},
	} {
		got, err := campaignEnd(tc.start, tc.total, tc.chunk, tc.report)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: end=%d err=%v", tc, got, err)
		}
	}
	for _, chunk := range []int{-1, 2, 26} {
		if _, err := campaignEnd(1, 1200, chunk, 25); err == nil {
			t.Fatalf("accepted invalid chunk %d", chunk)
		}
	}
}
