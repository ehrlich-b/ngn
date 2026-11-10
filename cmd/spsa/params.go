package main

import (
	"fmt"
	"strings"

	"github.com/ehrlich-b/ngn/engine"
)

// tuningParams rejects ranges too coarse for continuous SPSA perturbations.
// Discrete flags remain fixed engine options and need separate A/B tests.
func tuningParams(registry []engine.TunableParam, names string) ([]*pvar, error) {
	selected := make(map[string]bool)
	if names != "" {
		for _, name := range strings.Split(names, ",") {
			name = strings.TrimSpace(name)
			if name == "" || selected[name] {
				return nil, fmt.Errorf("empty or duplicate parameter %q in -params", name)
			}
			selected[name] = true
		}
	}
	var params []*pvar
	for _, tp := range registry {
		if names != "" && !selected[tp.Name] {
			continue
		}
		if tp.Max < tp.Min || float64(tp.Max)-float64(tp.Min) < 20 {
			return nil, fmt.Errorf("parameter %s range [%d,%d] has fewer than 20 steps; use -params to select wider ranges and test discrete options separately", tp.Name, tp.Min, tp.Max)
		}
		params = append(params, &pvar{name: tp.Name, theta: float64(tp.Def), min: float64(tp.Min), max: float64(tp.Max)})
		delete(selected, tp.Name)
	}
	if len(selected) != 0 {
		return nil, fmt.Errorf("unknown parameter in -params: %s", names)
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("no tunable params selected")
	}
	return params, nil
}
