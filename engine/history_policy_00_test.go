//go:build !h1producer && !h1consumer

package engine

import "testing"

func TestH1CompiledPolicy00(t *testing.T) {
	if got := HistoryPolicyName(); got != "00" {
		t.Fatalf("policy=%s want=00", got)
	}
}
