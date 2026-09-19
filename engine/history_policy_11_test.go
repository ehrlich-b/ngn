//go:build h1producer && h1consumer

package engine

import "testing"

func TestH1CompiledPolicy11(t *testing.T) {
	if got := HistoryPolicyName(); got != "11" {
		t.Fatalf("policy=%s want=11", got)
	}
}
