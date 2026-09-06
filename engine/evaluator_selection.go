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
		hce = e.worker.prepareHCEGeneration(pos, generation)
	} else {
		// NNUE still needs the admitted process PST generation and an unconditional
		// root accumulator rebuild, but it does not allocate the 16 MiB HCE cache.
		if e.worker.hce != nil && e.worker.hce.seenGeneration != generation {
			e.worker.hce.clearForGeneration(generation)
			e.worker.history = workerHistory{}
		}
		if pos != nil {
			pos.Board.recomputeAccumulator()
		}
	}
	identity := model.identity
	if e.worker.evaluatorIdentity != identity {
		// A never-admitted zero-value/default-HCE receiver has no stale model
		// state. Preserve history seeded through the legacy same-model setup path.
		// An explicit selection prebuilds workerEvaluator, so its first admission
		// still clears any state belonging to the prior/default backend.
		if e.worker.evaluator != nil || e.worker.evaluatorIdentity.adapterRevision != 0 {
			e.worker.history = workerHistory{}
		}
		e.worker.evaluatorIdentity = identity
	}
	if e.worker.evaluator == nil || e.worker.evaluator.Identity() != identity {
		worker, err := model.newWorker(hce)
		if err != nil {
			panic(fmt.Errorf("%w: construct admitted worker: %v", errWorkerEvaluator, err))
		}
		e.worker.evaluator = worker
	}
	if pos != nil {
		if err := e.worker.evaluator.Reset(pos); err != nil {
			panic(fmt.Errorf("%w: reset admitted worker: %v", errWorkerEvaluator, err))
		}
	}
	return e.worker.evaluator
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

	hce := e.worker.hce
	if hce == nil {
		hce = &hceEvaluator{seenGeneration: generation}
	}
	model := hceEvaluatorModel(generation)
	worker, err := model.newWorker(hce)
	if err != nil {
		return err
	}

	// Publish only after every fallible validation/allocation has succeeded.
	e.worker.hce = hce
	e.evaluatorModel = model
	e.worker.evaluator = worker
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
	e.worker.evaluator = worker
	return nil
}

const (
	EvaluatorBackendHCEName   = "hce"
	EvaluatorBackendNGNV1Name = "ngn-v1"
)

func evaluatorBackendName(backend evaluatorBackend) string {
	switch backend {
	case evaluatorBackendHCE:
		return EvaluatorBackendHCEName
	case evaluatorBackendNGNV1:
		return EvaluatorBackendNGNV1Name
	default:
		return fmt.Sprintf("unknown-%d", backend)
	}
}

// SelectedEvaluatorBackend reports the receiver's immutable evaluator selection.
// It is session serialized and participates in the process HCE/PST read lease.
func (e *SearchEngine) SelectedEvaluatorBackend() string {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation := mustAcquireHCEModelUse()
	defer releaseHCEModelUse()
	return evaluatorBackendName(e.selectedEvaluatorModel(generation).identity.backend)
}

// EvaluateSelected reports the selected backend's base SearchSTM score for pos:
// side-to-move centipawns with the backend's declared rule-50/score adapter, and
// without search correction-history terms.
func (e *SearchEngine) EvaluateSelected(pos *Position) (score int, backend string, err error) {
	if pos == nil {
		return 0, "", fmt.Errorf("%w: nil evaluation position", errWorkerEvaluator)
	}
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	generation, err := acquireHCEModelUse()
	if err != nil {
		return 0, "", err
	}
	defer releaseHCEModelUse()
	worker := e.prepareWorkerEvaluator(pos, generation)
	return worker.SearchSTM(pos), evaluatorBackendName(worker.Identity().backend), nil
}
