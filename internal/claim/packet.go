package claim

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ehrlich-b/ngn/engine"
)

// VerifyConfirmationInputs checks a retained packet without launching processes,
// probing devices, creating receipts, or accepting partial confirmation results.
// The returned analysis admits only the candidate frozen in the screen receipt.
func VerifyConfirmationInputs(path string) (Analysis, error) {
	var saved Analysis
	r, err := ReadReceipt(path)
	if err != nil {
		return saved, err
	}
	if r.Plan.Mode != "confirmation" {
		return saved, fmt.Errorf("confirmation receipt required")
	}
	if err = Recheck(r); err != nil {
		return saved, err
	}
	if _, err = PreflightPlan(r.Plan, r.ToolSourceRoot, r.ToolBuildArgv); err != nil {
		return saved, err
	}
	if r.ScreenReceipt.Path == "" || r.ScreenAnalysis.Path == "" {
		return saved, fmt.Errorf("admitted screen receipt and analysis required")
	}
	sr, err := ReadReceipt(r.ScreenReceipt.Path)
	if err != nil {
		return saved, err
	}
	if err = ReadJSON(r.ScreenAnalysis.Path, &saved); err != nil {
		return saved, err
	}
	fresh, err := Analyze(r.ScreenReceipt.Path, saved.Audit.Path, saved.GameDirectory)
	if err != nil {
		return saved, err
	}
	if !reflect.DeepEqual(saved, fresh) {
		return fresh, fmt.Errorf("retained screen analysis differs from independent reanalysis")
	}
	if !screenAdmitsConfirmation(sr, fresh, r) {
		return fresh, fmt.Errorf("screen does not admit frozen confirmation inputs")
	}
	return fresh, nil
}

// OpeningPositionKeys replays legal UCI histories and ignores move counters and
// uncapturable en passant squares when identifying transposed starting positions.
func OpeningPositionKeys(lines []string) ([]string, error) {
	keys := make([]string, 0, len(lines))
	for i, line := range lines {
		p, err := engine.ParseFEN(StartFEN)
		if err != nil {
			return nil, err
		}
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			return nil, fmt.Errorf("empty opening %d", i)
		}
		for _, token := range tokens {
			m, err := LegalMove(p, token)
			if err != nil {
				return nil, fmt.Errorf("opening %d: %w", i, err)
			}
			if _, _, _, ok := p.GameMakeMove(m); !ok {
				return nil, fmt.Errorf("opening %d refused move", i)
			}
		}
		keys = append(keys, PositionKey(p))
	}
	return keys, nil
}
