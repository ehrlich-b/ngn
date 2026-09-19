//go:build h1producer && !h1consumer

package engine

import "testing"

func TestH1CompiledPolicy10(t *testing.T) {
	if got := HistoryPolicyName(); got != "10" {
		t.Fatalf("policy=%s want=10", got)
	}
}
