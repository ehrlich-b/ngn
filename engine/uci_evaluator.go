package engine

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ehrlich-b/ngn/nnue"
)

const (
	uciEvaluatorHCE   = "hce"
	uciEvaluatorNGNV1 = "ngn-v1"
)

type uciEvaluatorConfig struct {
	backend string
	file    string
	model   *nnue.Model
}

func newUCIEvaluatorConfig() uciEvaluatorConfig {
	return uciEvaluatorConfig{backend: uciEvaluatorHCE}
}

func (c uciEvaluatorConfig) backendName() string {
	if c.backend == "" {
		return uciEvaluatorHCE
	}
	return c.backend
}

func loadUCINGNV1Model(path string) (*nnue.Model, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()
	model, err := nnue.Load(file)
	if err != nil {
		return nil, fmt.Errorf("load exact ngn-v1 %q: %w", path, err)
	}
	return model, nil
}

func (uci *UCIEngine) evaluatorConfigSnapshot() uciEvaluatorConfig {
	uci.lifecycleMu.Lock()
	defer uci.lifecycleMu.Unlock()
	config := uci.evaluatorConfig
	if config.backend == "" {
		config.backend = uciEvaluatorHCE
	}
	return config
}

func (uci *UCIEngine) sendEvalOptionError(output io.Writer, format string, args ...interface{}) {
	uci.sendUCIMessage(output, fmt.Sprintf("info string error eval option: "+format, args...))
}

func (uci *UCIEngine) handleEvalBackendOption(value string, output io.Writer) {
	uci.joinSearch(true)
	backend := strings.ToLower(strings.TrimSpace(value))
	if backend != uciEvaluatorHCE && backend != uciEvaluatorNGNV1 {
		uci.sendEvalOptionError(output, "unsupported EvalBackend %q (want hce or ngn-v1)", value)
		return
	}

	config := uci.evaluatorConfigSnapshot()
	var err error
	switch backend {
	case uciEvaluatorHCE:
		err = uci.searcher.SelectHCEEvaluator()
	case uciEvaluatorNGNV1:
		if config.model == nil || config.file == "" {
			uci.sendEvalOptionError(output, "EvalBackend ngn-v1 requires a valid staged EvalFile")
			return
		}
		err = uci.searcher.SelectNGNV1Evaluator(config.model)
	}
	if err != nil {
		uci.sendEvalOptionError(output, "select %s: %v", backend, err)
		return
	}

	uci.lifecycleMu.Lock()
	uci.evaluatorConfig.backend = backend
	uci.lifecycleMu.Unlock()
}

func (uci *UCIEngine) handleEvalFileOption(value string, output io.Writer) {
	uci.joinSearch(true)
	path := strings.TrimSpace(value)
	config := uci.evaluatorConfigSnapshot()
	if path == "" || strings.EqualFold(path, "<empty>") {
		if config.backendName() == uciEvaluatorNGNV1 {
			uci.sendEvalOptionError(output, "cannot clear EvalFile while EvalBackend is ngn-v1")
			return
		}
		uci.lifecycleMu.Lock()
		uci.evaluatorConfig.file = ""
		uci.evaluatorConfig.model = nil
		uci.lifecycleMu.Unlock()
		return
	}

	model, err := loadUCINGNV1Model(path)
	if err != nil {
		uci.sendEvalOptionError(output, "%v", err)
		return
	}
	if config.backendName() == uciEvaluatorNGNV1 {
		if err := uci.searcher.SelectNGNV1Evaluator(model); err != nil {
			uci.sendEvalOptionError(output, "replace active ngn-v1 model: %v", err)
			return
		}
	}

	// Publish the staged path/model only after exact format validation and any
	// active-backend replacement have both succeeded.
	uci.lifecycleMu.Lock()
	uci.evaluatorConfig.file = path
	uci.evaluatorConfig.model = model
	uci.lifecycleMu.Unlock()
}

func (uci *UCIEngine) handleEvalDiagnostic(output io.Writer) {
	uci.joinSearch(true)
	score, backend, err := uci.searcher.EvaluateSelected(uci.position)
	if err != nil {
		uci.sendUCIMessage(output, fmt.Sprintf("info string error eval: %v", err))
		return
	}
	uci.sendUCIMessage(output, fmt.Sprintf(
		"info string eval backend %s score_cp %d pov side-to-move policy base-SearchSTM rule50-and-backend-adapter correction-history excluded",
		backend, score,
	))
}
