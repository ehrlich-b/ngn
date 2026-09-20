package engine

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ehrlich-b/ngn/nnue/ngnk4"
)

func ngnK4TestModel(t *testing.T, identity string) (*ngnk4.Model, []byte) {
	t.Helper()
	tensors := new(ngnk4.Tensors)
	for hidden := 0; hidden < 8; hidden++ {
		tensors.InputBiases[hidden] = int16(80 + hidden*7)
	}
	for feature := 0; feature < ngnk4.TotalInputFeatures; feature++ {
		for hidden := 0; hidden < 8; hidden++ {
			tensors.InputWeights[feature][hidden] = int16((feature*13+hidden*19)%47 - 23)
		}
	}
	for bucket := 0; bucket < ngnk4.OutputBuckets; bucket++ {
		for perspective := 0; perspective < 2; perspective++ {
			for hidden := 0; hidden < 8; hidden++ {
				tensors.OutputWeights[bucket][perspective][hidden] = int16(9 + bucket*3 + perspective*5 + hidden)
			}
		}
		tensors.OutputBiases[bucket] = int16(300 + bucket*11)
	}
	manifest := sha256.Sum256([]byte("NGN K4 engine test manifest: " + identity))
	data, err := ngnk4.Marshal(tensors, manifest)
	if err != nil {
		t.Fatal(err)
	}
	model, err := ngnk4.Load(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return model, data
}

func TestNGNK4SelectionScorePolicyAndTransaction(t *testing.T) {
	model, _ := ngnK4TestModel(t, "A")
	searcher := NewSearchEngine()
	if err := searcher.SelectNGNK4Evaluator(nil); err == nil {
		t.Fatal("nil NGN K4 selection succeeded")
	}
	if searcher.SelectedEvaluatorBackend() != EvaluatorBackendHCEName {
		t.Fatal("failed NGN K4 selection changed the HCE default")
	}
	if err := searcher.SelectNGNK4Evaluator(model); err != nil {
		t.Fatal(err)
	}
	if got := searcher.SelectedEvaluatorBackend(); got != EvaluatorBackendNGNK4Name {
		t.Fatalf("selected backend = %q", got)
	}
	position := n3cPosition(t, "4k3/8/8/8/8/8/P7/4K3 w - - 50 1")
	k4Position, err := ngnK4PositionFromPosition(position)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := model.EvaluateRaw(k4Position)
	if err != nil {
		t.Fatal(err)
	}
	got, backend, err := searcher.EvaluateSelected(position)
	if err != nil || backend != EvaluatorBackendNGNK4Name || got != nnueSearchScore(raw, position.HalfMoveClock) {
		t.Fatalf("selected evaluation = %d/%q/%v, want %d/%q", got, backend, err, nnueSearchScore(raw, position.HalfMoveClock), EvaluatorBackendNGNK4Name)
	}
	workerBefore, modelBefore := searcher.worker.evaluator, searcher.evaluatorModel
	if err := searcher.SelectNGNK4Evaluator(nil); err == nil {
		t.Fatal("replacement with nil model succeeded")
	}
	if searcher.worker.evaluator != workerBefore || searcher.evaluatorModel != modelBefore {
		t.Fatal("failed replacement changed published evaluator state")
	}
}

func TestNGNK4IncrementalSearchMatchesFullRefresh(t *testing.T) {
	model, _ := ngnK4TestModel(t, "search")
	incremental := NewSearchEngine()
	fullRefresh := NewSearchEngine()
	if err := incremental.SelectNGNK4Evaluator(model); err != nil {
		t.Fatal(err)
	}
	if err := fullRefresh.SelectNGNK4Evaluator(model); err != nil {
		t.Fatal(err)
	}
	observed := 0
	incremental.worker.evaluator.transitionObserver = func(pos *Position, evaluator *workerEvaluator) {
		observed++
		wantPosition, err := ngnK4PositionFromPosition(pos)
		if err != nil {
			t.Fatal(err)
		}
		if gotPosition := evaluator.ngnK4Context.Position(); gotPosition != wantPosition {
			t.Fatal("incremental context position differs from engine board")
		}
		got, err := evaluator.ngnK4Context.EvaluateRaw()
		if err != nil {
			t.Fatal(err)
		}
		want, err := model.EvaluateRaw(wantPosition)
		if err != nil || got != want {
			t.Fatalf("incremental score = %d/%v, full refresh = %d", got, err, want)
		}
	}
	fullRefresh.worker.evaluator.fullRefreshOracle = true
	const fen = "r3k2r/p1ppqpb1/bn2pnp1/2pP4/1p2P3/2N2N2/PPQBBPPP/R3K2R w KQkq - 37 1"
	incrementalPosition := n3cPosition(t, fen)
	fullPosition := n3cPosition(t, fen)
	incrementalBefore := snapshotPVPosition(incrementalPosition)
	fullBefore := snapshotPVPosition(fullPosition)
	got := incremental.SearchFixed(incrementalPosition, 3, nil)
	want := fullRefresh.SearchFixed(fullPosition, 3, nil)
	if observed < 2 {
		t.Fatalf("transition observer ran %d times", observed)
	}
	if !reflect.DeepEqual(comparableN3CSearchInfo(got), comparableN3CSearchInfo(want)) {
		t.Fatalf("incremental and full-refresh search differ:\n%+v\n%+v", comparableN3CSearchInfo(got), comparableN3CSearchInfo(want))
	}
	if !reflect.DeepEqual(snapshotPVPosition(incrementalPosition), incrementalBefore) ||
		!reflect.DeepEqual(snapshotPVPosition(fullPosition), fullBefore) {
		t.Fatal("search did not restore root position")
	}
	if incremental.worker.evaluator.nnueDepth() != 0 || fullRefresh.worker.evaluator.nnueDepth() != 0 {
		t.Fatalf("context depths = %d/%d", incremental.worker.evaluator.nnueDepth(), fullRefresh.worker.evaluator.nnueDepth())
	}
}

func TestNGNK4UCIStartupAndStaging(t *testing.T) {
	model, data := ngnK4TestModel(t, "uci")
	path := filepath.Join(t.TempDir(), "owned.nnue")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	uci := NewUCIEngine()
	if err := uci.ConfigureStartupEvaluator(" NGN-K4-768-V1 ", " "+path+" "); err != nil {
		t.Fatal(err)
	}
	config := requireUCIEvaluatorConfig(t, uci, uciEvaluatorNGNK4, path)
	if config.staged.kind != stagedEvaluatorNGNK4 || config.staged.ngnK4 == nil ||
		uci.searcher.SelectedEvaluatorBackend() != EvaluatorBackendNGNK4Name {
		t.Fatal("startup did not activate and retain NGN K4")
	}
	wantPosition, err := ngnK4PositionFromPosition(uci.position)
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := model.EvaluateRaw(wantPosition)
	if err != nil {
		t.Fatal(err)
	}
	got, backend, err := uci.searcher.EvaluateSelected(uci.position)
	if err != nil || backend != EvaluatorBackendNGNK4Name || got != nnueSearchScore(wantRaw, uci.position.HalfMoveClock) {
		t.Fatalf("UCI selected evaluation = %d/%q/%v", got, backend, err)
	}
	var output bytes.Buffer
	uci.handleCommand("uci", &output)
	if !strings.Contains(output.String(), "option name EvalBackend type combo default ngn-k4-768-v1") {
		t.Fatalf("startup default not advertised:\n%s", output.String())
	}

	corrupt := append([]byte(nil), data...)
	corrupt[len(corrupt)-1] ^= 1
	corruptPath := filepath.Join(t.TempDir(), "corrupt.nnue")
	if err := os.WriteFile(corruptPath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	before := uci.searcher.evaluatorModel.identity
	output.Reset()
	uci.handleCommand("setoption name EvalFile value "+corruptPath, &output)
	if !strings.HasPrefix(output.String(), "info string error eval option: ") || uci.searcher.evaluatorModel.identity != before {
		t.Fatalf("corrupt replacement was not transactional:\n%s", output.String())
	}
}
