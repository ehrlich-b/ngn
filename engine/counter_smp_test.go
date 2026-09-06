package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/ehrlich-b/ngn/countereval"
)

func TestLazySMPCounterContextsStayPrivateBalancedAndReusable(t *testing.T) {
	model := loadVaryingCounterModel(t)
	searcher := newSMPTestEngine(t, 3)
	if err := searcher.SelectCounter55Evaluator(model); err != nil {
		t.Fatal(err)
	}
	pos := controlTestPosition(t)
	workers := searcher.configuredWorkers()
	transitioned := make(chan int, len(workers))
	release := make(chan struct{})
	var releaseOnce sync.Once
	var firstPush [3]sync.Once
	evaluators := [3]*workerEvaluator{}
	contexts := [3]*countereval.SearchContext{}
	identities := [3]evaluatorModelIdentity{}
	searcher.smpHooks = &smpSearchHooks{afterReset: func(index int) {
		evaluator := workers[index].evaluator
		evaluators[index] = evaluator
		contexts[index] = evaluator.counterContext
		identities[index] = evaluator.Identity()
		evaluator.transitionObserver = func(_ *Position, observed *workerEvaluator) {
			if observed.nnueDepth() <= 0 {
				return
			}
			firstPush[index].Do(func() {
				transitioned <- index
				<-release
			})
		}
	}}

	done := make(chan *SearchInfo, 1)
	finished := make(chan struct{})
	t.Cleanup(func() {
		searcher.RequestStop()
		releaseOnce.Do(func() { close(release) })
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Errorf("Counter helper session survived cleanup")
		}
	})
	go func() {
		defer close(finished)
		done <- searcher.Search(pos, 2)
	}()
	seen := [3]bool{}
	for range workers {
		seen[waitSMPIndex(t, transitioned, "Counter pushed frames")] = true
	}
	for index, worker := range workers {
		evaluator := worker.evaluator
		if !seen[index] || evaluator == nil || evaluator != evaluators[index] {
			t.Fatalf("Counter worker %d did not enter its captured evaluator", index)
		}
		if identities[index].backend != evaluatorBackendCounter55 || evaluator.model.counter55 != model {
			t.Fatalf("Counter worker %d model/identity mismatch", index)
		}
		if contexts[index] == nil || evaluator.counterContext != contexts[index] || evaluator.nnueDepth() <= 0 {
			t.Fatalf("Counter worker %d missing or unpushed context", index)
		}
		for prior := 0; prior < index; prior++ {
			if evaluator == evaluators[prior] || contexts[index] == contexts[prior] {
				t.Fatalf("Counter workers %d and %d share evaluator or context", index, prior)
			}
		}
	}

	searcher.RequestStop()
	releaseOnce.Do(func() { close(release) })
	var result *SearchInfo
	select {
	case result = <-done:
	case <-time.After(10 * time.Second):
		searcher.RequestStop()
		t.Fatal("Counter helper session did not join")
	}
	if !result.Stopped || result.EffectiveThreads != 3 {
		t.Fatalf("Counter stopped/effective=%v/%d want=true/3", result.Stopped, result.EffectiveThreads)
	}
	for index, worker := range workers {
		worker.evaluator.transitionObserver = nil
		if worker.evaluator != evaluators[index] || worker.evaluator.nnueDepth() != 0 {
			t.Fatalf("Counter worker %d evaluator/context not balanced after join", index)
		}
		requireCounterRawMatchesFull(t, worker.evaluator, pos)
	}

	searcher.smpHooks = nil
	searcher.NewGame()
	next := mustParseCounterPosition(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")
	searcher.Search(next, 1)
	for index, worker := range workers {
		if worker.evaluator != evaluators[index] || worker.evaluator.counterContext != contexts[index] || worker.evaluator.nnueDepth() != 0 {
			t.Fatalf("Counter NewGame/root reset did not reuse balanced worker %d", index)
		}
		requireCounterRawMatchesFull(t, worker.evaluator, next)
	}
}
