package main

import "github.com/ehrlich-b/ngn/engine"

// Match the ordinary probe's HCE run when no network was requested.
// For the NNUE run, select the embedded default without opening path.
func selectNetwork(e *engine.SearchEngine, path string) error {
	if path == "" {
		return nil
	}
	n, err := engine.LoadDefaultNet()
	if err != nil {
		return err
	}
	return e.SelectNGNN1Evaluator(n)
}
