package engine

import (
	"errors"
	"fmt"

	"github.com/ehrlich-b/ngn/nnue"
)

const (
	nnueStaticEvalLimit      int64  = 25000
	evaluatorAdapterRevision uint32 = 1
)

var errWorkerEvaluator = errors.New("invalid worker evaluator state")

type evaluatorBackend uint8

const (
	evaluatorBackendHCE evaluatorBackend = iota
	evaluatorBackendNGNV1
)

// evaluatorModelIdentity is comparable and covers every input that can change
// a search score or the hidden HCE accumulator maintenance performed by board
// make/unmake. The HCE generation remains part of an NNUE identity until those
// process-global PST accumulators are removed.
type evaluatorModelIdentity struct {
	backend         evaluatorBackend
	adapterRevision uint32
	hceGeneration   uint64
	nnueMetadata    nnue.Metadata
}

// evaluatorModel is a cold immutable selection. Its NGN-v1 pointer is safe to
// share because a validated nnue.Model is immutable; mutable Context state is
// constructed separately for every worker.
type evaluatorModel struct {
	identity evaluatorModelIdentity
	ngnV1    *nnue.Model
}

func hceEvaluatorModel(hceGeneration uint64) evaluatorModel {
	return evaluatorModel{identity: evaluatorModelIdentity{
		backend:         evaluatorBackendHCE,
		adapterRevision: evaluatorAdapterRevision,
		hceGeneration:   hceGeneration,
	}}
}

func ngnV1EvaluatorModel(model *nnue.Model, hceGeneration uint64) (evaluatorModel, error) {
	context, err := nnue.NewContext(model)
	if err != nil {
		return evaluatorModel{}, fmt.Errorf("%w: %v", errWorkerEvaluator, err)
	}
	_ = context // Validation only; each worker constructs its own Context below.
	return evaluatorModel{
		identity: evaluatorModelIdentity{
			backend:         evaluatorBackendNGNV1,
			adapterRevision: evaluatorAdapterRevision,
			hceGeneration:   hceGeneration,
			nnueMetadata:    model.Metadata(),
		},
		ngnV1: model,
	}, nil
}

// workerEvaluator is a hot concrete tagged union. Backend selection happens
// once per score/transition call; NNUE's hidden-unit loops remain concrete and
// contain no interface dispatch. It owns all mutable NNUE state for one worker.
type workerEvaluator struct {
	model      evaluatorModel
	hce        *hceEvaluator
	context    *nnue.Context
	rootBuffer nnueRootBuffer
}

func (m evaluatorModel) newWorker(hce *hceEvaluator) (*workerEvaluator, error) {
	worker := &workerEvaluator{model: m}
	switch m.identity.backend {
	case evaluatorBackendHCE:
		if hce == nil {
			return nil, fmt.Errorf("%w: nil HCE evaluator", errWorkerEvaluator)
		}
		worker.hce = hce
	case evaluatorBackendNGNV1:
		context, err := nnue.NewContext(m.ngnV1)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errWorkerEvaluator, err)
		}
		worker.context = context
	default:
		return nil, fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, m.identity.backend)
	}
	return worker, nil
}

func (e *workerEvaluator) Identity() evaluatorModelIdentity {
	if e == nil {
		return evaluatorModelIdentity{}
	}
	return e.model.identity
}

// Reset binds this worker's mutable context to the admitted search root. HCE's
// root PST refresh remains owned by search admission and is intentionally not
// duplicated here.
func (e *workerEvaluator) Reset(pos *Position) error {
	if e == nil || pos == nil {
		return fmt.Errorf("%w: nil evaluator or root position", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		if e.hce == nil {
			return fmt.Errorf("%w: nil HCE evaluator", errWorkerEvaluator)
		}
		return nil
	case evaluatorBackendNGNV1:
		if e.context == nil {
			return fmt.Errorf("%w: nil NNUE context", errWorkerEvaluator)
		}
		root, err := e.rootBuffer.position(pos)
		if err != nil {
			return err
		}
		if err := e.context.Reset(root); err != nil {
			return fmt.Errorf("%w: reset: %v", errWorkerEvaluator, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

type workerEvalMove struct {
	delta  nnue.Delta
	active bool
}

type workerEvalNull struct {
	before       nnue.PositionFacts
	beforePieces nnuePieceBitboards
	beforeSide   nnue.Color
	active       bool
}

// PrepareMove captures an NNUE transition from the actual pre-move position.
// Search may discard the token for an illegal move without touching Context.
func (e *workerEvaluator) PrepareMove(pos *Position, move Move) (workerEvalMove, error) {
	if e == nil {
		return workerEvalMove{}, fmt.Errorf("%w: nil evaluator", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		return workerEvalMove{}, nil
	case evaluatorBackendNGNV1:
		delta, err := nnueMoveDelta(pos, move)
		if err != nil {
			return workerEvalMove{}, err
		}
		return workerEvalMove{delta: delta, active: true}, nil
	default:
		return workerEvalMove{}, fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

// PushMove verifies the engine's real post-move facts and side before committing
// the prepared NNUE frame. Failures are transactional in Context.
func (e *workerEvaluator) PushMove(pos *Position, transition workerEvalMove) error {
	if e == nil {
		return fmt.Errorf("%w: nil evaluator", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		return nil
	case evaluatorBackendNGNV1:
		if !transition.active || e.context == nil {
			return fmt.Errorf("%w: missing NNUE move transition", errWorkerEvaluator)
		}
		if err := validateNNUEMoveAfter(pos, transition.delta); err != nil {
			return err
		}
		if err := e.context.PushDelta(transition.delta); err != nil {
			return fmt.Errorf("%w: push move: %v", errWorkerEvaluator, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

func (e *workerEvaluator) PrepareNull(pos *Position) (workerEvalNull, error) {
	if e == nil {
		return workerEvalNull{}, fmt.Errorf("%w: nil evaluator", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		return workerEvalNull{}, nil
	case evaluatorBackendNGNV1:
		before, err := nnuePositionFacts(pos)
		if err != nil {
			return workerEvalNull{}, err
		}
		beforePieces, err := nnuePositionPieceBitboards(pos)
		if err != nil {
			return workerEvalNull{}, err
		}
		return workerEvalNull{
			before:       before,
			beforePieces: beforePieces,
			beforeSide:   engineColorForNNUE(pos.Turn()),
			active:       true,
		}, nil
	default:
		return workerEvalNull{}, fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

// PushNull validates the actual engine side flip and unchanged board facts before
// committing the alias-equivalent NNUE null frame.
func (e *workerEvaluator) PushNull(pos *Position, transition workerEvalNull) error {
	if e == nil {
		return fmt.Errorf("%w: nil evaluator", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		return nil
	case evaluatorBackendNGNV1:
		if !transition.active || e.context == nil {
			return fmt.Errorf("%w: missing NNUE null transition", errWorkerEvaluator)
		}
		if err := validateNNUENullAfter(pos, transition.before, transition.beforePieces, transition.beforeSide); err != nil {
			return err
		}
		if err := e.context.PushNullFacts(transition.before); err != nil {
			return fmt.Errorf("%w: push null: %v", errWorkerEvaluator, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

// Pop must run before the matching engine Position unmake. HCE owns no
// move-indexed evaluator state, so its branch is deliberately empty.
func (e *workerEvaluator) Pop() error {
	if e == nil {
		return fmt.Errorf("%w: nil evaluator", errWorkerEvaluator)
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		return nil
	case evaluatorBackendNGNV1:
		if e.context == nil {
			return fmt.Errorf("%w: nil NNUE context", errWorkerEvaluator)
		}
		if err := e.context.Pop(); err != nil {
			return fmt.Errorf("%w: pop: %v", errWorkerEvaluator, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend)
	}
}

// SearchSTM is the ordinary static-evaluation route. HCE delegates to the exact
// M2 policy. NGN-v1 already returns side-to-move centipawns: attenuate rule-50
// optimism once in int64, clamp to the non-mate band, then convert to int.
func (e *workerEvaluator) SearchSTM(pos *Position) int {
	if e == nil || pos == nil {
		panic(fmt.Errorf("%w: nil evaluator or position", errWorkerEvaluator))
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		if e.hce == nil {
			panic(fmt.Errorf("%w: nil HCE evaluator", errWorkerEvaluator))
		}
		return e.hce.SearchSTM(pos)
	case evaluatorBackendNGNV1:
		raw, err := e.context.Evaluate()
		if err != nil {
			panic(fmt.Errorf("%w: incremental evaluate: %v", errWorkerEvaluator, err))
		}
		return nnueSearchScore(raw, pos.HalfMoveClock)
	default:
		panic(fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend))
	}
}

// LegacyUndampedSTM preserves the three alphaBeta emergency-return semantics.
// NGN-v1 deliberately performs a fresh full-model evaluation of the actual
// engine board, skips rule-50 attenuation, and clamps before int conversion.
func (e *workerEvaluator) LegacyUndampedSTM(pos *Position) int {
	if e == nil || pos == nil {
		panic(fmt.Errorf("%w: nil evaluator or position", errWorkerEvaluator))
	}
	switch e.model.identity.backend {
	case evaluatorBackendHCE:
		if e.hce == nil {
			panic(fmt.Errorf("%w: nil HCE evaluator", errWorkerEvaluator))
		}
		return e.hce.LegacyUndampedSTM(pos)
	case evaluatorBackendNGNV1:
		full, err := e.rootBuffer.position(pos)
		if err != nil {
			panic(err)
		}
		raw, err := e.model.ngnV1.Evaluate(full)
		if err != nil {
			panic(fmt.Errorf("%w: fresh evaluate: %v", errWorkerEvaluator, err))
		}
		return clampNNUEStatic(raw)
	default:
		panic(fmt.Errorf("%w: unknown backend %d", errWorkerEvaluator, e.model.identity.backend))
	}
}

func nnueSearchScore(raw int64, halfMoveClock uint8) int {
	const budget = int64(FiftyMoveDampBudget)
	factor := budget - int64(halfMoveClock)
	// Quotient/remainder form is exactly raw*factor/budget with truncation toward
	// zero, while remaining safe for every int64 raw score.
	attenuated := (raw/budget)*factor + (raw%budget)*factor/budget
	return clampNNUEStatic(attenuated)
}

func clampNNUEStatic(raw int64) int {
	if raw > nnueStaticEvalLimit {
		return int(nnueStaticEvalLimit)
	}
	if raw < -nnueStaticEvalLimit {
		return -int(nnueStaticEvalLimit)
	}
	return int(raw)
}
