package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

func TestSPSACandidateOptions(t *testing.T) {
	data, err := os.ReadFile("../experiments/burn-1104/spsa/final-vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint struct {
		Iter    int                `json:"iter"`
		Theta   map[string]float64 `json:"theta"`
		Rounded map[string]int     `json:"rounded"`
	}
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.Iter != 1100 || len(checkpoint.Theta) != 50 || len(checkpoint.Rounded) != 50 {
		t.Fatal("unexpected checkpoint")
	}
	f, err := os.Open("../experiments/burn-1104/spsa/candidate.uci")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	seen := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var name string
		var value int
		if n, err := fmt.Sscanf(scanner.Text(), "setoption name %s value %d", &name, &value); err != nil || n != 2 || scanner.Text() != fmt.Sprintf("setoption name %s value %d", name, value) {
			t.Fatalf("invalid command %q", scanner.Text())
		}
		theta, ok := checkpoint.Theta[name]
		if !ok || seen[name] || value != checkpoint.Rounded[name] || value != int(math.Round(theta)) {
			t.Fatalf("invalid value for %s", name)
		}
		seen[name] = true
		var param *TunableParam
		for i := range TunableSearchParams {
			if strings.EqualFold(TunableSearchParams[i].Name, name) {
				param = &TunableSearchParams[i]
				break
			}
		}
		if param == nil || value < param.Min || value > param.Max {
			t.Fatalf("unknown or out of range: %s", name)
		}
		old := *param.Ptr
		if !setTunableParam(name, fmt.Sprint(value)) || *param.Ptr != value {
			t.Fatalf("cannot apply %s", name)
		}
		setTunableParam(name, fmt.Sprint(old))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != len(checkpoint.Theta) {
		t.Fatalf("applied %d of %d", len(seen), len(checkpoint.Theta))
	}
}
