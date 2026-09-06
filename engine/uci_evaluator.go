package engine

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ehrlich-b/ngn/countereval"
	"github.com/ehrlich-b/ngn/nnue"
)

const (
	uciEvaluatorHCE       = "hce"
	uciEvaluatorNGNV1     = "ngn-v1"
	uciEvaluatorCounter55 = "counter-5.5"
)

type stagedEvaluatorKind uint8

const (
	stagedEvaluatorNone stagedEvaluatorKind = iota
	stagedEvaluatorNGNV1
	stagedEvaluatorCounter55
)

type stagedEvaluatorModel struct {
	kind      stagedEvaluatorKind
	ngnV1     *nnue.Model
	counter55 *countereval.Model
}

func (model stagedEvaluatorModel) backendName() string {
	switch model.kind {
	case stagedEvaluatorNGNV1:
		return uciEvaluatorNGNV1
	case stagedEvaluatorCounter55:
		return uciEvaluatorCounter55
	default:
		return ""
	}
}

type uciEvaluatorConfig struct {
	backend string
	file    string
	staged  stagedEvaluatorModel
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

func loadUCIEvaluatorModel(path string) (stagedEvaluatorModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return stagedEvaluatorModel{}, fmt.Errorf("open %q: %w", path, err)
	}
	defer file.Close()

	var prefix [countereval.LegacyHeaderSize]byte
	count, readErr := io.ReadFull(file, prefix[:])
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return stagedEvaluatorModel{}, fmt.Errorf("read evaluator header %q: %w", path, readErr)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return stagedEvaluatorModel{}, fmt.Errorf("rewind evaluator %q: %w", path, err)
	}
	switch {
	case nnue.RecognizesHeader(prefix[:count]):
		model, err := nnue.Load(file)
		if err != nil {
			return stagedEvaluatorModel{}, fmt.Errorf("load exact ngn-v1 %q: %w", path, err)
		}
		return stagedEvaluatorModel{kind: stagedEvaluatorNGNV1, ngnV1: model}, nil
	case countereval.RecognizesLegacyHeader(prefix[:count]):
		model, _, err := countereval.LoadCounter55Legacy(file)
		if err != nil {
			return stagedEvaluatorModel{}, fmt.Errorf("load exact counter-5.5 %q: %w", path, err)
		}
		return stagedEvaluatorModel{kind: stagedEvaluatorCounter55, counter55: model}, nil
	default:
		return stagedEvaluatorModel{}, fmt.Errorf("unsupported evaluator format %q", path)
	}
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
	if backend != uciEvaluatorHCE && backend != uciEvaluatorNGNV1 && backend != uciEvaluatorCounter55 {
		uci.sendEvalOptionError(output, "unsupported EvalBackend %q (want hce, ngn-v1, or counter-5.5)", value)
		return
	}

	config := uci.evaluatorConfigSnapshot()
	var err error
	switch backend {
	case uciEvaluatorHCE:
		err = uci.searcher.SelectHCEEvaluator()
	case uciEvaluatorNGNV1:
		if config.file == "" || config.staged.kind != stagedEvaluatorNGNV1 || config.staged.ngnV1 == nil {
			uci.sendEvalOptionError(output, "EvalBackend ngn-v1 requires a matching valid staged EvalFile")
			return
		}
		err = uci.searcher.SelectNGNV1Evaluator(config.staged.ngnV1)
	case uciEvaluatorCounter55:
		if config.file == "" || config.staged.kind != stagedEvaluatorCounter55 || config.staged.counter55 == nil {
			uci.sendEvalOptionError(output, "EvalBackend counter-5.5 requires a matching valid staged EvalFile")
			return
		}
		err = uci.searcher.SelectCounter55Evaluator(config.staged.counter55)
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
		if config.backendName() != uciEvaluatorHCE {
			uci.sendEvalOptionError(output, "cannot clear EvalFile while EvalBackend is %s", config.backendName())
			return
		}
		uci.lifecycleMu.Lock()
		uci.evaluatorConfig.file = ""
		uci.evaluatorConfig.staged = stagedEvaluatorModel{}
		uci.lifecycleMu.Unlock()
		return
	}

	staged, err := loadUCIEvaluatorModel(path)
	if err != nil {
		uci.sendEvalOptionError(output, "%v", err)
		return
	}
	active := config.backendName()
	if active != uciEvaluatorHCE && active != staged.backendName() {
		uci.sendEvalOptionError(output, "cannot replace active %s with %s EvalFile; select hce first", active, staged.backendName())
		return
	}
	switch active {
	case uciEvaluatorNGNV1:
		err = uci.searcher.SelectNGNV1Evaluator(staged.ngnV1)
	case uciEvaluatorCounter55:
		err = uci.searcher.SelectCounter55Evaluator(staged.counter55)
	}
	if err != nil {
		uci.sendEvalOptionError(output, "replace active %s model: %v", active, err)
		return
	}

	uci.lifecycleMu.Lock()
	uci.evaluatorConfig.file = path
	uci.evaluatorConfig.staged = staged
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
