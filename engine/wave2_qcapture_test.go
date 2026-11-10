package engine

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const wave2QCaptureFEN = "4k3/8/8/2p1p3/3P4/8/8/4K3 w - - 0 1"

func wave2QCaptureOption(t *testing.T, value int) {
	t.Helper()
	old := W2QCapture
	W2QCapture = value
	t.Cleanup(func() { W2QCapture = old })
}

func wave2QCaptureHistory(h *workerHistory, move Move, value int) {
	h.captureHistory[move.MovingPiece()][move.Destination()][move.CapturedPiece()] = value
}

func TestWave2QCaptureScore(t *testing.T) {
	wave2QCaptureOption(t, 1)
	pos := n3cPosition(t, wave2QCaptureFEN)
	a := adapterLegalMove(t, pos, "d4c5")
	b := adapterLegalMove(t, pos, "d4e5")
	h := new(workerHistory)
	base := h.wave2QCaptureScore(a, EmptyMove)
	if base != h.wave2QCaptureScore(b, EmptyMove) {
		t.Fatal("captures must start tied")
	}
	for _, value := range []int{-8192, -4096, -9, -7, 0, 7, 9, 4096, 8192} {
		wave2QCaptureHistory(h, a, value)
		want := base + max(-512, min(512, value/8))
		if got := h.wave2QCaptureScore(a, EmptyMove); got != want {
			t.Fatalf("history %d: score=%d want=%d", value, got, want)
		}
		if got := h.wave2QCaptureScore(a, a); got != 1<<30 {
			t.Fatalf("history %d overrode TT: %d", value, got)
		}
		W2QCapture = 0
		if got := h.wave2QCaptureScore(a, EmptyMove); got != base {
			t.Fatalf("off changed baseline: %d", got)
		}
		W2QCapture = 1
	}
	for _, favored := range []Move{a, b} {
		wave2QCaptureHistory(h, a, -8192)
		wave2QCaptureHistory(h, b, -8192)
		wave2QCaptureHistory(h, favored, 8192)
		other := a
		if favored == a {
			other = b
		}
		if h.wave2QCaptureScore(favored, EmptyMove) <= h.wave2QCaptureScore(other, EmptyMove) {
			t.Fatal("history failed to reverse tied captures")
		}
	}
}

func TestWave2QCapturePromotionsAndEnPassant(t *testing.T) {
	wave2QCaptureOption(t, 1)
	h := new(workerHistory)
	pos := n3cPosition(t, "r5k1/1P6/8/8/8/8/8/6K1 w - - 0 1")
	for _, suffix := range []string{"q", "r", "b", "n"} {
		quiet := adapterLegalMove(t, pos, "b7b8"+suffix)
		capture := adapterLegalMove(t, pos, "b7a8"+suffix)
		wave2QCaptureHistory(h, quiet, 8192)
		wave2QCaptureHistory(h, capture, 8192)
		for _, move := range []Move{quiet, capture} {
			base := 8*int(move.CapturedPiece().seeWeight()) - int(move.MovingPiece().seeWeight())
			want := base
			if move.IsCapture() {
				want += 512
			}
			if got := h.wave2QCaptureScore(move, EmptyMove); got != want {
				t.Fatalf("promotion %s: got=%d want=%d", move.ToString(), got, want)
			}
		}
	}
	for _, tc := range []struct {
		fen, notation string
		victim        Piece
	}{
		{"4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6", BlackPawn},
		{"4k3/8/8/8/3Pp3/8/8/4K3 b - d3 0 1", "e4d3", WhitePawn},
	} {
		pos := n3cPosition(t, tc.fen)
		move := adapterLegalMove(t, pos, tc.notation)
		if !move.IsEnPassant() || !move.IsCapture() || move.CapturedPiece() != tc.victim || pos.Board.PieceAt(move.Destination()) != NoPiece {
			t.Fatal("invalid en passant fixture")
		}
		h.captureHistory[move.MovingPiece()][move.Destination()][NoPiece] = -8192
		h.captureHistory[move.MovingPiece()][move.Destination()][tc.victim] = 800
		base := 8*int(tc.victim.seeWeight()) - int(move.MovingPiece().seeWeight())
		if got := h.wave2QCaptureScore(move, EmptyMove); got != base+100 {
			t.Fatalf("en passant indexed wrong victim: %d", got)
		}
	}
}

// Observe evaluator transitions in the real qsearch loop, rather than sorting a
// test copy. A controlled static evaluation forces the first capture to cut off.
func TestWave2QCaptureProductionOrderAndRestoration(t *testing.T) {
	originalSearchOptions(t)
	wave2QCaptureOption(t, 0)
	for _, tc := range []struct {
		name        string
		enabled     int
		favored, tt string
	}{
		{"off", 0, "d4e5", ""},
		{"favor-left", 1, "d4c5", ""},
		{"favor-right", 1, "d4e5", ""},
		{"tt-first", 1, "d4e5", "d4c5"},
		{"tie-1", 1, "", ""},
		{"tie-2", 1, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			W2QCapture = tc.enabled
			pos := n3cPosition(t, wave2QCaptureFEN)
			before := snapshotPVPosition(pos)
			var generated [64]Move
			n := GenerateCapturesIntoBuffer(pos, generated[:])
			if n != 2 {
				t.Fatalf("want exactly two legal captures, got %d", n)
			}
			want := generated[0]
			searcher, err := NewSearchEngineWithHash(1)
			if err != nil {
				t.Fatal(err)
			}
			evaluator := ngnn1NewWorker(t, ngnn3TestNetwork(t, 32, true), pos)
			ngnn3Assert(t, evaluator, pos)
			info := newN3CDirectSearchInfo(searcher, evaluator, NewCache(1), 0)
			info.maxNodes = 8
			if tc.favored != "" {
				for _, move := range generated[:n] {
					wave2QCaptureHistory(info.history, move, -8192)
				}
				favored := adapterLegalMove(t, pos, tc.favored)
				wave2QCaptureHistory(info.history, favored, 8192)
				if tc.enabled == 1 {
					want = favored
				}
			}
			if tc.tt != "" {
				want = adapterLegalMove(t, pos, tc.tt)
				info.tt.Set(pos.Hash(), want, -100, 0, LowerBound, false)
			}
			historyBefore := info.history.captureHistory
			var visited []Move
			evaluator.transitionObserver = func(p *Position, e *workerEvaluator) {
				oracle := e.staticOracle
				e.staticOracle = nil
				ngnn3Assert(t, e, p)
				e.staticOracle = oracle
				for _, move := range generated[:n] {
					if p.Board.PieceAt(move.Destination()) == WhitePawn {
						visited = append(visited, move)
					}
				}
			}
			evaluator.staticOracle = func(p *Position) int {
				if p.Turn() == White {
					return 0
				}
				return -100
			}
			if got := quiescence(pos, -50, 50, 0, info); got != 50 || info.Stopped || info.QBetaCutoffs != 1 {
				t.Fatalf("expected capture cutoff: score=%d stopped=%t cutoffs=%d", got, info.Stopped, info.QBetaCutoffs)
			}
			if len(visited) != 1 || visited[0] != want {
				t.Fatalf("production order=%v want first=%s", visited, want.ToString())
			}
			stored, _, _, _, hit, _ := info.tt.Get(pos.Hash())
			if !hit || stored != want {
				t.Fatal("TT did not record selected cutting move")
			}
			if !reflect.DeepEqual(before, snapshotPVPosition(pos)) || info.FrameDepth != 0 || info.RepStackLen != 0 || evaluator.evaluationDepth() != 0 {
				t.Fatal("cutoff did not restore board, frames, repetition and evaluator depth")
			}
			if !reflect.DeepEqual(historyBefore, info.history.captureHistory) {
				t.Fatal("qsearch changed capture learning")
			}
			evaluator.staticOracle = nil
			ngnn3Assert(t, evaluator, pos)
			t.Logf("W2QCapture=%d first=%s nodes=%d cutoff=restored", tc.enabled, want.ToString(), info.Nodes)
		})
	}
}

func TestWave2QCaptureUCIOption(t *testing.T) {
	wave2QCaptureOption(t, 0)
	u := NewUCIEngine()
	var out bytes.Buffer
	u.handleCommand("uci", &out)
	if !strings.Contains(out.String(), fmt.Sprintf("option name W2QCapture type spin default %d min 0 max 1\n", acceptedOptionDefaults["W2QCapture"])) {
		t.Fatal("missing UCI option with accepted default")
	}
	for _, value := range []string{"1", "0"} {
		u.handleCommand("setoption name W2QCapture value "+value, &out)
		if (W2QCapture == 1) != (value == "1") {
			t.Fatal("UCI option did not apply")
		}
	}
}
