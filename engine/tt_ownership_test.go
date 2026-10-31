package engine

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSearchEngineHashConfigurationIsCheckedAndLazy(t *testing.T) {
	for _, mb := range []int{0, -1, MaxHashMB + 1} {
		if engine, err := NewSearchEngineWithHash(mb); err == nil || engine != nil {
			t.Fatalf("NewSearchEngineWithHash(%d) = (%v, %v), want nil,error", mb, engine, err)
		}
	}
	for _, mb := range []int{MinHashMB, MaxHashMB} {
		engine, err := NewSearchEngineWithHash(mb)
		if err != nil {
			t.Fatalf("NewSearchEngineWithHash(%d): %v", mb, err)
		}
		if engine.tt != nil {
			t.Fatalf("constructor allocated %d MB TT eagerly", mb)
		}
		if got := engine.HashSize(); got != mb {
			t.Fatalf("HashSize = %d, want %d", got, mb)
		}
	}

	engine := NewSearchEngine()
	if err := engine.ResizeHash(0); err == nil || engine.HashSize() != DEFAULT_CACHE_SIZE || engine.tt != nil {
		t.Fatalf("failed resize changed state: err=%v size=%d tt=%p", err, engine.HashSize(), engine.tt)
	}
	if err := engine.SetTTMode(TTMode(99)); err == nil || engine.tt != nil {
		t.Fatalf("invalid mode changed state: err=%v tt=%p", err, engine.tt)
	}
}

func TestSearchEnginesOwnIndependentTranspositionTables(t *testing.T) {
	a, _ := NewSearchEngineWithHash(1)
	b, _ := NewSearchEngineWithHash(1)
	moveA := NewMove(A2, A3, WhitePawn, NoPiece, NoType, 0)
	moveB := NewMove(H7, H6, BlackPawn, NoPiece, NoType, 0)
	const keyA = uint64(0x1122334455667788)
	const keyB = uint64(0x8877665544332211)
	a.TTStore(keyA, moveA, 11, 3, Exact, false)
	b.TTStore(keyB, moveB, -22, 4, LowerBound, true)

	if got, eval, depth, node, hit, pv := a.TTProbe(keyA); !hit || got != moveA || eval != 11 || depth != 3 || node != Exact || pv {
		t.Fatalf("engine A tuple = (%v,%d,%d,%v,%v,%v)", got, eval, depth, node, hit, pv)
	}
	if _, _, _, _, hit, _ := a.TTProbe(keyB); hit {
		t.Fatal("engine A observed engine B entry")
	}
	if got, eval, depth, node, hit, pv := b.TTProbe(keyB); !hit || got != moveB || eval != -22 || depth != 4 || node != LowerBound || !pv {
		t.Fatalf("engine B tuple = (%v,%d,%d,%v,%v,%v)", got, eval, depth, node, hit, pv)
	}
	if _, _, _, _, hit, _ := b.TTProbe(keyA); hit {
		t.Fatal("engine B observed engine A entry")
	}
}

func TestSynchronizedCacheReturnsOnlyCoherentRecords(t *testing.T) {
	cache := newCacheWithMode(1, TTSynchronized)
	move := NewMove(A2, A4, WhitePawn, NoPiece, NoType, 0)
	const hashA = uint64(0x12345)
	hashB := hashA ^ uint64(cache.length)
	dataA := Pack(move, 111, 9, Exact, 0, false)
	dataB := Pack(move, -222, 9, Exact, 0, false)
	// Probe absent combinations while two colliding records are updated.
	// Cluster locking must publish each full-key/payload tuple coherently.
	hybridA := hashA ^ dataA ^ dataB
	hybridB := hashB ^ dataA ^ dataB
	if cache.index(hashA) != cache.index(hashB) {
		t.Fatal("test hashes do not collide")
	}

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		for i := 0; i < 4000; i++ {
			cache.Set(hashA, move, 111, 9, Exact, false)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 4000; i++ {
			cache.Set(hashB, move, -222, 9, Exact, false)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 8000; i++ {
			if _, _, _, _, hit, _ := cache.Get(hybridA); hit {
				t.Errorf("accepted A-key/B-data torn tuple")
				return
			}
			if _, _, _, _, hit, _ := cache.Get(hybridB); hit {
				t.Errorf("accepted B-key/A-data torn tuple")
				return
			}
			if got, eval, depth, node, hit, pv := cache.Get(hashA); hit && (got != move || eval != 111 || depth != 9 || node != Exact || pv) {
				t.Errorf("incoherent A tuple: (%v,%d,%d,%v,%v)", got, eval, depth, node, pv)
				return
			}
			if got, eval, depth, node, hit, pv := cache.Get(hashB); hit && (got != move || eval != -222 || depth != 9 || node != Exact || pv) {
				t.Errorf("incoherent B tuple: (%v,%d,%d,%v,%v)", got, eval, depth, node, pv)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			cache.AdvanceAge()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = cache.Consumed()
		}
	}()
	wg.Wait()
}

func TestCacheGenerationUsesTheSameWidthAsItsPayload(t *testing.T) {
	cache := NewCache(1)
	cache.age = 255
	cache.AdvanceAge()
	if cache.age != 0 {
		t.Fatalf("generation wrap=%d", cache.age)
	}
	for age := uint16(0); age < 256; age++ {
		_, _, _, _, got, _ := Unpack(Pack(EmptyMove, 0, 0, Exact, age, false))
		if got != age {
			t.Fatalf("generation %d became %d", age, got)
		}
	}
}

func TestNewGameClearsTTAndHistoryButKeepsWarmHCECaches(t *testing.T) {
	engine, _ := NewSearchEngineWithHash(1)
	pos := controlTestPosition(t)
	_ = engine.evaluateForPlayerCached(pos)
	engine.worker.hce.full[7] = evalCacheEntry{key: 1, val: 2}
	move := NewMove(A2, A3, WhitePawn, NoPiece, NoType, 0)
	engine.worker.history.historyTable[WhitePawn][A3] = 4
	engine.worker.history.killerMoves[0][0] = move
	engine.worker.history.lastMovePlayed = move
	const key = uint64(0xfeedbeef)
	engine.TTStore(key, move, 5, 2, Exact, false)
	oldTT := engine.tt
	oldEvaluator := engine.worker.hce

	engine.NewGame()
	if engine.tt != oldTT || engine.worker.hce != oldEvaluator {
		t.Fatal("same-generation new game replaced owned allocations")
	}
	if _, _, _, _, hit, _ := engine.TTProbe(key); hit {
		t.Fatal("new game retained TT entry")
	}
	if engine.worker.history.historyTable[WhitePawn][A3] != 0 || engine.worker.history.killerMoves[0][0] != EmptyMove || engine.worker.history.lastMovePlayed != EmptyMove {
		t.Fatal("new game retained game-scoped history")
	}
	if engine.worker.hce.full[7] != (evalCacheEntry{key: 1, val: 2}) {
		t.Fatal("same-generation new game cleared warm HCE cache")
	}
}

func TestCallbackGetsCopiedPVAndHashfullWithoutReceiverReentry(t *testing.T) {
	engine, _ := NewSearchEngineWithHash(1)
	pos := controlTestPosition(t)
	var callbacks int
	var lastPV []Move
	result := engine.SearchIterativeDeepeningWithCallback(pos, 2, nil, func(info *SearchInfo) {
		callbacks++
		if info.PVLength == 0 || info.PV[0] != info.BestMove {
			t.Fatalf("callback PV length=%d first=%v best=%v", info.PVLength, info.PV[0], info.BestMove)
		}
		if info.Hashfull < 0 || info.Hashfull > 1000 {
			t.Fatalf("hashfull=%d", info.Hashfull)
		}
		lastPV = append(lastPV[:0], info.PV[:info.PVLength]...)
	})
	if callbacks != 2 {
		t.Fatalf("callbacks=%d, want 2", callbacks)
	}
	if result.PVLength == 0 || !reflect.DeepEqual(result.PV[:result.PVLength], lastPV) {
		t.Fatalf("final copied PV=%v, callback PV=%v", result.PV[:result.PVLength], lastPV)
	}

	withoutCallback := engine.SearchIterativeDeepening(controlTestPosition(t), 1, nil)
	if withoutCallback.PVLength != 0 || withoutCallback.Hashfull != 0 {
		t.Fatalf("nil callback performed reporting: pv=%d hashfull=%d", withoutCallback.PVLength, withoutCallback.Hashfull)
	}
}

func TestHashResizeWaitsForActiveReceiverSession(t *testing.T) {
	engine, _ := NewSearchEngineWithHash(1)
	entered := make(chan struct{})
	release := make(chan struct{})
	searchDone := make(chan struct{})
	resizeDone := make(chan error, 1)
	go func() {
		engine.SearchIterativeDeepeningWithCallback(controlTestPosition(t), 1, nil, func(*SearchInfo) {
			close(entered)
			<-release
		})
		close(searchDone)
	}()
	<-entered
	go func() { resizeDone <- engine.ResizeHash(2) }()
	select {
	case err := <-resizeDone:
		close(release)
		<-searchDone
		t.Fatalf("resize crossed active callback: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-searchDone
	if err := <-resizeDone; err != nil {
		t.Fatal(err)
	}
	if engine.HashSize() != 2 {
		t.Fatalf("HashSize=%d, want 2", engine.HashSize())
	}
}

func TestCachePackingPreservesAllValidMovesAndScores(t *testing.T) {
	for _, move := range []Move{EmptyMove, NewMove(A7, A8, WhitePawn, BlackRook, Queen, Capture), NewMove(E1, G1, WhiteKing, NoPiece, NoType, KingSideCastle), NewMove(E5, D6, WhitePawn, BlackPawn, NoType, Capture|EnPassant), Move(ttMoveMask)} {
		for _, score := range []int16{-32768, -30000, -1, 0, 1, 30000, 32767} {
			for _, bound := range []NodeType{Exact, LowerBound, UpperBound} {
				for _, depth := range []int8{0, 1, 64, 127} {
					for _, pv := range []bool{false, true} {
						m, e, d, n, a, p := Unpack(Pack(move, score, depth, bound, 255, pv))
						if m != move || e != score || d != depth || n != bound || a != 255 || p != pv {
							t.Fatalf("roundtrip lost tuple: %v %d %d %v %d %v", m, e, d, n, a, p)
						}
					}
				}
			}
		}
	}
	if NewCache(0) != nil || NewCache(-1) != nil {
		t.Fatal("nonpositive allocation")
	}
	cache := NewCache(1)
	if cache.length*ttRecordBytes != 1024*1024 || len(cache.items)%ttWays != 0 {
		t.Fatal("allocation budget")
	}
}

func TestIndependentSearchEnginesOverlapWithPrivateControl(t *testing.T) {
	// Establish the survivor's deterministic isolated result before overlapping it
	// with a separately-owned search that will be cancelled.
	survivorFEN := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	isolatedPos := mustHCEPosition(t, survivorFEN)
	isolatedBefore := snapshotPVPosition(isolatedPos)
	isolatedEngine, _ := NewSearchEngineWithHash(1)
	isolated := isolatedEngine.Search(isolatedPos, 3)
	assertPVPositionRestored(t, isolatedPos, isolatedBefore)

	cancelled, _ := NewSearchEngineWithHash(1)
	survivor, _ := NewSearchEngineWithHash(1)
	cancelledPos := controlTestPosition(t)
	survivorPos := mustHCEPosition(t, survivorFEN)
	cancelledBefore := snapshotPVPosition(cancelledPos)
	survivorBefore := snapshotPVPosition(survivorPos)
	entered := make(chan int, 2)
	release := make(chan struct{})
	results := make([]*SearchInfo, 2)
	engines := []*SearchEngine{cancelled, survivor}
	positions := []*Position{cancelledPos, survivorPos}
	depths := []int{MaximumDepth, 3}
	var wg sync.WaitGroup
	wg.Add(2)
	for i := range engines {
		i := i
		go func() {
			defer wg.Done()
			var once sync.Once
			results[i] = engines[i].SearchIterativeDeepeningWithCallback(positions[i], depths[i], nil, func(*SearchInfo) {
				once.Do(func() {
					entered <- i
					<-release
				})
			})
		}()
	}
	seen := [2]bool{}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for i := 0; i < 2; i++ {
		select {
		case i := <-entered:
			seen[i] = true
		case <-deadline.C:
			for _, engine := range engines {
				engine.RequestStop()
			}
			close(release)
			wg.Wait()
			t.Fatalf("independent searches did not both enter callbacks: %v", seen)
		}
	}
	if !seen[0] || !seen[1] {
		for _, engine := range engines {
			engine.RequestStop()
		}
		close(release)
		wg.Wait()
		t.Fatalf("search callbacks did not overlap: %v", seen)
	}
	cancelled.RequestStop()
	close(release)
	wg.Wait()

	if !results[0].Stopped || results[0].Depth != 1 {
		t.Fatalf("cancelled receiver stopped=%v depth=%d", results[0].Stopped, results[0].Depth)
	}
	if results[1].Stopped || results[1].Depth != isolated.Depth || results[1].BestMove != isolated.BestMove ||
		results[1].BestScore != isolated.BestScore || results[1].Nodes != isolated.Nodes {
		t.Fatalf("survivor differs from isolated: concurrent=(stop=%v d=%d move=%v score=%d nodes=%d) isolated=(d=%d move=%v score=%d nodes=%d)",
			results[1].Stopped, results[1].Depth, results[1].BestMove, results[1].BestScore, results[1].Nodes,
			isolated.Depth, isolated.BestMove, isolated.BestScore, isolated.Nodes)
	}
	assertPVPositionRestored(t, cancelledPos, cancelledBefore)
	assertPVPositionRestored(t, survivorPos, survivorBefore)
	if cancelled.tt == nil || survivor.tt == nil || cancelled.tt == survivor.tt || cancelled.StopRequested() == survivor.StopRequested() {
		t.Fatalf("receiver state crossed: cancelledTT=%p survivorTT=%p cancelledStop=%v survivorStop=%v",
			cancelled.tt, survivor.tt, cancelled.StopRequested(), survivor.StopRequested())
	}
}

func TestHashContentOperationsWaitForActiveReceiverSession(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(*SearchEngine) error
	}{
		{name: "clear", run: func(e *SearchEngine) error { e.ClearHash(); return nil }},
		{name: "diagnostics", run: func(e *SearchEngine) error { _ = e.TTDiagnostics(); return nil }},
		{name: "mode", run: func(e *SearchEngine) error { return e.SetTTMode(TTSynchronized) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine, _ := NewSearchEngineWithHash(1)
			entered := make(chan struct{})
			release := make(chan struct{})
			searchDone := make(chan struct{})
			opDone := make(chan error, 1)
			go func() {
				engine.SearchIterativeDeepeningWithCallback(controlTestPosition(t), 1, nil, func(*SearchInfo) {
					close(entered)
					<-release
				})
				close(searchDone)
			}()
			<-entered
			go func() { opDone <- test.run(engine) }()
			select {
			case err := <-opDone:
				close(release)
				<-searchDone
				t.Fatalf("operation crossed active callback: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			close(release)
			<-searchDone
			if err := <-opDone; err != nil {
				t.Fatal(err)
			}
		})
	}
}
