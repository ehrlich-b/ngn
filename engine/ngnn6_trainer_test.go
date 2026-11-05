package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// This packet is trained for two CPU batches; its expected scores are produced
// by the independent NumPy integer evaluator, not a float tolerance.
func TestWave2ResidualParityTrained(t *testing.T) {
	const dir = "../experiments/wave2-1104/residual-trainer"
	raw, err := os.ReadFile(filepath.Join(dir, "trained-h32.ngn6"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(dir, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packet struct {
		SHA256 string
		Cases  []struct {
			FEN             string
			Bucket, STM, CP int
			ResidualCP      int `json:"residual_cp"`
			Acc             []int32
		}
	}
	if err := json.Unmarshal(expected, &packet); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	if packet.SHA256 != hex.EncodeToString(hash[:]) || len(packet.Cases) != 16 {
		t.Fatal("invalid trained packet")
	}
	net, err := ReadNGNN1(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if net.version != 6 || net.residual == nil {
		t.Fatal("expected residual evaluator")
	}
	coverage := make(map[[2]int]bool)
	for _, c := range packet.Cases {
		p, err := ParseFEN(c.FEN)
		if err != nil {
			t.Fatal(err)
		}
		stm := 0
		if p.Turn() == Black {
			stm = 1
		}
		if stm != c.STM || net.bucket(p) != c.Bucket || c.ResidualCP == 0 {
			t.Fatal("incorrect bucket, STM or nonzero control")
		}
		coverage[[2]int{c.Bucket, stm}] = true
		for _, wide := range []bool{false, true} {
			n := *net
			if !n.compactInput {
				t.Fatal("fixture unexpectedly needs wide input")
			}
			n.compactInput = !wide
			e := ngnn1NewWorker(t, &n, p)
			acc := make([]int32, 2*n.hidden)
			n.refresh(p, acc)
			if !reflect.DeepEqual(acc, c.Acc) || !reflect.DeepEqual(e.nnue.at(0), c.Acc) {
				t.Fatal("trained accumulator mismatch")
			}
			if got := e.SearchSTM(p); got != c.CP || n.evaluateBucket(acc, p.Turn(), c.Bucket) != c.CP {
				t.Fatalf("trained NumPy/Go mismatch: %s got=%d want=%d wide=%v", c.FEN, got, c.CP, wide)
			}
		}
	}
	if len(coverage) != 16 {
		t.Fatal("must cover eight buckets and both turns")
	}
	t.Log("16 trained NumPy/Go exact integer scores; all nonzero residuals; eight buckets x both STM; compact/wide")
}
