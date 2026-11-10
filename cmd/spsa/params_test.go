package main

import (
	"strings"
	"testing"

	"github.com/ehrlich-b/ngn/engine"
)

func TestTuningParamsMinimumSteps(t *testing.T) {
	for _, tc := range []struct {
		name     string
		min, max int
		wantOK   bool
	}{
		{"flag", 0, 1, false},
		{"nineteen steps", 100, 119, false},
		{"twenty steps", 100, 120, true},
		{"hundred steps", 100, 200, true},
		{"reversed", 200, 100, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := []engine.TunableParam{{Name: tc.name, Def: tc.min, Min: tc.min, Max: tc.max}}
			for _, selection := range []string{"", tc.name} {
				params, err := tuningParams(registry, selection)
				if (err == nil) != tc.wantOK {
					t.Fatalf("selection=%q params=%v err=%v", selection, params, err)
				}
				if !tc.wantOK && !strings.Contains(err.Error(), "fewer than 20 steps") {
					t.Fatalf("missing actionable rejection: %v", err)
				}
			}
		})
	}
}

func TestTuningParamsSelection(t *testing.T) {
	registry := engine.TunableSearchParams
	params, err := tuningParams(registry, "LMRDivisor, LMRBase, NullMoveDepthDivisor")
	if err != nil || len(params) != 3 {
		t.Fatalf("params=%v err=%v", params, err)
	}
	// Registry order, rather than CLI order, keeps the campaign recipe stable.
	if params[0].name != "LMRBase" || params[1].theta != 200 || params[2].theta != 600 {
		t.Fatalf("wrong registry values: %+v", params)
	}
	for _, selection := range []string{"", "T1104CorrMore", "NullMoveR", "LMRBase,T1104CorrMore", "LMRBase,missing", "LMRBase,LMRBase", "LMRBase,"} {
		if _, err := tuningParams(registry, selection); err == nil {
			t.Fatalf("accepted %q", selection)
		}
	}
	if _, err := tuningParams(nil, ""); err == nil {
		t.Fatal("accepted empty registry")
	}
}
