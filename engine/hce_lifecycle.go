package engine

import (
	"errors"
	"sync"
)

// ErrHCEModelBusy reports that an operation would cross an active HCE model
// lifetime. Apply and the checked tuner entry points return it; legacy search,
// evaluation, and tuner APIs fail fast by panicking with this value because
// their established signatures cannot return an error.
var ErrHCEModelBusy = errors.New("HCE model is busy")

// The lifecycle covers supported search, direct-evaluation, ApplyTexelModel,
// and checked tuner entry points. Raw ParseFEN/Bitboard make-unmake operations
// still read the process PST tables without a lease, and UCI Hash reconfiguration
// still replaces the package TT outside this lifecycle. Callers must not overlap
// those raw/configuration operations with Apply or tuning; receiver-owned TT and
// broader configuration ownership are later stages.

var hceModelLifecycle = struct {
	mu         sync.Mutex
	activeUses int
	mutating   bool
	generation uint64
}{generation: 1}

// acquireHCEModelUse admits a search or direct evaluation without retaining the
// lifecycle mutex. Concurrent readers are allowed. A tuner mutation never
// causes an unbounded wait: APIs without an error result fail fast instead.
func acquireHCEModelUse() (uint64, error) {
	hceModelLifecycle.mu.Lock()
	defer hceModelLifecycle.mu.Unlock()
	if hceModelLifecycle.mutating {
		return 0, ErrHCEModelBusy
	}
	hceModelLifecycle.activeUses++
	return hceModelLifecycle.generation, nil
}

func mustAcquireHCEModelUse() uint64 {
	generation, err := acquireHCEModelUse()
	if err != nil {
		panic(err)
	}
	return generation
}

func releaseHCEModelUse() {
	hceModelLifecycle.mu.Lock()
	hceModelLifecycle.activeUses--
	if hceModelLifecycle.activeUses < 0 {
		hceModelLifecycle.mu.Unlock()
		panic("unbalanced HCE model lease")
	}
	hceModelLifecycle.mu.Unlock()
}

// tryBeginHCEModelMutation reserves the process-wide HCE parameter/PST state.
// It never waits for a search, direct evaluation, Apply, or tuner callback.
func tryBeginHCEModelMutation() error {
	hceModelLifecycle.mu.Lock()
	defer hceModelLifecycle.mu.Unlock()
	if hceModelLifecycle.mutating || hceModelLifecycle.activeUses != 0 {
		return ErrHCEModelBusy
	}
	hceModelLifecycle.mutating = true
	return nil
}

// publishHCEModelMutation clears evaluator-sensitive shared TT state exactly
// once, then publishes the new generation last. Per-engine state is invalidated
// lazily when that receiver next observes the generation.
func publishHCEModelMutation() {
	hceModelLifecycle.mu.Lock()
	defer hceModelLifecycle.mu.Unlock()
	if !hceModelLifecycle.mutating {
		panic("publishing HCE model without mutation lease")
	}
	if TranspositionTable != nil {
		TranspositionTable.Clear()
	}
	hceModelLifecycle.generation++
	hceModelLifecycle.mutating = false
}

// abortHCEModelMutation releases a reservation that made no global changes.
func abortHCEModelMutation() {
	hceModelLifecycle.mu.Lock()
	defer hceModelLifecycle.mu.Unlock()
	if !hceModelLifecycle.mutating {
		panic("aborting HCE model without mutation lease")
	}
	hceModelLifecycle.mutating = false
}
