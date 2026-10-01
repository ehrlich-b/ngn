package engine

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/ehrlich-b/ngn/nnue/ngnk4"
)

const (
	ownedEnPassantNNUEEnv = "NGN_EP_OWNED_NNUE"
	ownedEnPassantNNUESHA = "1ec8fc1737ddfdd5f8b6ff0b4e26778085fb563f7c5e82c1fc5d3ea454b6ec29"
	ownedEnPassantK4Scale = 60
)

type ownedEnPassantSnapshot struct {
	position pvPositionSnapshot
	fen      string
	key      uint64
}

type ownedEnPassantUndo struct {
	move   Move
	ep     Square
	tag    PositionTag
	clock  uint8
	before ownedEnPassantSnapshot
}

func loadOwnedEnPassantModel(t *testing.T) *ngnk4.Model {
	t.Helper()
	path := os.Getenv(ownedEnPassantNNUEEnv)
	if path == "" {
		t.Skipf("set %s to the released owned WDL25 NGN K4 model", ownedEnPassantNNUEEnv)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fileSHA := fmt.Sprintf("%x", sha256.Sum256(data))
	if fileSHA != ownedEnPassantNNUESHA {
		t.Fatalf("model SHA-256 = %s, want %s", fileSHA, ownedEnPassantNNUESHA)
	}
	model, err := ngnk4.Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("load exact owned model: %v", err)
	}
	metadata := model.Metadata()
	if metadata.ArchitectureID != ngnk4.ArchitectureID ||
		fmt.Sprintf("%x", metadata.FileSHA256) != ownedEnPassantNNUESHA {
		t.Fatalf("loaded model identity = %q/%x, want %q/%s",
			metadata.ArchitectureID, metadata.FileSHA256, ngnk4.ArchitectureID, ownedEnPassantNNUESHA)
	}
	return model
}

func newOwnedEnPassantSearcher(t *testing.T, model *ngnk4.Model) *SearchEngine {
	t.Helper()
	searcher := NewSearchEngine()
	if err := searcher.SetK4EvalScale(ownedEnPassantK4Scale); err != nil {
		t.Fatal(err)
	}
	if err := searcher.SelectNGNK4Evaluator(model); err != nil {
		t.Fatal(err)
	}
	identity := searcher.worker.evaluator.Identity()
	if identity.backend != evaluatorBackendNGNK4 || identity.ngnK4ScalePercent != ownedEnPassantK4Scale {
		t.Fatalf("evaluator identity = %+v, want NGN K4 scale %d", identity, ownedEnPassantK4Scale)
	}
	return searcher
}

func snapshotOwnedEnPassantPosition(pos *Position) ownedEnPassantSnapshot {
	key := pos.Hash()
	return ownedEnPassantSnapshot{position: snapshotPVPosition(pos), fen: GenerateFEN(pos), key: key}
}

func assertOwnedEnPassantRestored(t *testing.T, pos *Position, want ownedEnPassantSnapshot) {
	t.Helper()
	raw := snapshotPVPosition(pos)
	if !reflect.DeepEqual(raw, want.position) {
		t.Fatalf("raw position not restored before hashing:\n got=%+v\nwant=%+v", raw, want.position)
	}
	got := snapshotOwnedEnPassantPosition(pos)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("position not exactly restored:\n got=%+v\nwant=%+v", got, want)
	}
}

func ownedEnPassantLegalMove(t *testing.T, pos *Position, text string) Move {
	t.Helper()
	for _, move := range GenerateLegalMoves(pos) {
		if move.ToString() == text {
			return move
		}
	}
	t.Fatalf("%s is not legal in %s (legal %v)", text, GenerateFEN(pos), legalMoveStrings(pos))
	return EmptyMove
}

func requireOwnedK4FreshParity(t *testing.T, model *ngnk4.Model, worker *workerEvaluator, pos *Position) {
	t.Helper()
	wantPosition, err := ngnK4PositionFromPosition(pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := worker.ngnK4Context.Position(); got != wantPosition {
		t.Fatal("incremental NGN K4 context position differs from the engine position")
	}
	incrementalRaw, err := worker.ngnK4Context.EvaluateRaw()
	if err != nil {
		t.Fatal(err)
	}
	freshSearcher := newOwnedEnPassantSearcher(t, model)
	fresh := freshSearcher.worker.evaluator
	if err := fresh.Reset(pos); err != nil {
		t.Fatal(err)
	}
	freshRaw, err := fresh.ngnK4Context.EvaluateRaw()
	if err != nil {
		t.Fatal(err)
	}
	if incrementalRaw != freshRaw {
		t.Fatalf("incremental raw evaluation = %d, separately refreshed = %d", incrementalRaw, freshRaw)
	}
	if incrementalScore, freshScore := worker.SearchSTM(pos), fresh.SearchSTM(pos); incrementalScore != freshScore {
		t.Fatalf("incremental scale-%d search evaluation = %d, separately refreshed = %d",
			ownedEnPassantK4Scale, incrementalScore, freshScore)
	}
	if fresh.nnueDepth() != 0 {
		t.Fatalf("fresh evaluator depth = %d, want 0", fresh.nnueDepth())
	}
}

func requireOwnedK4DirectParity(t *testing.T, model *ngnk4.Model, worker *workerEvaluator, pos *Position) {
	t.Helper()
	wantPosition, err := ngnK4PositionFromPosition(pos)
	if err != nil {
		t.Fatal(err)
	}
	if got := worker.ngnK4Context.Position(); got != wantPosition {
		t.Fatal("search's incremental NGN K4 context position differs from the engine position")
	}
	got, err := worker.ngnK4Context.EvaluateRaw()
	if err != nil {
		t.Fatal(err)
	}
	want, err := model.EvaluateRaw(wantPosition)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("search incremental raw evaluation = %d, full-position evaluation = %d", got, want)
	}
}

func TestOwnedK4EnPassantRepetitionReadiness(t *testing.T) {
	model := loadOwnedEnPassantModel(t)
	t.Run("played histories and inverse paths", func(t *testing.T) {
		for _, fixture := range enPassantRepetitionFixtures {
			t.Run(fixture.name, func(t *testing.T) {
				pos := mustParseEnPassantPosition(t, fixture.startFEN)
				root := snapshotOwnedEnPassantPosition(pos)
				searcher := newOwnedEnPassantSearcher(t, model)
				worker := searcher.worker.evaluator
				if err := worker.Reset(pos); err != nil {
					t.Fatal(err)
				}
				requireOwnedK4FreshParity(t, model, worker, pos)

				sequence := []string{fixture.push, fixture.epMove}
				if fixture.illegalEnPassant {
					sequence = append([]string{fixture.push}, fixture.cycle...)
					sequence = append(sequence, fixture.cycle...)
				}
				undos := make([]ownedEnPassantUndo, 0, len(sequence))
				for ply, text := range sequence {
					move := ownedEnPassantLegalMove(t, pos, text)
					transition, err := worker.PrepareMove(pos, move)
					if err != nil {
						t.Fatal(err)
					}
					undo := ownedEnPassantUndo{move: move, before: snapshotOwnedEnPassantPosition(pos)}
					var ok bool
					undo.ep, undo.tag, undo.clock, ok = pos.GameMakeMove(move)
					if !ok {
						t.Fatalf("GameMakeMove(%s) rejected", text)
					}
					if err := worker.PushMove(pos, transition); err != nil {
						t.Fatal(err)
					}
					undos = append(undos, undo)
					assertIncrementalEnPassantKeyMatchesFEN(t, "owned played ply", pos)
					requireOwnedK4FreshParity(t, model, worker, pos)

					if ply == 0 {
						assertPinnedOracleStage(t, "owned real push", pos, fixture.wantEPMoves, fixture.wantEPPerft2, fixture.wantPolyglotEP)
						epFEN := mustParseEnPassantPosition(t, fixture.epFEN)
						noEPFEN := mustParseEnPassantPosition(t, fixture.noEPFEN)
						assertPinnedOracleStage(t, "owned EP FEN", epFEN, fixture.wantEPMoves, fixture.wantEPPerft2, fixture.wantPolyglotEP)
						assertPinnedOracleStage(t, "owned no-EP FEN", noEPFEN, fixture.wantNoEPMoves, fixture.wantNoEPPerft2, fixture.wantPolyglotNoEP)

						original := snapshotOwnedEnPassantPosition(pos)
						copied := pos.Copy()
						assertOwnedEnPassantRestored(t, copied, original)
						copyWorker := newOwnedEnPassantSearcher(t, model).worker.evaluator
						if err := copyWorker.Reset(copied); err != nil {
							t.Fatal(err)
						}
						requireOwnedK4FreshParity(t, model, copyWorker, copied)
						marker := pos.Hash() ^ 0x9e3779b97f4a7c15
						for {
							if _, exists := copied.Positions[marker]; !exists {
								break
							}
							marker++
						}
						copied.Positions[marker] = 7
						if _, leaked := pos.Positions[marker]; leaked {
							t.Fatal("Position.Copy shares its occurrence map with the source")
						}
						delete(copied.Positions, marker)
						assertOwnedEnPassantRestored(t, copied, original)
						assertOwnedEnPassantRestored(t, pos, original)
					}
				}

				if fixture.illegalEnPassant {
					if pos.Positions[pos.Hash()] != 3 || !pos.IsFIDEDrawRule() {
						t.Fatalf("played history did not reach true threefold: count=%d draw=%v",
							pos.Positions[pos.Hash()], pos.IsFIDEDrawRule())
					}
				}
				for index := len(undos) - 1; index >= 0; index-- {
					undo := undos[index]
					if err := worker.Pop(); err != nil {
						t.Fatal(err)
					}
					pos.GameUnMakeMove(undo.move, undo.tag, undo.ep, undo.clock)
					assertOwnedEnPassantRestored(t, pos, undo.before)
					requireOwnedK4FreshParity(t, model, worker, pos)
				}
				assertOwnedEnPassantRestored(t, pos, root)
				if worker.nnueDepth() != 0 {
					t.Fatalf("evaluator depth after full inverse path = %d, want 0", worker.nnueDepth())
				}
			})
		}
	})

	t.Run("null quiet child and inverse", func(t *testing.T) {
		for _, fixture := range enPassantRepetitionFixtures {
			t.Run(fixture.name, func(t *testing.T) {
				pos := mustParseEnPassantPosition(t, fixture.startFEN)
				start := snapshotOwnedEnPassantPosition(pos)
				worker := newOwnedEnPassantSearcher(t, model).worker.evaluator
				if err := worker.Reset(pos); err != nil {
					t.Fatal(err)
				}

				push := ownedEnPassantLegalMove(t, pos, fixture.push)
				pushTransition, err := worker.PrepareMove(pos, push)
				if err != nil {
					t.Fatal(err)
				}
				pushEP, pushTag, pushClock, ok := pos.GameMakeMove(push)
				if !ok {
					t.Fatal("legal double push rejected")
				}
				if err := worker.PushMove(pos, pushTransition); err != nil {
					t.Fatal(err)
				}
				root := snapshotOwnedEnPassantPosition(pos)
				requireOwnedK4FreshParity(t, model, worker, pos)

				nullTransition, err := worker.PrepareNull(pos)
				if err != nil {
					t.Fatal(err)
				}
				nullEP := pos.MakeNullMove()
				if err := worker.PushNull(pos, nullTransition); err != nil {
					t.Fatal(err)
				}
				nullState := snapshotOwnedEnPassantPosition(pos)
				requireOwnedK4FreshParity(t, model, worker, pos)

				var quiet Move
				for _, move := range GenerateLegalMoves(pos) {
					piece := move.MovingPiece()
					if piece != WhitePawn && piece != BlackPawn && !move.IsCapture() && !move.IsCastle() {
						quiet = move
						break
					}
				}
				if quiet == EmptyMove {
					t.Fatal("fixture has no quiet non-pawn null descendant")
				}
				quietTransition, err := worker.PrepareMove(pos, quiet)
				if err != nil {
					t.Fatal(err)
				}
				quietEP, quietTag, quietClock, ok := pos.MakeMove(quiet)
				if !ok {
					t.Fatal("legal quiet null descendant rejected")
				}
				if err := worker.PushMove(pos, quietTransition); err != nil {
					t.Fatal(err)
				}
				requireOwnedK4FreshParity(t, model, worker, pos)

				if err := worker.Pop(); err != nil {
					t.Fatal(err)
				}
				pos.UnMakeMove(quiet, quietTag, quietEP, quietClock)
				assertOwnedEnPassantRestored(t, pos, nullState)
				requireOwnedK4FreshParity(t, model, worker, pos)
				if err := worker.Pop(); err != nil {
					t.Fatal(err)
				}
				pos.UnMakeNullMove(nullEP)
				assertOwnedEnPassantRestored(t, pos, root)
				requireOwnedK4FreshParity(t, model, worker, pos)
				if err := worker.Pop(); err != nil {
					t.Fatal(err)
				}
				pos.GameUnMakeMove(push, pushTag, pushEP, pushClock)
				assertOwnedEnPassantRestored(t, pos, start)
				requireOwnedK4FreshParity(t, model, worker, pos)
				if worker.nnueDepth() != 0 {
					t.Fatalf("evaluator depth after null inverse path = %d, want 0", worker.nnueDepth())
				}
			})
		}
	})

	t.Run("finite search restores caller", func(t *testing.T) {
		for _, fixtureIndex := range []int{0, 1, 4, 5} {
			fixture := enPassantRepetitionFixtures[fixtureIndex]
			t.Run(fixture.name, func(t *testing.T) {
				incrementalPos := mustParseEnPassantPosition(t, fixture.startFEN)
				mustPlayEnPassantMove(t, incrementalPos, fixture.push, true)
				fullRefreshPos := incrementalPos.Copy()
				incrementalBefore := snapshotOwnedEnPassantPosition(incrementalPos)
				fullRefreshBefore := snapshotOwnedEnPassantPosition(fullRefreshPos)

				incremental := newOwnedEnPassantSearcher(t, model)
				fullRefresh := newOwnedEnPassantSearcher(t, model)
				observed := 0
				incremental.worker.evaluator.transitionObserver = func(pos *Position, evaluator *workerEvaluator) {
					observed++
					requireOwnedK4DirectParity(t, model, evaluator, pos)
				}
				fullRefresh.worker.evaluator.fullRefreshOracle = true
				got := incremental.SearchFixed(incrementalPos, 2, nil)
				want := fullRefresh.SearchFixed(fullRefreshPos, 2, nil)
				if observed < 2 {
					t.Fatalf("search transition observer ran %d times, want root and children", observed)
				}
				if gotComparable, wantComparable := comparableN3CSearchInfo(got), comparableN3CSearchInfo(want); !reflect.DeepEqual(gotComparable, wantComparable) {
					t.Fatalf("incremental and full-refresh searches differ:\n incremental=%+v\nfull-refresh=%+v",
						gotComparable, wantComparable)
				}
				assertOwnedEnPassantRestored(t, incrementalPos, incrementalBefore)
				assertOwnedEnPassantRestored(t, fullRefreshPos, fullRefreshBefore)
				if incremental.worker.evaluator.nnueDepth() != 0 || fullRefresh.worker.evaluator.nnueDepth() != 0 {
					t.Fatalf("post-search evaluator depths = %d/%d, want 0/0",
						incremental.worker.evaluator.nnueDepth(), fullRefresh.worker.evaluator.nnueDepth())
				}
				requireOwnedK4FreshParity(t, model, incremental.worker.evaluator, incrementalPos)
				t.Logf("depth-2 search restored caller after %d committed evaluator frames", observed)
			})
		}
	})
}
