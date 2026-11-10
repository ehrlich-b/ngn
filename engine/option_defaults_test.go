package engine

import "testing"

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
