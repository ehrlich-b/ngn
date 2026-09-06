package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ehrlich-b/ngn/nnue"
)

func n3cPosition(t *testing.T, fen string) *Position {
	t.Helper()
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	return pos
}

func comparableN3CSearchInfo(info *SearchInfo) SearchInfo {
	copy := *info
	copy.TimeManager = nil
	copy.MoveBuffer = nil
	copy.CaptureBuffer = nil
	copy.OrderedBuffer = nil
	copy.SEEGains = nil
	copy.control = nil
	copy.history = nil
	copy.evaluator = nil
	copy.tt = nil
	copy.Frames = nil
	return copy
}

func TestSearchNGNV1IncrementalMatchesFullRefreshEveryCommittedFrame(t *testing.T) {
	model, tensors := adapterContextModel(t)
	const fen = "r3k2r/p1ppqpb1/bn2pnp1/2pP4/1p2P3/2N2N2/PPQBBPPP/R3K2R w KQkq - 37 1"

	incremental := NewSearchEngine()
	fullRefresh := NewSearchEngine()
	if err := incremental.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	if err := fullRefresh.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}

	observed := 0
	incremental.workerEvaluator.transitionObserver = func(pos *Position, context *nnue.Context) {
		observed++
		requireAdapterContextLanes(t, context, tensors, fullPositionForWorkerTest(t, pos))
	}
	fullRefresh.workerEvaluator.fullRefreshOracle = true

	incrementalPos := n3cPosition(t, fen)
	fullRefreshPos := n3cPosition(t, fen)
	incrementalBefore := snapshotPVPosition(incrementalPos)
	fullRefreshBefore := snapshotPVPosition(fullRefreshPos)
	got := incremental.SearchFixed(incrementalPos, 3, nil)
	want := fullRefresh.SearchFixed(fullRefreshPos, 3, nil)

	if observed < 2 {
		t.Fatalf("incremental lane observer ran %d times, want root and child frames", observed)
	}
	if gotComparable, wantComparable := comparableN3CSearchInfo(got), comparableN3CSearchInfo(want); !reflect.DeepEqual(gotComparable, wantComparable) {
		t.Fatalf("incremental search differs from full-refresh oracle:\nincremental=%+v\nfull-refresh=%+v", gotComparable, wantComparable)
	}
	if after := snapshotPVPosition(incrementalPos); !reflect.DeepEqual(after, incrementalBefore) {
		t.Fatal("incremental search changed root position")
	}
	if after := snapshotPVPosition(fullRefreshPos); !reflect.DeepEqual(after, fullRefreshBefore) {
		t.Fatal("full-refresh search changed root position")
	}
	if incremental.workerEvaluator.context.Depth() != 0 || fullRefresh.workerEvaluator.context.Depth() != 0 {
		t.Fatalf("context depths incremental=%d full=%d", incremental.workerEvaluator.context.Depth(), fullRefresh.workerEvaluator.context.Depth())
	}
	if incremental.evaluator != nil || fullRefresh.evaluator != nil {
		t.Fatal("NNUE-only search allocated an unused HCE cache")
	}
	requireWorkerRawMatchesFull(t, incremental.workerEvaluator, incrementalPos)
}

func TestEvaluatorSelectionIsTransactionalAndCompositeIdentityClearsState(t *testing.T) {
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	terminal := n3cPosition(t, "7k/6Q1/6K1/8/8/8/8/8 b - - 0 1")
	searcher.SearchFixed(terminal, 1, nil)

	poisonMove := NewMove(A1, A2, WhiteKing, NoPiece, NoType, 0)
	searcher.history.historyTable[WhiteKing][A2] = 1
	searcher.history.continuationHistory[WhiteKing][A1][WhiteKing][A2] = 2
	searcher.history.followupHistory[WhiteKing][A1][WhiteKing][A2] = 3
	searcher.history.pawnCorrectionHistory[White][1] = 4
	searcher.history.nonPawnCorrectionHistory[Black][2] = 5
	searcher.history.minorCorrectionHistory[White][3] = 6
	searcher.history.captureHistory[WhiteKing][A2][BlackPawn] = 7
	searcher.history.killerMoves[0][0] = poisonMove
	searcher.history.counterMoves[A1][A2] = poisonMove
	searcher.history.lastMovePlayed = poisonMove
	const hceKey = uint64(0x4e3348ce)
	searcher.TTStore(hceKey, poisonMove, 17, 3, Exact, false)
	hceIdentity := searcher.ttIdentity

	beforeModel := searcher.evaluatorModel
	beforeWorker := searcher.workerEvaluator
	if err := searcher.SelectNGNV1Evaluator(nil); !errors.Is(err, errWorkerEvaluator) {
		t.Fatalf("invalid selection error = %v", err)
	}
	if searcher.evaluatorModel != beforeModel || searcher.workerEvaluator != beforeWorker || searcher.ttIdentity != hceIdentity {
		t.Fatal("failed NNUE selection changed published receiver state")
	}

	model, _ := adapterContextModel(t)
	if err := searcher.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	if searcher.workerEvaluator == nil || searcher.workerEvaluator.context == nil {
		t.Fatal("NNUE selection did not prebuild a private worker Context")
	}
	if _, _, _, _, hit, _ := searcher.TTProbe(hceKey); hit {
		t.Fatal("backend swap retained HCE TT entry")
	}
	if searcher.ttIdentity == hceIdentity || searcher.ttIdentity.backend != evaluatorBackendNGNV1 {
		t.Fatalf("TT identity after NNUE selection = %+v, old = %+v", searcher.ttIdentity, hceIdentity)
	}

	searcher.SearchFixed(terminal.Copy(), 1, nil)
	if searcher.history != (workerHistory{}) {
		t.Fatal("backend swap did not clear the complete worker-history family")
	}
	nnueIdentity := searcher.evaluatorIdentity
	searcher.history.lastMovePlayed = poisonMove
	if err := searcher.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	searcher.SearchFixed(terminal.Copy(), 1, nil)
	if searcher.evaluatorIdentity != nnueIdentity || searcher.history.lastMovePlayed != poisonMove {
		t.Fatal("semantically identical model selection discarded same-identity warm history")
	}

	const nnueKey = uint64(0x4e3348cf)
	searcher.TTStore(nnueKey, poisonMove, 18, 3, Exact, false)
	if err := searcher.SelectHCEEvaluator(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, hit, _ := searcher.TTProbe(nnueKey); hit {
		t.Fatal("NNUE-to-HCE swap retained NNUE TT entry")
	}
	searcher.SearchFixed(terminal.Copy(), 1, nil)
	if searcher.evaluatorIdentity.backend != evaluatorBackendHCE || searcher.history != (workerHistory{}) {
		t.Fatalf("HCE re-selection state identity=%+v history-cleared=%v", searcher.evaluatorIdentity, searcher.history == (workerHistory{}))
	}

	if err := tryBeginHCEModelMutation(); err != nil {
		t.Fatal(err)
	}
	if err := searcher.SelectNGNV1Evaluator(model); !errors.Is(err, ErrHCEModelBusy) {
		abortHCEModelMutation()
		t.Fatalf("selection during mutation error = %v", err)
	}
	abortHCEModelMutation()
}

func TestNGNV1StoppedSearchUnwindsContextAndPosition(t *testing.T) {
	model, tensors := adapterContextModel(t)
	searcher := NewSearchEngine()
	if err := searcher.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	pos := n3cPosition(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	before := snapshotPVPosition(pos)
	observed := 0
	searcher.workerEvaluator.transitionObserver = func(pos *Position, context *nnue.Context) {
		observed++
		requireAdapterContextLanes(t, context, tensors, fullPositionForWorkerTest(t, pos))
	}
	searcher.SetMaxNodes(1)
	t.Cleanup(func() { searcher.SetMaxNodes(0) })
	info := searcher.SearchFixed(pos, 5, nil)
	if !info.Stopped {
		t.Fatal("fixed node limit did not stop NNUE search")
	}
	if observed < 2 {
		t.Fatalf("observer calls = %d, want root and searched child", observed)
	}
	if searcher.workerEvaluator.context.Depth() != 0 {
		t.Fatalf("stopped search left Context depth %d", searcher.workerEvaluator.context.Depth())
	}
	if after := snapshotPVPosition(pos); !reflect.DeepEqual(after, before) {
		t.Fatal("stopped NNUE search did not restore root position")
	}
	requireWorkerRawMatchesFull(t, searcher.workerEvaluator, pos)
}

func newN3CDirectSearchInfo(searcher *SearchEngine, evaluator *workerEvaluator, tt *Cache, depth int) *SearchInfo {
	var moveBuffer [256]Move
	var captureBuffer [64]Move
	var orderedBuffer [256]Move
	var seeGains [32]int
	return &SearchInfo{
		Depth:         depth,
		RootDepth:     depth,
		BestMove:      EmptyMove,
		BestScore:     -INFINITY,
		MoveBuffer:    &moveBuffer,
		CaptureBuffer: &captureBuffer,
		OrderedBuffer: &orderedBuffer,
		SEEGains:      &seeGains,
		Frames:        make([]searchFrame, searchFramePoolSize),
		control:       &searcher.control,
		maxNodes:      searcher.control.MaxNodes(),
		history:       &searcher.history,
		evaluator:     evaluator,
		tt:            tt,
	}
}

func TestNGNV1StopInsideSingularVerificationLeavesParentContext(t *testing.T) {
	model := loadEvaluatorTensors(t, new(nnue.Tensors))
	searcher, err := NewSearchEngineWithHash(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := searcher.SelectNGNV1Evaluator(model); err != nil {
		t.Fatal(err)
	}
	pos := n3cPosition(t, "4k3/8/8/8/8/8/8/R3K3 w - - 0 1")
	before := snapshotPVPosition(pos)

	searcher.sessionMu.Lock()
	defer searcher.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	tt := searcher.prepareTTGeneration(generation)
	evaluator := searcher.prepareWorkerEvaluator(pos, generation)
	depth := SINGULAR_DEPTH
	info := newN3CDirectSearchInfo(searcher, evaluator, tt, depth)
	ttMove := adapterLegalMove(t, pos, "a1a8")
	tt.Set(pos.Hash(), ttMove, scoreToTT(100, 0), int8(depth), Exact, false)

	injected := false
	evaluator.transitionObserver = func(_ *Position, _ *nnue.Context) {
		if info.ExcludedMove != EmptyMove && !injected {
			injected = true
			info.control.RequestStop()
		}
	}
	t.Cleanup(info.control.ClearStop)
	score := alphaBetaPV(pos, depth, 0, -INFINITY, INFINITY, true, true, false, info)
	if !injected || info.SingularTries == 0 {
		t.Fatalf("singular stop witness injected=%v tries=%d", injected, info.SingularTries)
	}
	if !info.Stopped || score != stoppedSearchScore {
		t.Fatalf("singular stopped result score=%d stopped=%v", score, info.Stopped)
	}
	if evaluator.context.Depth() != 0 {
		t.Fatalf("stopped singular verification left Context depth %d", evaluator.context.Depth())
	}
	if after := snapshotPVPosition(pos); !reflect.DeepEqual(after, before) {
		t.Fatal("stopped singular verification did not leave parent position restored")
	}
	requireWorkerRawMatchesFull(t, evaluator, pos)
}
