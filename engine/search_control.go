package engine

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// SearchControl owns cancellation and the configured node limit for one
// SearchEngine. It is deliberately independent of clocks, TT, histories, and
// evaluator state; those remain separate ownership stages.
type SearchControl struct {
	stopRequested atomic.Bool
	maxNodes      atomic.Uint64
}

func (c *SearchControl) RequestStop()         { c.stopRequested.Store(true) }
func (c *SearchControl) ClearStop()           { c.stopRequested.Store(false) }
func (c *SearchControl) StopRequested() bool  { return c.stopRequested.Load() }
func (c *SearchControl) SetMaxNodes(n uint64) { c.maxNodes.Store(n) }
func (c *SearchControl) MaxNodes() uint64     { return c.maxNodes.Load() }

// SearchEngine owns one search control, persistent worker-history state, HCE
// caches, and transposition table. It must not be copied after first use.
// Receiver lifecycle methods serialize through the final callback and are
// intentionally non-reentrant there. Different receivers own independent
// mutable state and may search concurrently under one immutable HCE generation.
type SearchEngine struct {
	sessionMu    sync.Mutex
	control      SearchControl
	history      workerHistory
	evaluator    *hceEvaluator
	tt           *Cache
	hashMB       int
	ttMode       TTMode
	ttGeneration uint64
}

const (
	MinHashMB = 1
	MaxHashMB = 1024
)

func NewSearchEngine() *SearchEngine {
	e, _ := NewSearchEngineWithHash(DEFAULT_CACHE_SIZE)
	return e
}

// NewSearchEngineWithHash validates the UCI-supported Hash range. Allocation is
// lazy so an unused compatibility receiver does not reserve a second large TT.
func NewSearchEngineWithHash(megabytes int) (*SearchEngine, error) {
	if err := validateHashMB(megabytes); err != nil {
		return nil, err
	}
	return &SearchEngine{hashMB: megabytes, ttMode: TTDirect}, nil
}

func validateHashMB(megabytes int) error {
	if megabytes < MinHashMB || megabytes > MaxHashMB {
		return fmt.Errorf("Hash must be between %d and %d MB", MinHashMB, MaxHashMB)
	}
	return nil
}

func validTTMode(mode TTMode) bool { return mode == TTDirect || mode == TTSynchronized }

// prepareTTGeneration runs under sessionMu and an HCE model-use lease. A model
// publication invalidates this receiver lazily before any probe, store, or
// diagnostic can observe stale evaluator-dependent entries.
func (e *SearchEngine) prepareTTGeneration(generation uint64) *Cache {
	if e.hashMB == 0 {
		e.hashMB = DEFAULT_CACHE_SIZE
	}
	if e.tt == nil {
		e.tt = newCacheWithMode(e.hashMB, e.ttMode)
		e.ttGeneration = generation
	} else if e.ttGeneration != generation {
		e.tt.Clear()
		e.ttGeneration = generation
	}
	return e.tt
}

// ResizeHash changes this receiver's table while it is idle. The old table is
// discarded only after the replacement allocation succeeds.
func (e *SearchEngine) ResizeHash(megabytes int) error {
	if err := validateHashMB(megabytes); err != nil {
		return err
	}
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	replacement := newCacheWithMode(megabytes, e.ttMode)
	e.tt = replacement
	e.hashMB = megabytes
	e.ttGeneration = generation
	return nil
}

func (e *SearchEngine) HashSize() int {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	if e.hashMB == 0 {
		return DEFAULT_CACHE_SIZE
	}
	return e.hashMB
}

// SetTTMode is an idle configuration operation. Changing mode starts with an
// empty table; no helper search is started by this ownership slice.
func (e *SearchEngine) SetTTMode(mode TTMode) error {
	if !validTTMode(mode) {
		return fmt.Errorf("invalid TT mode %d", mode)
	}
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	if e.ttMode == mode {
		e.prepareTTGeneration(generation)
		return nil
	}
	e.ttMode = mode
	if e.hashMB == 0 {
		e.hashMB = DEFAULT_CACHE_SIZE
	}
	e.tt = newCacheWithMode(e.hashMB, mode)
	e.ttGeneration = generation
	return nil
}

type TTDiagnostics struct {
	SizeMB     int
	Hashfull   int
	Generation uint64
	Mode       TTMode
}

func (e *SearchEngine) TTDiagnostics() TTDiagnostics {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	tt := e.prepareTTGeneration(generation)
	return TTDiagnostics{SizeMB: e.hashMB, Hashfull: tt.Consumed(), Generation: generation, Mode: e.ttMode}
}

func (e *SearchEngine) AdvanceHashAge() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareTTGeneration(generation).AdvanceAge()
}

func (e *SearchEngine) ClearHash() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareTTGeneration(generation).Clear()
}

func (e *SearchEngine) TTProbe(hash uint64) (Move, int16, int8, NodeType, bool, bool) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	return e.prepareTTGeneration(generation).Get(hash)
}

func (e *SearchEngine) TTStore(hash uint64, move Move, eval int16, depth int8, nodeType NodeType, ttPv bool) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareTTGeneration(generation).Set(hash, move, eval, depth, nodeType, ttPv)
}

// NewGame clears game-scoped search state while retaining same-generation HCE
// full and pawn caches. A model mismatch still invalidates all evaluator and
// heuristic state through prepareHCEGeneration.
func (e *SearchEngine) NewGame() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareTTGeneration(generation).Clear()
	e.prepareHCEGeneration(nil, generation)
	e.history.ClearHistoryTable()
	e.history.ClearKillerMoves()
	e.history.ClearCounterMoves()
	e.history.SetLastMovePlayed(EmptyMove)
}

func (e *SearchEngine) RequestStop()         { e.control.RequestStop() }
func (e *SearchEngine) ClearStop()           { e.control.ClearStop() }
func (e *SearchEngine) StopRequested() bool  { return e.control.StopRequested() }
func (e *SearchEngine) SetMaxNodes(n uint64) { e.control.SetMaxNodes(n) }
func (e *SearchEngine) MaxNodes() uint64     { return e.control.MaxNodes() }

// prepareHCEGeneration runs only while sessionMu and a process-model use lease
// are held. A changed generation invalidates every worker-private heuristic and
// cache. Root accumulators are rebuilt unconditionally because any Position may
// have been created under an older process-wide PST generation.
func (e *SearchEngine) prepareHCEGeneration(pos *Position, generation uint64) *hceEvaluator {
	if e.evaluator == nil {
		// A new receiver has no stale private state to clear. Recording the current
		// generation avoids touching its large zeroed tables on first admission.
		e.evaluator = &hceEvaluator{seenGeneration: generation}
	} else if e.evaluator.seenGeneration != generation {
		e.evaluator.clearForGeneration(generation)
		e.history = workerHistory{}
	}
	if pos != nil {
		pos.Board.recomputeAccumulator()
	}
	return e.evaluator
}

func (e *SearchEngine) ClearHistoryTable() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareHCEGeneration(nil, generation)
	e.history.ClearHistoryTable()
}
func (e *SearchEngine) ClearKillerMoves() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareHCEGeneration(nil, generation)
	e.history.ClearKillerMoves()
}
func (e *SearchEngine) ClearCounterMoves() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareHCEGeneration(nil, generation)
	e.history.ClearCounterMoves()
}
func (e *SearchEngine) SetLastMovePlayed(move Move) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareHCEGeneration(nil, generation)
	e.history.SetLastMovePlayed(move)
}
func (e *SearchEngine) GetLastMovePlayed() Move {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	e.prepareHCEGeneration(nil, generation)
	return e.history.GetLastMovePlayed()
}

func (e *SearchEngine) evaluateForPlayerCached(pos *Position) int {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	evaluator := e.prepareHCEGeneration(pos, generation)
	return evaluator.SearchSTM(pos)
}

func (e *SearchEngine) Search(pos *Position, depth int) *SearchInfo {
	return e.SearchIterativeDeepening(pos, depth, nil)
}

func (e *SearchEngine) SearchIterativeDeepening(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
	return e.SearchIterativeDeepeningWithCallback(pos, maxDepth, timeManager, nil)
}

func (e *SearchEngine) SearchIterativeDeepeningWithCallback(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.control.ClearStop()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	tt := e.prepareTTGeneration(generation)
	evaluator := e.prepareHCEGeneration(pos, generation)
	return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager, callback, &e.control, &e.history, evaluator, tt)
}

// searchIterativeDeepeningPrepared is used by UCI after it has cleared the
// control and published searching=true. It must not clear again: a stop may
// arrive after go launches the goroutine but before the goroutine acquires this
// session lock.
func (e *SearchEngine) searchIterativeDeepeningPrepared(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	tt := e.prepareTTGeneration(generation)
	evaluator := e.prepareHCEGeneration(pos, generation)
	return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager, callback, &e.control, &e.history, evaluator, tt)
}

func (e *SearchEngine) SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.control.ClearStop()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	tt := e.prepareTTGeneration(generation)
	evaluator := e.prepareHCEGeneration(pos, generation)
	return searchFixedUnsafe(pos, depth, timeManager, &e.control, &e.history, evaluator, tt)
}

// ExtractPVFromTT returns a generation-coherent snapshot outside a search
// callback. Callbacks must consume SearchInfo.PV because the receiver session is
// intentionally held while callbacks run.
func (e *SearchEngine) ExtractPVFromTT(pos *Position, firstMove Move, maxLength int) []Move {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	return extractPVFromTT(e.prepareTTGeneration(generation), pos, firstMove, maxLength)
}

// Compatibility wrappers retain the package API and its single default search
// lifetime. New code that needs independent control should own a SearchEngine.
var defaultSearchEngine = NewSearchEngine()

func RequestStop()          { defaultSearchEngine.RequestStop() }
func ClearStop()            { defaultSearchEngine.ClearStop() }
func IsStopRequested() bool { return defaultSearchEngine.StopRequested() }
func SetMaxNodes(n uint64)  { defaultSearchEngine.SetMaxNodes(n) }
func MaxNodes() uint64      { return defaultSearchEngine.MaxNodes() }
func Search(pos *Position, depth int) *SearchInfo {
	return defaultSearchEngine.Search(pos, depth)
}
func SearchIterativeDeepening(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
	return defaultSearchEngine.SearchIterativeDeepening(pos, maxDepth, timeManager)
}
func SearchIterativeDeepeningWithCallback(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	return defaultSearchEngine.SearchIterativeDeepeningWithCallback(pos, maxDepth, timeManager, callback)
}
func SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	return defaultSearchEngine.SearchFixed(pos, depth, timeManager)
}

func ResizeHash(megabytes int) error          { return defaultSearchEngine.ResizeHash(megabytes) }
func ClearHash()                              { defaultSearchEngine.ClearHash() }
func AdvanceHashAge()                         { defaultSearchEngine.AdvanceHashAge() }
func HashSize() int                           { return defaultSearchEngine.HashSize() }
func TranspositionDiagnostics() TTDiagnostics { return defaultSearchEngine.TTDiagnostics() }
func StoreTransposition(hash uint64, move Move, eval int16, depth int8, nodeType NodeType, ttPv bool) {
	defaultSearchEngine.TTStore(hash, move, eval, depth, nodeType, ttPv)
}
func ProbeTransposition(hash uint64) (Move, int16, int8, NodeType, bool, bool) {
	return defaultSearchEngine.TTProbe(hash)
}
func ExtractPVFromTT(pos *Position, firstMove Move, maxLength int) []Move {
	return defaultSearchEngine.ExtractPVFromTT(pos, firstMove, maxLength)
}
