//go:build !h1producer && h1consumer

package engine

import "testing"

func TestH1CompiledPolicy01(t *testing.T) {
	if got := HistoryPolicyName(); got != "01" {
		t.Fatalf("policy=%s want=01", got)
	}
}
