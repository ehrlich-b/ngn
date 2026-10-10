package main

import (
	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/internal/uci"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSummaryPreservesReversedPairStatisticAndFailures(t *testing.T) {
	pairs := []pair{
		{Games: []playedGame{{Result: uci.GameResult{Res: uci.Win}}, {Result: uci.GameResult{Res: uci.Loss}}}},
		{Games: []playedGame{{Result: uci.GameResult{Res: uci.Win}}, {Result: uci.GameResult{Res: uci.Draw}}}},
		{Games: []playedGame{{Result: uci.GameResult{Res: uci.Win}}}, Fault: true},
	}
	s := summary(pairs)
	if s["penta"] != [5]int{0, 0, 1, 1, 0} || s["games"] != 4 || s["faulty_pairs"] != 1 || s["score"] != .625 {
		t.Fatal(s)
	}
	if !gameFault(playedGame{Result: uci.GameResult{Reason: "time-forfeit"}}) || gameFault(playedGame{Result: uci.GameResult{Reason: "checkmate"}}) {
		t.Fatal("fault misclassified")
	}
}
func TestFinalReportedWorkIgnoresPVTokens(t *testing.T) {
	rows := []string{"info depth 4 nodes 10 pv e2e4", "info depth 5 nodes 19 time 27 pv e2e4 nodes 999 depth 77", "bestmove e2e4"}
	if infoInt(rows, "nodes") != 19 || infoInt(rows, "time") != 27 || infoInt(rows, "depth") != 5 {
		t.Fatal(rows)
	}
}
func TestFrozenProbePositionsAreUniqueAndNonterminal(t *testing.T) {
	data, err := os.ReadFile("positions.txt")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, fen := range strings.Split(string(data), "\n") {
		if fen == "" || strings.HasPrefix(fen, "#") {
			continue
		}
		p, err := engine.ParseFEN(fen)
		if err != nil {
			t.Fatal(fen, err)
		}
		if seen[fen] || len(engine.GenerateLegalMoves(p)) == 0 {
			t.Fatal("duplicate/terminal", fen)
		}
		seen[fen] = true
	}
	if len(seen) != 32 {
		t.Fatal(len(seen))
	}
}

func TestProbeRejectsSyntacticButIllegalMove(t *testing.T) {
	if !legalBestmove(uci.StartFEN, "bestmove e2e4") || legalBestmove(uci.StartFEN, "bestmove e2e5") || legalBestmove(uci.StartFEN, "bestmove (none)") {
		t.Fatal("probe legality failed")
	}
}

func TestOfflineMedianBudgetAndIncompleteProbeRejection(t *testing.T) {
	code := `import json, pathlib, runpy, tempfile
ns=runpy.run_path('review.py')
with tempfile.TemporaryDirectory() as td:
 p=pathlib.Path(td)/'probes.jsonl'
 rows=[dict(engine=engine,movetime_ms=ms,nodes=1000,natural_budget_compliance=True) for _ in range(96) for engine in ['ngn','opponent'] for ms in [1000,5000]]
 p.write_text(''.join(json.dumps(r)+'\n' for r in rows))
 assert ns['node_budget'](td)==1000
 p.write_text(''.join(json.dumps(r)+'\n' for r in rows[:-1]))
 try:ns['node_budget'](td)
 except AssertionError:pass
 else:raise AssertionError('incomplete probe admitted')
`
	if out, err := exec.Command("python3", "-c", code).CombinedOutput(); err != nil {
		t.Fatalf("offline reviewer: %v %s", err, out)
	}
}
