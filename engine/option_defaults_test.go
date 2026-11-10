package engine

import (
	"strings"
	"testing"
)

func TestAcceptedOptionDefaultsApplied(t *testing.T) {
	for _, p := range TunableSearchParams {
		if value, ok := acceptedOptionDefaults[p.Name]; ok {
			if value != 1 || p.Def != value || *p.Ptr != value {
				t.Fatalf("%s: accepted default not applied: %+v", p.Name, p)
			}
		}
	}
	if newUCITimeManager().scoreTime.enabled != (acceptedOptionDefaults["W2ScoreTime"] == 1) {
		t.Fatal("accepted score-time default not applied to UCI time manager")
	}
}

// Historical node fixtures record the original all-off search configuration.
// Restore the caller's defaults so other tests still exercise accepted options.
func originalSearchOptions(t *testing.T) {
	t.Helper()
	saved := make(map[*int]int)
	for _, p := range TunableSearchParams {
		if strings.HasPrefix(p.Name, "T1104") || strings.HasPrefix(p.Name, "W2") {
			saved[p.Ptr] = *p.Ptr
			*p.Ptr = 0
		}
	}
	t.Cleanup(func() {
		for ptr, value := range saved {
			*ptr = value
		}
	})
}
