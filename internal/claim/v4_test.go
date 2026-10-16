package claim

import (
	"reflect"
	"strings"
	"testing"
)

func TestV4ResourcesAndScheduler(t *testing.T) {
	for _, mode := range []string{"screen", "confirmation"} {
		p, err := MakePlan(mode, Identity{}, Artifact{}, "/owned/book")
		if err != nil || p.Threads != 8 || p.HashMB != 256 || p.CPUSet != "0,2,4,6,8,10,12,14" || p.Concurrency != 1 {
			t.Fatalf("wrong resources: %+v %v", p, err)
		}
		for _, mutate := range []func(*Plan){
			func(p *Plan) { p.Threads = 1 },
			func(p *Plan) { p.HashMB = 64 },
			func(p *Plan) { p.Concurrency = 8 },
			func(p *Plan) { p.CPUSet = "0-15" },
			func(p *Plan) { p.Schema = "NGN-CLAIM-3" },
		} {
			changed := p
			mutate(&changed)
			if validatePlan(changed) == nil {
				t.Fatal("changed v4 resource contract admitted")
			}
		}
	}
	parent := []string{"PATH=/owned/bin", "GOMAXPROCS=1", "OTHER=value", "GOMAXPROCS=2"}
	before := append([]string(nil), parent...)
	got := engineEnvironment(parent)
	if !reflect.DeepEqual(parent, before) || !reflect.DeepEqual(got, []string{"PATH=/owned/bin", "OTHER=value", "GOMAXPROCS=8"}) {
		t.Fatalf("serialized/duplicate scheduler: %v", got)
	}
}

func TestV4ProbeRequiresEightThreadsAndHash256(t *testing.T) {
	id := Identity{}
	s := receiptFixtureSession(Artifact{Path: "/owned/ngn"}, &id, true, "HCE")
	if err := validateProbe(s, true, "HCE"); err != nil {
		t.Fatal(err)
	}
	for _, options := range []map[string]string{
		{"Threads": "spin default 1 min 1 max 4", "Hash": "spin default 64 min 1 max 1024"},
		{"Threads": "spin default 1 min 1 max 8", "Hash": "spin default 64 min 1 max 128"},
		{"Hash": "spin default 64 min 1 max 1024"},
	} {
		if validateResources(options, s.Selected) == nil {
			t.Fatal("unsupported v4 resources admitted")
		}
	}
	for i := range s.Events {
		if strings.Contains(s.Events[i].Line, "threads configured 8 effective 8") {
			s.Events[i].Line = "info string threads configured 8 effective 1"
		}
	}
	if validateProbe(s, true, "HCE") == nil {
		t.Fatal("serialized NGN probe admitted")
	}
}
