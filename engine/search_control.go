package engine

import (
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

// SearchEngine owns one search control and one persistent worker-history state.
// It must not be copied after first use.
// Searches through one instance are serialized from entry through the final callback.
// Searches through different instances are not yet safe to overlap because TT, evaluator
// caches, parameters, and other resources remain package-owned.
type SearchEngine struct {
	sessionMu sync.Mutex
	control   SearchControl
	history   workerHistory
}

func NewSearchEngine() *SearchEngine { return &SearchEngine{} }

func (e *SearchEngine) RequestStop()         { e.control.RequestStop() }
func (e *SearchEngine) ClearStop()           { e.control.ClearStop() }
func (e *SearchEngine) StopRequested() bool  { return e.control.StopRequested() }
func (e *SearchEngine) SetMaxNodes(n uint64) { e.control.SetMaxNodes(n) }
func (e *SearchEngine) MaxNodes() uint64     { return e.control.MaxNodes() }

func (e *SearchEngine) ClearHistoryTable() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.history.ClearHistoryTable()
}
func (e *SearchEngine) ClearKillerMoves() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.history.ClearKillerMoves()
}
func (e *SearchEngine) ClearCounterMoves() {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.history.ClearCounterMoves()
}
func (e *SearchEngine) SetLastMovePlayed(move Move) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.history.SetLastMovePlayed(move)
}
func (e *SearchEngine) GetLastMovePlayed() Move {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	return e.history.GetLastMovePlayed()
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
	return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager, callback, &e.control, &e.history)
}

// searchIterativeDeepeningPrepared is used by UCI after it has cleared the
// control and published searching=true. It must not clear again: a stop may
// arrive after go launches the goroutine but before the goroutine acquires this
// session lock.
func (e *SearchEngine) searchIterativeDeepeningPrepared(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager, callback, &e.control, &e.history)
}

func (e *SearchEngine) SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	e.control.ClearStop()
	return searchFixedUnsafe(pos, depth, timeManager, &e.control, &e.history)
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
