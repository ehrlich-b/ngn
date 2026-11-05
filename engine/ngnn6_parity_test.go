package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const residualParityDir = "../experiments/wave2-1104/residual-convert"

type residualParityCase struct {
	FEN       string `json:"fen"`
	Bucket    int    `json:"bucket"`
	ZeroCP    int    `json:"zero_cp"`
	NonzeroCP int    `json:"nonzero_cp"`
	Acc       []int32
}

type residualParityPacket struct {
	Input, Output, Nonzero struct{ SHA256 string }
	Cases                  []residualParityCase
	Transitions            []struct {
		Name, FEN, Move     string
		Before, After, Null residualParityCase
	}
}

func readResidualParity(t *testing.T, name string, dst any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(residualParityDir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatal(err)
	}
}

func loadResidualParity(t *testing.T, name, hash string, version int) *NGNN1Network {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(residualParityDir, name))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != hash {
		t.Fatal("fixture hash mismatch: " + name)
	}
	n, err := ReadNGNN1(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if n.version != version || (version == 6 && n.residual == nil) {
		t.Fatal("wrong format route: " + name)
	}
	return n
}

func assertResidualParity(t *testing.T, e *workerEvaluator, p *Position, c residualParityCase, nonzero bool) {
	t.Helper()
	want := c.ZeroCP
	if nonzero {
		want = c.NonzeroCP
	}
	n := e.nnue.network
	acc := make([]int32, 2*n.hidden)
	n.refresh(p, acc)
	if !reflect.DeepEqual(acc, c.Acc) || !reflect.DeepEqual(e.nnue.at(e.depth), c.Acc) {
		t.Fatalf("Python/Go refresh/incremental accumulator mismatch: %s", c.FEN)
	}
	if got := e.SearchSTM(p); got != want || n.evaluateBucket(acc, p.Turn(), n.bucket(p)) != want || n.bucket(p) != c.Bucket {
		t.Fatalf("Python/Go parity: %s got %d want %d", c.FEN, got, want)
	}
}

func TestWave2ResidualParityRecords(t *testing.T) {
	var packet residualParityPacket
	readResidualParity(t, "parity.json", &packet)
	if len(packet.Cases) != 32 {
		t.Fatal("expected 32 packed-record cases")
	}
	base := loadResidualParity(t, "source-h32.ngn3", packet.Input.SHA256, 3)
	zero := loadResidualParity(t, "zero-h32.ngn6", packet.Output.SHA256, 6)
	nonzero := loadResidualParity(t, "nonzero-h32.ngn6", packet.Nonzero.SHA256, 6)
	if !reflect.DeepEqual(base.weights, zero.weights) || !reflect.DeepEqual(base.bias, zero.bias) ||
		!reflect.DeepEqual(base.output, zero.output) || base.kingMap != zero.kingMap ||
		base.outputBias != zero.outputBias || !reflect.DeepEqual(base.otherBiases, zero.otherBiases) {
		t.Fatal("lift changed inherited bytes")
	}
	changed := 0
	coverage := make(map[[3]int]bool)
	for _, c := range packet.Cases {
		if c.ZeroCP != c.NonzeroCP {
			changed++
		}
		p, err := ParseFEN(c.FEN)
		if err != nil {
			t.Fatal(err)
		}
		coverage[[3]int{c.Bucket, int(zero.kingContext(p, White).mirror), int(p.Turn())}] = true
		for _, n := range []*NGNN1Network{base, zero, nonzero} {
			for _, wide := range []bool{false, true} {
				copyNet := *n
				if !copyNet.compactInput {
					t.Fatal("fixture unexpectedly requires wide storage")
				}
				copyNet.compactInput = !wide
				e := ngnn1NewWorker(t, &copyNet, p)
				assertResidualParity(t, e, p, c, n == nonzero)
			}
		}
	}
	if len(coverage) != 32 || changed < 24 {
		t.Fatalf("insufficient coverage or nonzero route control: coverage=%d changed=%d", len(coverage), changed)
	}
	t.Logf("32 NumPy/Go integer cases, eight buckets x mirrors x STM; %d nonzero scores differ from NGNN3; compact/wide", changed)
}

func TestWave2ResidualParityTransitions(t *testing.T) {
	var packet residualParityPacket
	readResidualParity(t, "parity.json", &packet)
	zero := loadResidualParity(t, "zero-h32.ngn6", packet.Output.SHA256, 6)
	nonzero := loadResidualParity(t, "nonzero-h32.ngn6", packet.Nonzero.SHA256, 6)
	var captures, castles, promotions, epMoves, nulls, mirrors, buckets int
	for _, n := range []*NGNN1Network{zero, nonzero} {
		for _, wide := range []bool{false, true} {
			copyNet := *n
			copyNet.compactInput = !wide
			for _, c := range packet.Transitions {
				p, err := ParseFEN(c.FEN)
				if err != nil {
					t.Fatal(err)
				}
				e := ngnn1NewWorker(t, &copyNet, p)
				assertResidualParity(t, e, p, c.Before, n == nonzero)
				if p.IsInCheck() {
					t.Fatal("null fixture starts in check")
				}
				trNull := e.mustPrepareNull(p)
				ep := p.MakeNullMove()
				e.mustPushMadeNull(p, trNull, ep)
				assertResidualParity(t, e, p, c.Null, n == nonzero)
				e.mustPop()
				p.UnMakeNullMove(ep)
				nulls++
				assertResidualParity(t, e, p, c.Before, n == nonzero)
				move := EmptyMove
				for _, candidate := range GenerateLegalMoves(p) {
					if candidate.ToString() == c.Move {
						move = candidate
					}
				}
				if move == EmptyMove {
					t.Fatal("missing legal move: " + c.Name + " " + c.Move)
				}
				if move.CapturedPiece() != NoPiece {
					captures++
				}
				if move.IsCastle() {
					castles++
				}
				if move.IsEnPassant() {
					epMoves++
				}
				if move.PromoType() != NoType {
					promotions++
				}
				if move.MovingPiece().Type() == King {
					before := n.kingContext(p, p.Turn())
					after := n.contextAt(move.Destination(), p.Turn())
					if before.mirror != after.mirror {
						mirrors++
					}
					if before.bucket != after.bucket {
						buckets++
					}
				}
				tr := e.mustPrepareMove(p, move)
				ep, tag, clock, _ := p.MakeMove(move)
				e.mustPushMadeMove(p, tr, move, tag, ep, clock)
				assertResidualParity(t, e, p, c.After, n == nonzero)
				after, err := ParseFEN(c.After.FEN)
				if err != nil || after.Board.mailbox != p.Board.mailbox || after.Turn() != p.Turn() {
					t.Fatal("transition board disagrees with Python input")
				}
				e.mustPop()
				p.UnMakeMove(move, tag, ep, clock)
				assertResidualParity(t, e, p, c.Before, n == nonzero)
			}
		}
	}
	if captures == 0 || castles == 0 || promotions == 0 || epMoves == 0 || nulls == 0 || mirrors == 0 || buckets == 0 {
		t.Fatal("missing special transition coverage")
	}
	t.Logf("NumPy/Go special transitions: captures=%d castles=%d promotions=%d ep=%d nulls=%d mirrors=%d buckets=%d", captures, castles, promotions, epMoves, nulls, mirrors, buckets)
}

// The explicit scratch model is optional in the short suite; the experiment
// command supplies it and requires four exact production depth-four searches.
func TestWave2ResidualParityH768Search(t *testing.T) {
	path := os.Getenv("NGN_RESIDUAL_H768")
	if path == "" {
		t.Skip("run residual-convert/run.py h768 with an external scratch directory")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lift, err := ReadNGNN1(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	base, err := LoadDefaultNet()
	if err != nil || base.version != 3 || base.hidden != 768 || lift.version != 6 || lift.hidden != 768 {
		t.Fatal("expected accepted H768 NGNN3 and its NGNN6 lift")
	}
	if !bytes.Equal(raw[8:96], defaultNetBytes[8:96]) || !bytes.Equal(raw[120:120+len(defaultNetBytes)-100], defaultNetBytes[96:len(defaultNetBytes)-4]) {
		t.Fatal("H768 inherited bytes changed")
	}
	for _, w := range lift.residual.output {
		if w != 0 {
			t.Fatal("expected zero projection")
		}
	}
	for _, b := range lift.residual.outputBias {
		if b != 0 {
			t.Fatal("expected zero residual bias")
		}
	}
	integerData, err := os.ReadFile(os.Getenv("NGN_RESIDUAL_INTEGER_RECEIPT"))
	if err != nil {
		t.Fatal(err)
	}
	var integerReceipt struct {
		Cases []struct {
			FEN             string
			Bucket, STM, CP int
		} `json:"integer_parity"`
	}
	if err := json.Unmarshal(integerData, &integerReceipt); err != nil || len(integerReceipt.Cases) != 32 {
		t.Fatal("missing H768 NumPy integer receipt")
	}
	for _, c := range integerReceipt.Cases {
		p, err := ParseFEN(c.FEN)
		if err != nil {
			t.Fatal(err)
		}
		stm := 0
		if p.Turn() == Black {
			stm = 1
		}
		for _, n := range []*NGNN1Network{base, lift} {
			for _, wide := range []bool{false, true} {
				copyNet := *n
				copyNet.compactInput = !wide
				e := ngnn1NewWorker(t, &copyNet, p)
				if e.SearchSTM(p) != c.CP || n.bucket(p) != c.Bucket || stm != c.STM {
					t.Fatalf("H768 NumPy/Go mismatch: %s score=%d want=%d bucket=%d want=%d stm=%d want=%d", c.FEN, e.SearchSTM(p), c.CP, n.bucket(p), c.Bucket, stm, c.STM)
				}
			}
		}
	}
	t.Log("32 actual H768 NumPy/Go integer records through compact/wide paths")
	var fens []string
	readResidualParity(t, "search-fens.json", &fens)
	if len(fens) != 4 {
		t.Fatal("exactly four search inputs required")
	}
	type searchReceipt struct {
		FEN, BestMove, PV string
		Depth, Score      int
		Nodes             uint64
	}
	var receipts []searchReceipt
	for _, fen := range fens {
		var pair []searchReceipt
		for _, n := range []*NGNN1Network{base, lift} {
			s := NewSearchEngine()
			if err := s.ResizeHash(1); err != nil {
				t.Fatal(err)
			}
			if err := s.SelectNGNN1Evaluator(n); err != nil {
				t.Fatal(err)
			}
			if s.SelectedEvaluatorBackend() != n.backendName() {
				t.Fatal("wrong search backend")
			}
			p, err := ParseFEN(fen)
			if err != nil {
				t.Fatal(err)
			}
			info := s.SearchIterativeDeepeningWithCallback(p, 4, nil, func(*SearchInfo) {})
			if info.Depth != 4 || info.Stopped || info.PVLength == 0 {
				t.Fatal("search did not complete depth four")
			}
			pv := ""
			for _, m := range info.PV[:info.PVLength] {
				pv += m.ToString() + " "
			}
			pair = append(pair, searchReceipt{fen, info.BestMove.ToString(), pv, info.Depth, info.BestScore, info.Nodes})
		}
		if pair[0] != pair[1] {
			t.Fatalf("H768 search differs: %+v vs %+v", pair[0], pair[1])
		}
		receipts = append(receipts, pair[0])
		t.Logf("exact H768/NGNN6 search: %+v", pair[0])
	}
	result := struct {
		InputSHA256, OutputSHA256 string
		InputFormat, OutputFormat string
		Searches                  []searchReceipt
	}{DefaultNetSHA256, fmt.Sprintf("%x", sha256.Sum256(raw)), "NGNN3", "NGNN6", receipts}
	if output := os.Getenv("NGN_RESIDUAL_SEARCH_RECEIPT"); output != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
