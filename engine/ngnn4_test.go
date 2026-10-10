package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func ngnn4TestBytes(h, h2, kb, nb int, extreme bool) []byte {
	old := ngnn3TestBytes(h, kb, nb, extreme)
	var out bytes.Buffer
	out.WriteString("NGN4")
	for _, field := range []uint32{4, uint32(h), uint32(kb), uint32(nb), uint32(h2), 255, 64, 400} {
		_ = binary.Write(&out, binary.LittleEndian, field)
	}
	out.Write(old[32:96])
	out.Write(old[96 : 96+(kb*768+1)*h*2])
	rng := rand.New(rand.NewSource(91008))
	for i := 0; i < nb*h2*2*h; i++ {
		out.WriteByte(byte(int8(rng.Intn(129) - 64)))
	}
	for i := 0; i < nb*h2*2; i++ {
		w := int16(rng.Intn(257) - 128)
		if extreme {
			w = int16(rng.Uint32())
		}
		_ = binary.Write(&out, binary.LittleEndian, w)
	}
	for i := 0; i < nb; i++ {
		_ = binary.Write(&out, binary.LittleEndian, int32(i*137-1300))
	}
	_ = binary.Write(&out, binary.LittleEndian, crc32.ChecksumIEEE(out.Bytes()))
	return out.Bytes()
}

// Independent scalar layer: widen before products and clamp each activation
// explicitly; no production hidden-layer helper or output kernel is used.
func ngnn4Scalar(n *NGNN1Network, acc []int32, turn Color, bucket int) int {
	clip := func(x int64) int64 {
		if x < 0 {
			return 0
		}
		if x > 255 {
			return 255
		}
		return x
	}
	us, them := 0, n.hidden
	if turn == Black {
		us, them = them, us
	}
	var total int64
	for j := 0; j < n.hidden2; j++ {
		row := (bucket*n.hidden2 + j) * 2 * n.hidden
		var dot int64
		for i := 0; i < n.hidden; i++ {
			dot += clip(int64(acc[us+i])) * int64(n.layer2[row+i])
			dot += clip(int64(acc[them+i])) * int64(n.layer2[row+n.hidden+i])
		}
		x := clip(dot/64 + int64(n.bias2[bucket*n.hidden2+j]))
		total += x * x * int64(n.output[bucket*n.hidden2+j])
	}
	bias := n.outputBias
	if bucket > 0 {
		bias = n.otherBiases[bucket-1]
	}
	cp := (total/255 + int64(bias)) * 400 / (255 * 64)
	if cp < -24999 {
		return -24999
	}
	if cp > 24999 {
		return 24999
	}
	return int(cp)
}

func ngnn4Assert(t *testing.T, e *workerEvaluator, pos *Position) {
	t.Helper()
	n := e.nnue.network
	view := *n
	view.output = make([]int16, view.buckets*2*view.hidden)
	want, _ := ngnn3Oracle(&view, pos)
	cp := ngnn4Scalar(n, want, pos.Turn(), n.bucket(pos))
	if !reflect.DeepEqual(e.nnue.at(e.depth), want) || e.SearchSTM(pos) != cp {
		t.Fatalf("NGNN4 scalar/worker mismatch at ply %d", e.depth)
	}
}

func TestNGNN4Validation(t *testing.T) {
	good := ngnn4TestBytes(16, 16, 8, 8, false)
	for _, field := range [][2]uint32{{4, 3}, {8, 17}, {12, 0}, {12, 9}, {16, 3}, {20, 0}, {20, 15}, {20, 17}, {20, 80}, {20, 0xffffffff}, {24, 254}, {28, 65}, {32, 401}} {
		bad := bytes.Clone(good)
		binary.LittleEndian.PutUint32(bad[field[0]:], field[1])
		binary.LittleEndian.PutUint32(bad[len(bad)-4:], crc32.ChecksumIEEE(bad[:len(bad)-4]))
		if n, err := ReadNGNN1(bytes.NewReader(bad)); err == nil || n != nil {
			t.Fatalf("accepted field %v", field)
		}
	}
	for _, offset := range []int{36, 99, 100 + (8*768+1)*16*2} {
		bad := bytes.Clone(good)
		bad[offset] = 127
		binary.LittleEndian.PutUint32(bad[len(bad)-4:], crc32.ChecksumIEEE(bad[:len(bad)-4]))
		if n, err := ReadNGNN1(bytes.NewReader(bad)); err == nil || n != nil {
			t.Fatalf("accepted bad byte %d", offset)
		}
	}
	for _, cut := range []int{0, 23, 24, 35, 99, 100, len(good) - 1} {
		if n, err := ReadNGNN1(bytes.NewReader(good[:cut])); err == nil || n != nil {
			t.Fatalf("accepted truncation %d", cut)
		}
	}
	bad := bytes.Clone(good)
	bad[104] ^= 1
	for _, data := range [][]byte{bad, append(bytes.Clone(good), 0)} {
		if n, err := ReadNGNN1(bytes.NewReader(data)); err == nil || n != nil {
			t.Fatal("accepted CRC/trailing bytes")
		}
	}
}

func TestNGNN4LayerBoundsAndScalar(t *testing.T) {
	rng := rand.New(rand.NewSource(444))
	for _, h := range []int{16, 48, 512, 2048} {
		for _, h2 := range []int{16, 32, 48, 64} {
			n, err := ReadNGNN1(bytes.NewReader(ngnn4TestBytes(h, h2, 1, 8, true)))
			if err != nil || !n.valid() {
				t.Fatalf("H=%d L=%d: %v", h, h2, err)
			}
			acc := make([]int32, 2*h)
			for trial := 0; trial < 5; trial++ {
				for i := range acc {
					acc[i] = int32(rng.Uint32())
					if trial == 1 {
						acc[i] = 255
					}
					if trial == 2 {
						acc[i] = -32768
					}
					if trial == 3 {
						acc[i] = int32(i%257) - 1
					}
				}
				for bucket := 0; bucket < 8; bucket++ {
					for _, turn := range []Color{White, Black} {
						if got, want := n.evaluateBucket(acc, turn, bucket), ngnn4Scalar(n, acc, turn, bucket); got != want {
							t.Fatalf("H=%d L=%d b=%d trial=%d got=%d want=%d", h, h2, bucket, trial, got, want)
						}
					}
				}
			}
		}
	}
}

func TestNGNN4IncrementalAndReload(t *testing.T) {
	n, err := ReadNGNN1(bytes.NewReader(ngnn4TestBytes(32, 16, 8, 8, false)))
	if err != nil {
		t.Fatal(err)
	}
	for _, fen := range []string{
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
		"r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
		"4k3/8/8/3pP3/8/8/PP6/4K3 w - d6 0 1",
		"1r2k3/P7/8/8/8/8/PP6/4K3 w - - 0 1",
		"7k/8/8/8/8/8/8/3K4 w - - 0 1",
	} {
		pos, err := ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		e := ngnn1NewWorker(t, n, pos)
		ngnn4Assert(t, e, pos)
		for _, move := range GenerateLegalMoves(pos) {
			tr := e.mustPrepareMove(pos, move)
			for visit := 0; visit < 2; visit++ {
				ep, tag, clock, _ := pos.MakeMove(move)
				e.mustPushMadeMove(pos, tr, move, tag, ep, clock)
				ngnn4Assert(t, e, pos)
				e.mustPop()
				pos.UnMakeMove(move, tag, ep, clock)
				ngnn4Assert(t, e, pos)
			}
		}
	}
	u := NewUCIEngine()
	path := filepath.Join(t.TempDir(), "net.nnue")
	if err := os.WriteFile(path, ngnn4TestBytes(32, 16, 8, 8, false), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, command := range []string{
		"setoption name OwnBook value false", "setoption name Threads value 2",
		"setoption name EvalFile value " + path, "setoption name UseNNUE value true",
		"position startpos", "go depth 3",
	} {
		u.handleCommand(command, &out)
	}
	u.lifecycleMu.Lock()
	session := u.activeSearch
	u.lifecycleMu.Unlock()
	if session != nil {
		<-session.done
	}
	if u.searcher.SelectedEvaluatorBackend() != EvaluatorBackendNGNN4Name {
		t.Fatal(out.String())
	}
	for _, w := range u.searcher.configuredWorkers() {
		ngnn4Assert(t, w.evaluator, u.position)
	}
	old := u.evaluatorConfigSnapshot().network
	if err := os.WriteFile(path, ngnn4TestBytes(16, 32, 4, 2, false), 0600); err != nil {
		t.Fatal(err)
	}
	u.handleCommand("setoption name EvalFile value "+path, &out)
	u.handleCommand("eval", &out)
	if u.evaluatorConfigSnapshot().network == old || u.searcher.SelectedEvaluatorBackend() != EvaluatorBackendNGNN4Name {
		t.Fatal("reload retained old NGNN4 model")
	}
	ngnn4Assert(t, u.searcher.worker.evaluator, u.position)
}

func TestNGNN4TrainerParity(t *testing.T) {
	netPath, evalsPath := os.Getenv("NGN_NGNN4_PARITY_NET"), os.Getenv("NGN_NGNN4_PARITY_EVALS")
	if netPath == "" && evalsPath == "" {
		netPath = filepath.Join("..", "testdata", "ngnn4", "random_h16_l16_k8_b8.nnue")
		evalsPath = filepath.Join("..", "testdata", "ngnn4", "random_h16_l16_k8_b8_evals.json")
	}
	raw, err := os.ReadFile(netPath)
	if err != nil {
		t.Fatal(err)
	}
	n, err := ReadNGNN1(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(evalsPath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Format      string `json:"format"`
		Hidden      int    `json:"hidden"`
		Hidden2     int    `json:"hidden2"`
		Buckets     int    `json:"buckets"`
		KingBuckets int    `json:"king_buckets"`
		SHA         string `json:"network_sha256"`
		Cases       []struct {
			FEN    string `json:"fen"`
			CP     *int   `json:"eval_cp"`
			Bucket int    `json:"bucket"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Format != "NGNN4" || n.version != 4 || fixture.Hidden != n.hidden || fixture.Hidden2 != n.hidden2 || fixture.Buckets != n.buckets || fixture.KingBuckets != n.kingBuckets || fixture.SHA != fmt.Sprintf("%x", sha256.Sum256(raw)) || len(fixture.Cases) < 20 {
		t.Fatal("NGNN4 fixture metadata mismatch")
	}
	e := ngnn1NewWorker(t, n, newUCIStartingPosition())
	for i, c := range fixture.Cases {
		pos, err := ParseFEN(c.FEN)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Reset(pos); err != nil {
			t.Fatal(err)
		}
		ngnn4Assert(t, e, pos)
		if c.CP == nil || e.SearchSTM(pos) != *c.CP || n.bucket(pos) != c.Bucket {
			t.Fatalf("trainer case %d: %s", i, c.FEN)
		}
	}
	t.Logf("%d exact NGNN4 trainer cases", len(fixture.Cases))
}
