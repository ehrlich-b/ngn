package engine

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type residualEvalFixture struct {
	Accum      []int32
	Walks      []struct{ FEN, First string }
	Arithmetic []struct {
		Name       string
		LB, RB, HB int32
		L, A, R    int16
		X          int32
		Want       int
	}
}

func residualFixture(t *testing.T) residualEvalFixture {
	t.Helper()
	raw, err := os.ReadFile("../experiments/wave2-1104/residual-infer/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var f residualEvalFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// Read the handcrafted wire fixture produced by the independent Python encoder.
func residualBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../experiments/wave2-1104/residual-infer/synthetic-h32.ngn6")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func residualLoad(t *testing.T, raw []byte) *NGNN1Network {
	t.Helper()
	wave6CRC(raw)
	n, err := ReadNGNN1(bytes.NewReader(raw))
	if err != nil || !n.valid() || n.version != 6 {
		t.Fatalf("load: %v", err)
	}
	return n
}

// An intentionally straightforward int64 oracle: no production activation,
// residual helper, output kernel or clamped linear evaluator is called.
func residualOracle(n *NGNN1Network, acc []int32, turn Color, bucket int) (int, int64, int64) {
	h := n.hidden
	x := make([]int64, 2*h)
	for half := 0; half < 2; half++ {
		src := half
		if turn == Black {
			src = 1 - half
		}
		for i := 0; i < h; i++ {
			v := int64(acc[src*h+i])
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			x[half*h+i] = v
		}
	}
	sum := int64(0)
	for i := 0; i < 2*h; i++ {
		sum += x[i] * x[i] * int64(n.output[bucket*2*h+i])
	}
	bias := n.outputBias
	if bucket > 0 {
		bias = n.otherBiases[bucket-1]
	}
	linear := (sum/255 + int64(bias)) * 400 / 16320
	projection := int64(n.residual.outputBias[bucket])
	for j := 0; j < 16; j++ {
		z := int64(n.residual.hiddenBias[j])
		for i := 0; i < 2*h; i++ {
			z += x[i] * int64(n.residual.hiddenWeight[j*2*h+i])
		}
		z /= 64
		if z < 0 {
			z = 0
		}
		if z > 255 {
			z = 255
		}
		projection += z * int64(n.residual.output[bucket*16+j])
	}
	residual := projection * 400 / 16320
	cp := linear + residual
	if cp > 24999 {
		cp = 24999
	}
	if cp < -24999 {
		cp = -24999
	}
	return int(cp), linear, residual
}

func residualAssertStorage(t *testing.T, n *NGNN1Network, acc []int32, turn Color, bucket, want int) {
	t.Helper()
	if got := n.evaluateBucket(acc, turn, bucket); got != want {
		t.Fatalf("wide turn=%v bucket=%d got=%d want=%d", turn, bucket, got, want)
	}
	// Construct an exact material bucket and install the same raw accumulator in
	// a compact frame. This exercises the production compact output dispatch.
	p := &Position{Side: turn, EnPassant: NoSquare}
	p.Board.UpdateSquare(A1, WhiteKing, NoPiece)
	p.Board.UpdateSquare(H8, BlackKing, NoPiece)
	for i := 0; i < bucket*4; i++ {
		p.Board.UpdateSquare(Square(i+8), WhitePawn, NoPiece)
	}
	if n.bucket(p) != bucket {
		t.Fatal("bad material fixture")
	}
	compact := make([]int16, len(acc))
	for i, v := range acc {
		compact[i] = int16(v)
	}
	s := ngnn1State{network: n, frames: []ngnn1Frame{{compact: compact}}}
	if n.compactInput {
		if got := s.evaluatePosition(0, p, n); got != want {
			t.Fatalf("compact turn=%v bucket=%d got=%d want=%d", turn, bucket, got, want)
		}
		if len(s.wideView) != 0 {
			t.Fatal("compact search allocated diagnostic wide view")
		}
	}
}

func TestWave2ResidualEvalOracle(t *testing.T) {
	f := residualFixture(t)
	raw := residualBytes(t)
	n := residualLoad(t, raw)
	if !n.compactInput {
		t.Fatal("small fixture did not select compact storage")
	}
	nonzero := 0
	for sample := 0; sample < len(f.Accum); sample++ {
		acc := make([]int32, 64)
		for i := range acc {
			acc[i] = f.Accum[(i+sample)%len(f.Accum)]
		}
		for _, turn := range []Color{White, Black} {
			for bucket := 0; bucket < 8; bucket++ {
				want, _, r := residualOracle(n, acc, turn, bucket)
				if r != 0 {
					nonzero++
				}
				residualAssertStorage(t, n, acc, turn, bucket, want)
			}
		}
	}
	if nonzero < 100 {
		t.Fatal("insufficient nonzero residual coverage")
	}
	// Zero R and RB, while A and HB remain nonzero: the full residual calculation
	// still runs. Compare against the original NGNN3 evaluator and the oracle.
	for i := wave6Output; i < len(raw)-4; i++ {
		raw[i] = 0
	}
	zero := residualLoad(t, raw)
	for sample := 0; sample < len(f.Accum); sample++ {
		acc := make([]int32, 64)
		for i := range acc {
			acc[i] = f.Accum[(i*5+sample)%len(f.Accum)]
		}
		for _, turn := range []Color{White, Black} {
			for bucket := 0; bucket < 8; bucket++ {
				want, _, r := residualOracle(zero, acc, turn, bucket)
				if r != 0 || want != zero.residual.linear.evaluateBucket(acc, turn, bucket) {
					t.Fatal("zero residual changed NGNN3")
				}
				residualAssertStorage(t, zero, acc, turn, bucket, want)
			}
		}
	}
	for _, c := range f.Arithmetic {
		t.Run(c.Name, func(t *testing.T) {
			raw := residualBytes(t)
			for i := 120; i < len(raw)-4; i++ {
				raw[i] = 0
			}
			binary.LittleEndian.PutUint16(raw[wave6Linear:], uint16(c.L))
			binary.LittleEndian.PutUint32(raw[wave6LinearBias:], uint32(c.LB))
			binary.LittleEndian.PutUint16(raw[wave6Hidden:], uint16(c.A))
			binary.LittleEndian.PutUint32(raw[wave6HiddenBias:], uint32(c.HB))
			binary.LittleEndian.PutUint16(raw[wave6Output:], uint16(c.R))
			binary.LittleEndian.PutUint32(raw[wave6OutputBias:], uint32(c.RB))
			n := residualLoad(t, raw)
			acc := make([]int32, 64)
			acc[0] = c.X
			want, _, _ := residualOracle(n, acc, White, 0)
			if want != c.Want {
				t.Fatalf("hand-calculated fixture=%d oracle=%d", c.Want, want)
			}
			residualAssertStorage(t, n, acc, White, 0, want)
		})
	}
	t.Logf("both STM, eight buckets, %d nonzero oracle cases, zero projection and %d signed/clamp cases", nonzero, len(f.Arithmetic))
}

func TestWave2ResidualEvalStorageBounds(t *testing.T) {
	for _, c := range []struct {
		name                 string
		bias, weight, output int16
		compact              bool
	}{
		{"positive endpoint", 32703, 1, 128, true}, {"positive overflow", 32704, 1, 128, false},
		{"negative endpoint", -32704, -1, -128, true}, {"negative overflow", -32705, -1, -128, false},
		{"large linear output", 0, 1, 129, false}, {"full int16", -32768, 32767, -32768, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := residualBytes(t)
			for k := 0; k < 8*768; k++ {
				binary.LittleEndian.PutUint16(raw[120+2*k*32:], uint16(c.weight))
			}
			binary.LittleEndian.PutUint16(raw[120+2*8*768*32:], uint16(c.bias))
			binary.LittleEndian.PutUint16(raw[wave6Linear:], uint16(c.output))
			n := residualLoad(t, raw)
			if n.compactInput != c.compact {
				t.Fatalf("compact=%v want %v", n.compactInput, c.compact)
			}
			p := newUCIStartingPosition()
			e := ngnn1NewWorker(t, n, p)
			acc, _ := ngnn3Oracle(n, p)
			want, _, _ := residualOracle(n, acc, p.Turn(), n.bucket(p))
			if e.SearchSTM(p) != want {
				t.Fatal("fallback score mismatch")
			}
			if c.compact && len(e.nnue.stack) != 0 {
				t.Fatal("compact allocated wide stack")
			}
			if !c.compact && len(e.nnue.compactStack) != 0 {
				t.Fatal("wide allocated compact stack")
			}
		})
	}
}

func residualWalkAssert(t *testing.T, e *workerEvaluator, p *Position) {
	t.Helper()
	n := e.nnue.network
	acc, _ := ngnn3Oracle(n, p) // independent board-coordinate accumulator oracle
	refreshed := make([]int32, 2*n.hidden)
	n.refresh(p, refreshed)
	want, _, _ := residualOracle(n, acc, p.Turn(), n.bucket(p))
	if !reflect.DeepEqual(acc, refreshed) || !reflect.DeepEqual(acc, e.nnue.at(e.depth)) || e.SearchSTM(p) != want || n.evaluateBucket(refreshed, p.Turn(), n.bucket(p)) != want {
		t.Fatalf("incremental/refresh/oracle mismatch at ply %d", e.depth)
	}
}

func TestWave2ResidualEvalWalk(t *testing.T) {
	f := residualFixture(t)
	for _, wide := range []bool{false, true} {
		for _, zero := range []bool{false, true} {
			raw := residualBytes(t)
			if wide {
				binary.LittleEndian.PutUint16(raw[120:], 32767)
			}
			if zero {
				for i := wave6Output; i < len(raw)-4; i++ {
					raw[i] = 0
				}
			}
			n := residualLoad(t, raw)
			if n.compactInput == wide {
				t.Fatal("walk storage not selected by bounds")
			}
			castles, epMoves, promos, mirrors, buckets, nulls := 0, 0, 0, 0, 0, 0
			for wi, w := range f.Walks {
				p, err := ParseFEN(w.FEN)
				if err != nil {
					t.Fatal(err)
				}
				e := ngnn1NewWorker(t, n, p)
				type undo struct {
					move  Move
					ep    Square
					tag   PositionState
					clock uint8
				}
				var undos []undo
				for ply := 0; ply < 128; ply++ {
					residualWalkAssert(t, e, p)
					if !p.IsInCheck() {
						tr := e.mustPrepareNull(p)
						ep := p.MakeNullMove()
						e.mustPushMadeNull(p, tr, ep)
						if e.nnue.frames[e.depth].slot != e.nnue.frames[e.depth-1].slot {
							t.Fatal("null frame failed to alias")
						}
						residualWalkAssert(t, e, p)
						e.mustPop()
						p.UnMakeNullMove(ep)
						nulls++
						residualWalkAssert(t, e, p)
					}
					moves := GenerateLegalMoves(p)
					if len(moves) == 0 {
						t.Fatalf("walk %d ended at %d", wi, ply)
					}
					move := moves[(ply*17+wi*11)%len(moves)]
					if ply == 0 {
						found := false
						for _, candidate := range moves {
							if candidate.ToString() == w.First {
								move = candidate
								found = true
								break
							}
						}
						if !found {
							t.Fatal("missing scripted special move: " + w.First)
						}
					} else {
						// Prefer safe king moves so both king bucket and mirror refreshes recur.
						for j := 0; j < len(moves); j++ {
							c := moves[(ply+j)%len(moves)]
							if c.MovingPiece().Type() == King {
								move = c
								break
							}
						}
					}
					if move.IsCastle() {
						castles++
					}
					if move.IsEnPassant() {
						epMoves++
					}
					if move.PromoType() != NoType {
						promos++
					}
					if move.MovingPiece().Type() == King {
						old := n.kingContext(p, p.Turn())
						next := n.contextAt(move.Destination(), p.Turn())
						if old.mirror != next.mirror {
							mirrors++
						}
						if old.bucket != next.bucket {
							buckets++
						}
					}
					tr := e.mustPrepareMove(p, move)
					ep, tag, clock, _ := p.MakeMove(move)
					e.mustPushMadeMove(p, tr, move, tag, ep, clock)
					undos = append(undos, undo{move, ep, tag, clock})
				}
				residualWalkAssert(t, e, p)
				for i := len(undos) - 1; i >= 0; i-- {
					u := undos[i]
					e.mustPop()
					p.UnMakeMove(u.move, u.tag, u.ep, u.clock)
					residualWalkAssert(t, e, p)
				}
				if err := e.Reset(p); err != nil {
					t.Fatal(err)
				}
				residualWalkAssert(t, e, p)
			}
			if castles == 0 || epMoves == 0 || promos == 0 || mirrors == 0 || buckets == 0 || nulls == 0 {
				t.Fatalf("missing walk coverage: %d %d %d %d %d %d", castles, epMoves, promos, mirrors, buckets, nulls)
			}
			t.Logf("wide=%v zero=%v: four 128-ply walks; castles=%d ep=%d promotions=%d mirrors=%d buckets=%d nulls=%d", wide, zero, castles, epMoves, promos, mirrors, buckets, nulls)
		}
	}
}

func TestWave2ResidualEvalUCIIdentity(t *testing.T) {
	u := NewUCIEngine()
	if err := u.searcher.ResizeHash(1); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "residual.ngn6")
	raw := residualBytes(t)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	u.handleCommand("setoption name EvalFile value "+path, &out)
	u.handleCommand("setoption name UseNNUE value true", &out)
	u.handleCommand("eval", &out)
	old := u.evaluatorConfigSnapshot().network
	if old == nil || u.searcher.SelectedEvaluatorBackend() != "ngnn6" || !strings.Contains(out.String(), "eval backend ngnn6") {
		t.Fatal(out.String())
	}
	acc, _ := ngnn3Oracle(old, u.position)
	want, _, _ := residualOracle(old, acc, u.position.Turn(), old.bucket(u.position))
	if score, backend, err := u.searcher.EvaluateSelected(u.position); err != nil || backend != "ngnn6" || score != want {
		t.Fatalf("selected eval: %d %s %v", score, backend, err)
	}
	// Two independent workers share immutable model identity, never their stacks.
	model := u.searcher.evaluatorModel
	a, err := model.newWorker(&hceEvaluator{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := model.newWorker(&hceEvaluator{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Reset(u.position); err != nil {
		t.Fatal(err)
	}
	if err := b.Reset(u.position); err != nil {
		t.Fatal(err)
	}
	if a.Identity() != b.Identity() || &a.nnue.compactStack[0] == &b.nnue.compactStack[0] {
		t.Fatal("worker identity/storage mismatch")
	}
	const key = 98764321
	u.searcher.TTStore(key, EmptyMove, 42, 1, Exact, false)
	u.searcher.worker.history.lastMovePlayed = NewMove(A1, A2, WhiteKing, NoPiece, NoType, 0)
	// Only residual bytes change; reloading must invalidate TT and worker history.
	binary.LittleEndian.PutUint32(raw[wave6OutputBias+7*4:], 123456)
	wave6CRC(raw)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	u.handleCommand("setoption name EvalFile value "+path, &out)
	if u.evaluatorConfigSnapshot().network == old {
		t.Fatal("reused identity")
	}
	if _, _, _, _, hit, _ := u.searcher.TTProbe(key); hit {
		t.Fatal("retained TT")
	}
	if score, backend, err := u.searcher.EvaluateSelected(u.position); err != nil || backend != "ngnn6" || score == want {
		t.Fatalf("residual-only reload did not change the selected score: %d %s %v", score, backend, err)
	}
	if u.searcher.worker.history.lastMovePlayed != EmptyMove || u.searcher.worker.evaluator.Identity() == a.Identity() {
		t.Fatal("retained worker identity/history")
	}
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	u.handleCommand("setoption name EvalFile value "+path, &out)
	if u.searcher.SelectedEvaluatorBackend() != "hce" || u.evaluatorConfigSnapshot().network != nil {
		t.Fatal("bad residual reload did not fail closed")
	}
	t.Log("EvalFile ngnn6 backend, residual-only reload invalidation, private workers, corrupt reload fail-closed")
}

func TestWave2ResidualEvalDefaultH768Zero(t *testing.T) {
	base, err := LoadDefaultNet()
	if err != nil {
		t.Fatal(err)
	}
	if base.version != 3 || base.hidden != 768 || base.residual != nil {
		t.Fatal("embedded NGNN3 default changed")
	}
	// Convert the actual winning embedded NGNN3 bytes to version 6, retaining its
	// linear payload exactly. Nonzero A and HB rule out a vacuous empty head.
	raw := append([]byte(nil), defaultNetBytes[:96]...)
	copy(raw, "NGN6")
	binary.LittleEndian.PutUint32(raw[4:], 6)
	for _, v := range []uint32{16, 64, 255, 64, 400, 0} {
		raw = binary.LittleEndian.AppendUint32(raw, v)
	}
	raw = append(raw, defaultNetBytes[96:len(defaultNetBytes)-4]...)
	for i := 0; i < 16*2*768; i++ {
		raw = binary.LittleEndian.AppendUint16(raw, uint16(i%17))
	}
	for j := 0; j < 16; j++ {
		raw = binary.LittleEndian.AppendUint32(raw, uint32(j*137))
	}
	raw = append(raw, make([]byte, 2*8*16+4*8+4)...)
	n := residualLoad(t, raw)
	for _, wide := range []bool{false, true} {
		copyNet := *n
		if wide {
			copyNet.compactInput = false
		}
		positions := []*Position{newUCIStartingPosition()}
		for _, w := range residualFixture(t).Walks {
			p, err := ParseFEN(w.FEN)
			if err != nil {
				t.Fatal(err)
			}
			positions = append(positions, p)
		}
		for _, p := range positions {
			e := ngnn1NewWorker(t, &copyNet, p)
			for _, turn := range []Color{White, Black} {
				p.Side = turn
				acc, want := ngnn3Oracle(base, p)
				oracle, _, r := residualOracle(n, acc, turn, n.bucket(p))
				if r != 0 || oracle != want || e.SearchSTM(p) != want {
					t.Fatal("H768 zero residual changed embedded NGNN3 score")
				}
			}
		}
		// Include every output bucket and both STM on clipped synthetic inputs
		// with the actual H768 weights, independently of material distribution.
		acc := make([]int32, 2*n.hidden)
		values := residualFixture(t).Accum
		for i := range acc {
			acc[i] = values[i%len(values)]
		}
		for _, turn := range []Color{White, Black} {
			for bucket := 0; bucket < 8; bucket++ {
				want := base.evaluateBucket(acc, turn, bucket)
				oracle, _, r := residualOracle(n, acc, turn, bucket)
				if r != 0 || oracle != want {
					t.Fatal("H768 zero oracle mismatch")
				}
				residualAssertStorage(t, &copyNet, acc, turn, bucket, want)
			}
		}
	}
	t.Logf("embedded H768 exact NGNN3 identity through compact/wide, both STM and all buckets, nonzero A/HB, zero R/RB; residual numeric bytes=%d", 64*768+352)
}
