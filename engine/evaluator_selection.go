package engine

import (
	"fmt"

	"github.com/ehrlich-b/ngn/nnue"
)

// selectedEvaluatorModel returns the receiver's immutable evaluator selection
// with the process HCE/PST generation admitted for this operation. A zero-value
// SearchEngine preserves the historical HCE default.
func (e *SearchEngine) selectedEvaluatorModel(generation uint64) evaluatorModel {
	model := e.evaluatorModel
	if model.identity.adapterRevision == 0 {
		return hceEvaluatorModel(generation)
	}
	model.identity.hceGeneration = generation
	return model
}

// prepareWorkerEvaluator runs under sessionMu and an HCE model-use lease.
// Composite identity changes invalidate every evaluator-sensitive private
// heuristic. Root PST accumulators are rebuilt by prepareHCEGeneration for both
// backends because board make/unmake still uses process-global HCE tables.
func (e *SearchEngine) prepareWorkerEvaluator(pos *Position, generation uint64) *workerEvaluator {
	model := e.selectedEvaluatorModel(generation)
	var hce *hceEvaluator
	if model.identity.backend == evaluatorBackendHCE {
		hce = e.prepareHCEGeneration(pos, generation)
	} else {
		// NNUE still needs the admitted process PST generation and an unconditional
		// root accumulator rebuild, but it does not allocate the 16 MiB HCE cache.
		if e.evaluator != nil && e.evaluator.seenGeneration != generation {
			e.evaluator.clearForGeneration(generation)
			e.history = workerHistory{}
		}
		if pos != nil {
			pos.Board.recomputeAccumulator()
		}
	}
	identity := model.identity
	if e.evaluatorIdentity != identity {
		// A never-admitted zero-value/default-HCE receiver has no stale model
		// state. Preserve history seeded through the legacy same-model setup path.
		// An explicit selection prebuilds workerEvaluator, so its first admission
		// still clears any state belonging to the prior/default backend.
		if e.workerEvaluator != nil || e.evaluatorIdentity.adapterRevision != 0 {
			e.history = workerHistory{}
		}
		e.evaluatorIdentity = identity
	}
	if e.workerEvaluator == nil || e.workerEvaluator.Identity() != identity {
		worker, err := model.newWorker(hce)
		if err != nil {
			panic(fmt.Errorf("%w: construct admitted worker: %v", errWorkerEvaluator, err))
		}
		e.workerEvaluator = worker
	}
	if pos != nil {
		if err := e.workerEvaluator.Reset(pos); err != nil {
			panic(fmt.Errorf("%w: reset admitted worker: %v", errWorkerEvaluator, err))
		}
	}
	return e.workerEvaluator
}

// SelectHCEEvaluator transactionally selects the historical HCE backend for
// this receiver. It is idle/session serialized and returns ErrHCEModelBusy
// rather than waiting if a process-wide HCE mutation is active.
func (e *SearchEngine) SelectHCEEvaluator() error {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation, err := acquireHCEModelUse()
	if err != nil {
		return err
	}
	defer releaseHCEModelUse()

	hce := e.evaluator
	if hce == nil {
		hce = &hceEvaluator{seenGeneration: generation}
	}
	model := hceEvaluatorModel(generation)
	worker, err := model.newWorker(hce)
	if err != nil {
		return err
	}

	// Publish only after every fallible validation/allocation has succeeded.
	e.evaluator = hce
	e.evaluatorModel = model
	e.workerEvaluator = worker
	return nil
}

// SelectNGNV1Evaluator transactionally selects one exact validated NGN-v1
// model for this receiver. The immutable model may be shared; the candidate
// worker Context is built privately before publication. Arbitrary Stockfish
// .nnue formats are outside this API and are rejected by nnue.Load.
func (e *SearchEngine) SelectNGNV1Evaluator(model *nnue.Model) error {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation, err := acquireHCEModelUse()
	if err != nil {
		return err
	}
	defer releaseHCEModelUse()

	candidate, err := ngnV1EvaluatorModel(model, generation)
	if err != nil {
		return err
	}
	worker, err := candidate.newWorker(nil)
	if err != nil {
		return err
	}

	// Publication is one receiver-local critical section. TT/history invalidation
	// occurs lazily before the next receiver operation observes this identity.
	e.evaluatorModel = candidate
	e.workerEvaluator = worker
	return nil
}
