package engine

import (
	"testing"
	"time"
)

func newSearchInfoForTest(tm *TimeManager, rootDepth int) *SearchInfo {
	var moveBuffer [256]Move
	var captureBuffer [64]Move
	var orderedBuffer [256]Move
	var seeGains [32]int
	return &SearchInfo{
		control:       &defaultSearchEngine.control,
		history:       &defaultSearchEngine.history,
		evaluator:     &hceEvaluator{},
		tt:            defaultTTForDirectTest(),
		Depth:         rootDepth,
		RootDepth:     rootDepth,
		BestMove:      EmptyMove,
		BestScore:     -INFINITY,
		TimeManager:   tm,
		MoveBuffer:    &moveBuffer,
		CaptureBuffer: &captureBuffer,
		OrderedBuffer: &orderedBuffer,
		SEEGains:      &seeGains,
		Frames:        make([]searchFrame, searchFramePoolSize),
	}
}

func TestStoppedChildDoesNotStoreParentTT(t *testing.T) {
	ClearStop()
	ClearHistoryTable()
	ClearKillerMoves()
	ClearCounterMoves()
	defaultSearchEngine, _ = NewSearchEngineWithHash(1)

	pos, err := ParseFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	rootHash := pos.Hash()

	tm := NewTimeManager()
	tm.timeControl = TimePerMove
	tm.allocatedTime = time.Nanosecond
	tm.startTime = time.Now().Add(-time.Second)
	tm.checkCounter = 1021 // parent checks miss; first child check trips the clock.

	info := newSearchInfoForTest(tm, 1)
	score := alphaBetaPV(pos, 1, 0, -INFINITY, INFINITY, true, true, false, info)
	if !info.Stopped {
		t.Fatal("search did not stop in the child as expected")
	}
	if score != stoppedSearchScore {
		t.Fatalf("stopped search returned %d, want stoppedSearchScore", score)
	}
	if _, _, _, _, hit, _ := info.tt.Get(rootHash); hit {
		t.Fatal("stopped parent wrote a transposition-table entry")
	}
}

func TestQuiescenceHonorsTimeLimit(t *testing.T) {
	ClearStop()
	t.Cleanup(ClearStop)
	defaultSearchEngine, _ = NewSearchEngineWithHash(1)
	pos, err := ParseFEN("4k3/8/8/3q4/4P3/8/8/4K3 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	hash := pos.Hash()
	tm := NewTimeManager()
	tm.timeControl = TimePerMove
	tm.allocatedTime = time.Nanosecond
	tm.startTime = time.Now().Add(-time.Second)
	tm.checkCounter = 1023 // The next time check must observe the expired clock.
	info := newSearchInfoForTest(tm, 1)
	score := quiescence(pos, -INFINITY, INFINITY, 1, info)
	if !info.Stopped || score != stoppedSearchScore {
		t.Fatalf("expired qsearch returned score=%d stopped=%v", score, info.Stopped)
	}
	if info.Nodes != 0 || pos.Hash() != hash {
		t.Fatal("expired qsearch searched moves or changed the position")
	}
}
