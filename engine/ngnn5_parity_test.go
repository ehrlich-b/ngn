package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type ngnn5Positions struct{ Scores, Search []string }

func ngnn5Panel(t *testing.T) ngnn5Positions {
	t.Helper()
	data, err := os.ReadFile("../testdata/ngnn5/positions.json")
	if err != nil {
		t.Fatal(err)
	}
	var panel ngnn5Positions
	if err := json.Unmarshal(data, &panel); err != nil {
		t.Fatal(err)
	}
	if len(panel.Scores) != 20 || len(panel.Search) != 12 {
		t.Fatal("parity panel must contain 20 scores and 12 searches")
	}
	return panel
}

func ngnn5ConvertedDefault(t *testing.T) (*NGNN1Network, *NGNN1Network) {
	t.Helper()
	base, err := LoadDefaultNet()
	if err != nil {
		t.Fatal(err)
	}
	if base.version != 3 || base.backendName() != "ngnn3" {
		t.Fatal("embedded default must remain NGNN3")
	}
	var converted bytes.Buffer
	if err := ConvertNGNN3ToNGNN5(&converted, bytes.NewReader(DefaultNetBytes())); err != nil {
		t.Fatal(err)
	}
	n, err := ReadNGNN1(bytes.NewReader(converted.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if n.version != 5 || n.backendName() != "ngnn5" || len(n.threatWeights) != 0 {
		t.Fatal("converter format/zero threat mismatch")
	}
	if !reflect.DeepEqual(base.weights, n.weights) || !reflect.DeepEqual(base.bias, n.bias) || !reflect.DeepEqual(base.output, n.output) || base.kingMap != n.kingMap || base.outputBias != n.outputBias || !reflect.DeepEqual(base.otherBiases, n.otherBiases) {
		t.Fatal("converter changed NGNN3 coefficients")
	}
	t.Logf("source=NGNN3 sha256=%x converted=NGNN5 sha256=%x zero-threats=true H=%d KB=%d NB=%d F=%d", sha256.Sum256(DefaultNetBytes()), sha256.Sum256(converted.Bytes()), n.hidden, n.kingBuckets, n.buckets, ngnn5Features)
	return base, n
}

func ngnn5Report(t *testing.T, name string, value any) {
	t.Helper()
	dir := os.Getenv("NGN_NGNN5_REPORT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNGNN5ScoreParity(t *testing.T) {
	base, n := ngnn5ConvertedDefault(t)
	type row struct {
		FEN          string
		NGNN3, NGNN5 int
	}
	var rows []row
	for _, fen := range ngnn5Panel(t).Scores {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		a, b := ngnn1NewWorker(t, base, pos), ngnn1NewWorker(t, n, pos)
		if !reflect.DeepEqual(a.nnue.at(0), b.nnue.at(0)) || a.SearchSTM(pos) != b.SearchSTM(pos) {
			t.Fatalf("zero threat score mismatch %s", fen)
		}
		rows = append(rows, row{fen, a.SearchSTM(pos), b.SearchSTM(pos)})
	}
	ngnn5Report(t, "scores.json", rows)
	t.Log("20/20 exact scores and accumulators; embedded default NGNN3")
}

func ngnn5SearchEngine(t *testing.T, n *NGNN1Network) *SearchEngine {
	t.Helper()
	e, err := NewSearchEngineWithHash(16)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ConfigureThreads(1); err != nil {
		t.Fatal(err)
	}
	if err := e.SelectNGNN1Evaluator(n); err != nil {
		t.Fatal(err)
	}
	if e.ThreadCount() != 1 || e.SelectedEvaluatorBackend() != n.backendName() {
		t.Fatal("search format/Threads mismatch")
	}
	return e
}

func TestNGNN5SearchParity(t *testing.T) {
	if os.Getenv("NGN_NGNN5_PARITY") != "1" {
		t.Skip("set NGN_NGNN5_PARITY=1 for the 36 depth-10 search pairs")
	}
	base, n := ngnn5ConvertedDefault(t)
	type result struct {
		Depth   int
		Nodes   uint64
		Score   int
		Best    string
		PV      []string
		Seconds float64
	}
	type row struct {
		Position     int
		Mode, FEN    string
		NGNN3, NGNN5 result
		NPSRatio     float64
	}
	var rows []row
	for index, fen := range ngnn5Panel(t).Search {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			// Two cold engines; the third pass deliberately retains the preceding
			// iterative search's TT/history in each engine, checking warm parity.
			var engines [2]*SearchEngine
			for _, mode := range []string{"fixed", "iterative", "warm-iterative"} {
				var pair [2]result
				for side, network := range []*NGNN1Network{base, n} {
					if mode != "warm-iterative" {
						engines[side] = ngnn5SearchEngine(t, network)
					}
					e := engines[side]
					pos, err := ParseFEN(fen)
					if err != nil {
						t.Fatal(err)
					}
					start := time.Now()
					var info *SearchInfo
					if mode == "fixed" {
						info = e.SearchFixed(pos, 10, nil)
					} else {
						info = e.SearchIterativeDeepening(pos, 10, nil)
					}
					elapsed := time.Since(start).Seconds()
					if info.Stopped || info.Depth != 10 || info.Nodes == 0 {
						t.Fatalf("incomplete depth-10 search: %+v", info)
					}
					pv := []string{}
					for _, move := range e.ExtractPVFromTT(pos, info.BestMove, 10) {
						pv = append(pv, move.ToString())
					}
					pair[side] = result{info.Depth, info.Nodes, info.BestScore, info.BestMove.ToString(), pv, elapsed}
				}
				a, b := pair[0], pair[1]
				a.Seconds, b.Seconds = 0, 0
				if !reflect.DeepEqual(a, b) {
					t.Fatalf("search parity differs mode=%s NGNN3=%+v NGNN5=%+v", mode, a, b)
				}
				ratio := pair[0].Seconds / pair[1].Seconds
				rows = append(rows, row{index, mode, fen, pair[0], pair[1], ratio})
				t.Logf("%s depth=10 nodes=%d score=%d best=%s NGNN3=%.6fs NGNN5=%.6fs ratio=%.4f", mode, a.Nodes, a.Score, a.Best, pair[0].Seconds, pair[1].Seconds, ratio)
			}
		})
	}
	if len(rows) != 36 {
		t.Fatal("missing search pairs")
	}
	ngnn5Report(t, "search-speed.json", struct {
		Note     string
		Searches any
	}{"36 depth-10 paired searches on a shared host; diagnostic elapsed times, not a quiet-rig speed claim; long speedcheck deferred", rows})
	t.Log("36/36 exact depth-10 node/score/best/PV search pairs at Threads=1")
}

func TestNGNN5FeatureCounts(t *testing.T) {
	_, n := ngnn5ConvertedDefault(t)
	type row struct {
		Position                                      int
		FEN                                           string
		Depth                                         int
		Nodes, Samples, PieceFeatures, ThreatFeatures uint64
		MeanPiece, MeanThreat, MeanTotal              float64
	}
	var rows []row
	for index, fen := range ngnn5Panel(t).Search {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		e := ngnn5SearchEngine(t, n)
		worker := e.mustPreparePrimaryEvaluator(pos, mustAcquireHCEModelUse())
		releaseHCEModelUse()
		var samples, pieces, threats uint64
		worker.nodeObserver = func() { p, t := n.ngnn5ActiveFeatures(pos); samples++; pieces += uint64(p); threats += uint64(t) }
		info := e.SearchFixed(pos, 4, nil)
		worker.nodeObserver = nil
		if info.Stopped || info.Depth != 4 || samples != info.Nodes || samples == 0 || pieces == 0 || threats == 0 {
			t.Fatalf("feature sample mismatch position=%d samples=%d nodes=%d", index, samples, info.Nodes)
		}
		row := row{index, fen, 4, info.Nodes, samples, pieces, threats, float64(pieces) / float64(samples), float64(threats) / float64(samples), float64(pieces+threats) / float64(samples)}
		rows = append(rows, row)
		t.Logf("position=%d nodes=%d samples=%d piece/node=%.3f threat/node=%.3f total/node=%.3f", index, info.Nodes, samples, row.MeanPiece, row.MeanThreat, row.MeanTotal)
	}
	ngnn5Report(t, "active-features.json", struct {
		Definition string
		Positions  any
	}{"Every increment of SearchInfo.Nodes samples the current board (including root/parent counts after undo); both perspective piece blocks plus directed occupied attacks by each perspective's own pieces; independent of zero weights; fixed depth 4, Threads=1", rows})
}
