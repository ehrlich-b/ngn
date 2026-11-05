package engine

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func wave2QFixtures(t *testing.T) []wave2CheckFixture {
	t.Helper()
	data, err := os.ReadFile("../experiments/wave2-1104/qchecks/witnesses.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []wave2CheckFixture
	if err := json.Unmarshal(data, &fixtures); err != nil || len(fixtures) == 0 {
		t.Fatalf("fixtures: %v", err)
	}
	return fixtures
}

func wave2QPosition(t *testing.T, fen string) *Position {
	t.Helper()
	pos, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	pos.Hash()
	pos.Positions[42] = 3
	return pos
}

// The oracle uses full make/check/unmake, never the passed predicate or the
// production append helper. Hashes identify root moves in the evaluator observer.
func wave2QRootMoves(pos *Position) (map[Move]bool, map[uint64]Move) {
	quiet := map[Move]bool{}
	hashes := map[uint64]Move{}
	var buffer [256]Move
	n := GenerateMovesIntoBuffer(pos, buffer[:])
	for _, move := range buffer[:n] {
		if move.IsCastle() && !isLegalCastle(pos, move) {
			continue
		}
		ep, tag, hc, _ := pos.MakeMove(move)
		legal, check := !isInCheck(pos, move.MovingPiece().Color()), pos.IsInCheck()
		if legal {
			hashes[pos.Hash()] = move
			if check && !move.IsCapture() && !move.IsEnPassant() && move.PromoType() == NoType {
				quiet[move] = true
			}
		}
		pos.UnMakeMove(move, tag, ep, hc)
	}
	return quiet, hashes
}

func wave2QInfo(pos *Position) *SearchInfo {
	return &SearchInfo{
		control: &SearchControl{}, history: &workerHistory{},
		evaluator: testHCEWorkerEvaluator(nil), tt: NewCache(1),
		CaptureBuffer: new([64]Move), Frames: make([]searchFrame, 16),
		maxNodes: 2000,
	}
}

func TestWave2QChecksProduction(t *testing.T) {
	old := W2QChecks
	defer func() { W2QChecks = old }()
	for _, fixture := range wave2QFixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			pos := wave2QPosition(t, fixture.FEN)
			before := wave2Snapshot(pos)
			quiet, hashes := wave2QRootMoves(pos)
			// Named witnesses must exist and independently give the advertised check.
			var all [256]Move
			nAll := GenerateMovesIntoBuffer(pos, all[:])
			for _, witness := range fixture.Witnesses {
				found := false
				for _, move := range all[:nAll] {
					if move.ToString() != witness.Move {
						continue
					}
					found = true
					legal := IsLegalMove(pos, move)
					ep, tag, hc, _ := pos.MakeMove(move)
					check := pos.IsInCheck()
					pos.UnMakeMove(move, tag, ep, hc)
					if check != witness.Check || legal == witness.Illegal {
						t.Fatalf("witness %s: check=%v legal=%v", witness.Move, check, legal)
					}
				}
				if !found {
					t.Fatalf("missing witness %s", witness.Move)
				}
			}
			for _, cold := range []bool{false, true} {
				pos.hash = before.Hash
				if cold {
					pos.hash = 0
				}
				saved := wave2Snapshot(pos)
				var moves, scratch [256]Move
				n := appendWave2QChecks(pos, moves[:], scratch[:])
				got := map[Move]bool{}
				for _, move := range moves[:n] {
					if got[move] {
						t.Fatalf("duplicate %s", move.ToString())
					}
					got[move] = true
				}
				if !reflect.DeepEqual(got, quiet) || !reflect.DeepEqual(saved, wave2Snapshot(pos)) {
					t.Fatal("append class/state differs from oracle")
				}
			}
			pos.hash = before.Hash
			var noisyPrefix []Move
			var offScore int
			var offNodes uint64
			for _, option := range []int{0, 1} {
				W2QChecks = option
				info := wave2QInfo(pos)
				// Zero static leaves keep the tiny branch proof independent of tuning.
				info.evaluator.staticOracle = func(*Position) int { return 0 }
				var searched []Move
				info.evaluator.transitionObserver = func(p *Position, e *workerEvaluator) {
					if e.depth == 1 {
						searched = append(searched, hashes[p.Hash()])
					}
				}
				// Seed a noncutting ordinary TT move, including at the enhanced
				// entry where only its ordering hint may cross the boundary.
				hashMove := EmptyMove
				for _, move := range hashes {
					if move.IsCapture() && (hashMove == EmptyMove || move.ToString() < hashMove.ToString()) {
						hashMove = move
					}
				}
				if hashMove != EmptyMove {
					info.tt.Set(pos.Hash(), hashMove, -1000, 0, LowerBound, false)
				}
				if option == 1 {
					for move := range quiet {
						info.tt.Set(pos.Hash()^wave2QChecksTTKey, move, -1000, 0, LowerBound, false)
						break
					}
				}
				score := quiescenceWithDepth(pos, -INFINITY, INFINITY, 0, info, 0)
				if info.Stopped {
					t.Fatal("bounded witness exhausted node limit")
				}
				if !reflect.DeepEqual(before, wave2Snapshot(pos)) || info.FrameDepth != 0 || info.RepStackLen != 0 || info.evaluator.depth != 0 {
					t.Fatal("production search changed root state")
				}
				if option == 0 {
					noisyPrefix = searched
					offScore = score
					offNodes = info.Nodes
				}
				if option == 1 {
					want := len(quiet)
					if pos.IsInCheck() {
						want = 0
					}
					if info.QChecksAdmitted != uint64(want) || info.QChecksSearched != uint64(want) {
						t.Fatalf("production admitted=%d searched=%d want=%d", info.QChecksAdmitted, info.QChecksSearched, want)
					}
					if len(searched) < len(noisyPrefix) {
						t.Fatal("noisy relative order changed")
					}
					for i, move := range noisyPrefix {
						if searched[i] != move {
							t.Fatal("noisy relative order changed")
						}
					}
					seen := map[Move]int{}
					for _, move := range searched {
						seen[move]++
					}
					for move := range quiet {
						if !pos.IsInCheck() && seen[move] != 1 {
							t.Fatalf("check %s searched %d times", move.ToString(), seen[move])
						}
					}
					if pos.IsInCheck() && (score != offScore || info.Nodes != offNodes) {
						t.Fatal("ordinary evasion search changed")
					}
				}
				t.Logf("W2QChecks=%d score=%d nodes=%d admitted=%d searched=%d rootMoves=%d", option, score, info.Nodes, info.QChecksAdmitted, info.QChecksSearched, len(searched))
			}
			W2QChecks = 1
			deeper := wave2QInfo(pos)
			deeper.evaluator.staticOracle = func(*Position) int { return 0 }
			quiescenceWithDepth(pos, -INFINITY, INFINITY, 0, deeper, 1)
			if deeper.Stopped || deeper.QChecksAdmitted != 0 || deeper.QChecksSearched != 0 {
				t.Fatal("checks admitted below first layer")
			}
		})
	}
}

// Deliberately tiny exhaustive tactical oracle: no TT, delta, SEE, alpha/beta
// cutoff or predicate. It follows noisy moves/evasions plus root quiet checks.
func wave2QOracle(t *testing.T, pos *Position, ply int) int {
	t.Helper()
	if ply > 8 {
		t.Fatal("tiny oracle exceeded its fixture bound")
	}
	inCheck := pos.IsInCheck()
	best := 0 // Same zero static leaves used by the production witness.
	if inCheck {
		best = -INFINITY
	}
	legal := 0
	var buffer [256]Move
	n := GenerateMovesIntoBuffer(pos, buffer[:])
	for _, move := range buffer[:n] {
		if move.IsCastle() && !isLegalCastle(pos, move) {
			continue
		}
		ep, tag, hc, _ := pos.MakeMove(move)
		if isInCheck(pos, move.MovingPiece().Color()) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue
		}
		legal++
		eligible := inCheck || move.IsCapture() || move.PromoType() != NoType || (ply == 0 && pos.IsInCheck())
		if eligible {
			score := -wave2QOracle(t, pos, ply+1)
			if score > best {
				best = score
			}
		}
		pos.UnMakeMove(move, tag, ep, hc)
	}
	if legal == 0 {
		if inCheck {
			return -MATE_VALUE + ply
		}
		return 0
	}
	return best
}

func TestWave2QChecksTacticalOracle(t *testing.T) {
	old := W2QChecks
	defer func() { W2QChecks = old }()
	pos := wave2QPosition(t, wave2QFixtures(t)[0].FEN)
	before := wave2Snapshot(pos)
	want := wave2QOracle(t, pos, 0)
	losingCheck := NewMove(G6, G8, WhiteQueen, NoPiece, NoType, 0)
	if staticExchangeEvaluation(pos, losingCheck) >= 0 {
		t.Fatal("quiet sacrificial check must be SEE-losing")
	}
	if want != MATE_VALUE-1 {
		t.Fatalf("oracle=%d", want)
	}
	for _, option := range []int{0, 1} {
		W2QChecks = option
		info := wave2QInfo(pos)
		info.evaluator.staticOracle = func(*Position) int { return 0 }
		// alpha exceeds zero stand-pat + delta margin, proving the quiet mate
		// bypasses capture-only delta pruning, even on a fail-low noisy prefix.
		got := quiescence(pos, 500, INFINITY, 0, info)
		if info.Stopped || (option == 1 && got != want) || (option == 0 && got != 0) {
			t.Fatalf("option=%d got=%d oracle=%d", option, got, want)
		}
		t.Logf("option=%d production=%d no-pruning oracle=%d admitted=%d", option, got, want, info.QChecksAdmitted)
	}
	if !reflect.DeepEqual(before, wave2Snapshot(pos)) {
		t.Fatal("oracle/search changed state")
	}
}

func TestWave2QChecksCancellation(t *testing.T) {
	old := W2QChecks
	defer func() { W2QChecks = old }()
	W2QChecks = 1
	network, err := LoadDefaultNet()
	if err != nil {
		t.Fatal(err)
	}
	for _, nnue := range []bool{false, true} {
		for _, limit := range []uint64{0, 2, 3, 4, 5} {
			pos := wave2QPosition(t, wave2QFixtures(t)[0].FEN)
			before := wave2Snapshot(pos)
			info := wave2QInfo(pos)
			info.RepStack[0] = 77
			info.RepStackLen = 1
			info.maxNodes = limit
			if nnue {
				info.evaluator = ngnn1NewWorker(t, network, pos)
			}
			var rootAcc []int32
			if nnue {
				rootAcc = append([]int32(nil), info.evaluator.nnue.at(0)...)
			}
			info.evaluator.staticOracle = func(*Position) int { return 0 }
			pushed := 0
			if limit == 0 {
				info.evaluator.transitionObserver = func(p *Position, e *workerEvaluator) {
					if e.depth == 1 && p.IsInCheck() {
						pushed++
						info.control.RequestStop()
					}
				}
			}
			got := quiescence(pos, -INFINITY, INFINITY, 0, info)
			if !info.Stopped || got != stoppedSearchScore || (limit == 0 && pushed != 1) {
				t.Fatalf("limit=%d stopped=%v pushes=%d score=%d", limit, info.Stopped, pushed, got)
			}
			if !reflect.DeepEqual(before, wave2Snapshot(pos)) || info.FrameDepth != 0 || info.RepStackLen != 1 || info.RepStack[0] != 77 || info.evaluator.depth != 0 {
				t.Fatal("cancel did not restore board/hash/phase/history/frames/repetition/evaluator")
			}
			if _, _, _, _, hit, _ := info.tt.Get(pos.Hash() ^ wave2QChecksTTKey); hit {
				t.Fatal("stopped parent stored TT bound")
			}
			if nnue && !reflect.DeepEqual(rootAcc, info.evaluator.nnue.at(0)) {
				t.Fatal("cancel changed H768 root accumulator")
			}
		}
	}
}

func TestWave2QChecksTTIsolation(t *testing.T) {
	old := W2QChecks
	defer func() { W2QChecks = old }()
	W2QChecks = 1
	pos := wave2QPosition(t, wave2QFixtures(t)[0].FEN)
	info := wave2QInfo(pos)
	info.evaluator.staticOracle = func(*Position) int { return 0 }
	if got := quiescence(pos, -100, 1000, 0, info); got != 1000 {
		t.Fatalf("mate cutoff=%d", got)
	}
	if _, _, _, _, hit, _ := info.tt.Get(pos.Hash()); hit {
		t.Fatal("enhanced bound stored in capture-only table")
	}
	if _, _, _, _, hit, _ := info.tt.Get(pos.Hash() ^ wave2QChecksTTKey); !hit {
		t.Fatal("missing enhanced bound")
	}
	info.QChecksAdmitted = 0
	if got := quiescenceWithDepth(pos, -INFINITY, INFINITY, 0, info, 1); got != 0 || info.QChecksAdmitted != 0 {
		t.Fatalf("enhanced bound leaked below first layer: %d", got)
	}
	W2QChecks = 0
	if got := quiescence(pos, -INFINITY, INFINITY, 0, info); got != 0 {
		t.Fatalf("enhanced bound leaked with option off: %d", got)
	}
}
