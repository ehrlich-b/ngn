# NGN Chess Engine - Development Roadmap

## Goal: 2500+ ELO (Grandmaster Level)

**Current Status:** ~1659 ELO, 95% CI [1468, 1851] — 14-game asymmetric (NGN 2500ms vs SF 1500 @ 200ms, E-cores). 10W-0D-4L = 71.4%.

---

## Recent Fixes (2026-04-26)

### Hash Corruption on UnMakeMove (FIXED)
- `unMakeMoveHelper` mutated `p.Tag`/`p.EnPassant` before reversing the Zobrist hash, so the XOR undo used the wrong state.
- Fix: call `updateHash` symmetrically before reverting state. XOR is its own inverse.
- Regression test: `engine/hash_makeunmake_test.go` walks 6 FENs at depth 3 confirming `pos.Hash()` returns to its pre-make value after every UnMakeMove.

### countPassedPawnsFast Returned Phantom Passed Pawns (FIXED)
- The "all ranks ahead" mask used `uint64(0xFF) << ((sq/8 + 1) * 8)` which sets ONE rank, not all ranks. Black-side mask was always 0 for most pawns. Per-pawn rank masks in `opponentControl` had the same single-rank bug despite the comment claiming otherwise.
- Effect: starting position reported 8 passed pawns each side (correct: 0). Worth roughly +20cp × phantom passes per eval call.
- Fix: precompute `passedPawnFrontSpan[color][sq]` table in init() — file plus adjacent files at all ranks ahead.
- Regression test: `engine/passed_pawn_compare_test.go` compares slow vs fast counts across 8 positions.
- ELO impact: ~+160 ELO over post-hash baseline (CI lower bound 1468 vs prior ~1398).

### Aspiration Windows Treated Completed Iterations as Fail-Low (FIXED)
- Root loop raised `alpha` to the best score during the move loop, then classified the iteration with `iterationBestScore <= alpha` — which always fires because alpha was just raised to iterationBestScore. Past depth 3, every successful iteration was marked fail-low, attempts exhausted, but `info.Depth` and the UCI callback fired with stale data.
- Fix: snapshot `windowAlpha`/`windowBeta` before the move loop and classify against the snapshot. Only update `info.Depth` and call the callback when the iteration actually completed (`engine/search.go`).

### Quiescence Stand-Pat in Check (FIXED) + Quiet Checks Dropped
- Qsearch unconditionally stand-patted even when side-to-move was in check.
- Fix: at qsearch entry, compute `inCheck`. If in check, search ALL legal evasions (no stand-pat); return mate score if no legal evasion. If not in check, stand-pat + captures only (`engine/search.go`).
- Also dropped quiet checks from qsearch and removed `moveGivesCheck` — it missed discovered/EP/promotion checks, and incomplete-quiet-checks is worse than captures-only at the horizon (`engine/movegen.go`).

### TestIterativeDeepening Determinism (FIXED)
- Test failed reproducibly with `ID: b1c3, Fixed: g1f3` because global state (TT, history, killers, counters) carried between the two searches.
- Fix: reset global search state before each search call inside the test (`engine/search_test.go`).

### UCI `stop` Wired Into Recursive Search (FIXED)
- Stop was only checked in the depth-completion callback. `go infinite` + `stop` hung until max depth.
- Fix: added a global atomic stop flag (`globalStopRequested` in `engine/search.go`). `RequestStop()` is called by `handleStop`; `IsStopRequested()` is sampled at the top of `alphaBetaPV` and `quiescenceWithDepth` and at every iterative-deepening depth boundary. Cleared at search entry.

### ShouldStopSearch Iteration-Start (FIXED)
- All callers passed `time.Now()` where iteration-start was expected, making `iterationTime ≈ 0` and disabling soft-stop.
- Fix: TimeManager owns `iterationStartTime` internally. `NewIteration()` sets it at the top of each iterative-deepening depth. The legacy parameter is now ignored. Soft-stop falls back to "don't stop" if `iterationStartTime` was never set (`engine/time.go`).

### Test Hygiene (PARTIAL)
- Replaced `Rook_vs_Bishop_Knight` FEN (`8/8/8/8/8/8/2R5/2BNK2k`, which was white R+B+N vs lone king) with a real R-vs-B+N endgame.
- Toned down the misleading `CRITICAL ERROR` wording in `free_piece_test.go` — the test never failed and the position isn't a clean free-piece test at depth.

### Threefold Repetition Detection (FIXED)
- `handlePosition` used `MakeMove` instead of `GameMakeMove` — position history wasn't tracked.
- Root search didn't check `IsFIDEDrawRule()` before returning moves.

---

## Remaining Known Issues

These are deferred from the 2026-04-26 batch. Lower priority; do not block strength gains.

### Pri 5 follow-up: discovered/EP/promotion check detection
- Currently qsearch is captures-only (no quiet checks). If qsearch quiet-check coverage is desired later, implement a proper discovered-check detector or use make/unmake + `isInCheck`. Don't reintroduce the broken `moveGivesCheck` direct-attack-only path.

### Pri 6: External Polyglot books silently broken
- `engine/polyglot.go:33-55` — `polyglotRandom[781]` is generated from xorshift seed=1 instead of the spec-defined constants. Verified mismatch: `PolyglotHash` of starting position yields `0x4b9aa9e5e768fe35`, spec says `0x463b96181691fc9c`.
- A real `.bin` polyglot book will never match positions. Embedded book still works.
- Fix: embed the official 781 constants (large literal table) and add a starting-position hash check.

### Test hygiene (cosmetic)
- `engine/play_quality_test.go` (`TestEnginePlayingStrength`) compares `Move.ToString()` UCI output (`b1c3`, `e1g1`) to expected SAN (`Nc3`, `O-O`). Engine plays the right move, test marks it neutral. Most outcomes never trigger `t.Errorf`, so this is misleading rather than blocking. Fix: store expected moves in UCI or parse SAN.
- `engine/crash_stress_test.go:250-278` (`TestConcurrencyStress`) shares one mutable `*Position` across goroutines. Fix: copy per-goroutine, or document non-thread-safety of `Position`.

---

## Assessment Strategy

### Default: `make smoke` (~5 min, E-cores, battery-friendly)
Runs a 30-position tactical suite plus a 5-game asymmetric match (NGN 1s vs
SF1500 200ms). Appends results to `output/smoke_history.jsonl`. Prints a
rolling Wilson CI that tightens with every run. Fail conditions: tactical
solve rate < 40% or 0/5 game outcome.

```bash
make smoke              # Default cadence
make smoke-tactical     # Tactical only (~5 sec) — for in-flight changes
./build/smoke -rolling  # Show rolling estimate without running
```

The rolling estimate is the source of truth for current ELO, not any single
smoke. After 6-10 smokes at the default config the cumulative CI is tighter
than a single 14-game Bayesian and you've burned an order of magnitude less
battery.

### Full Bayesian (AC power only, ~15-30 min)
```bash
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```
Use when you need a tight one-shot CI before a major decision.

### Legacy Presets (hotter, mostly historical)
```bash
./build/elo-assess -level 1   # 8 games @ 250ms
./build/elo-assess -level 5   # 20 games @ 400ms
./build/elo-assess -level 10  # 40 games @ 600ms
```

---

## Verified Implementation Status

### Search Techniques
- [x] Iterative deepening
- [x] Aspiration windows
- [x] Alpha-beta with transposition table
- [x] Null move pruning
- [x] Futility pruning (extended, depth 1-3)
- [x] Reverse futility pruning
- [x] Late Move Reduction (LMR)
- [x] Late Move Pruning (LMP)
- [x] Internal Iterative Deepening (IID)
- [x] SEE (Static Exchange Evaluation)
- [x] SEE pruning for captures
- [x] History heuristic
- [x] History pruning
- [x] Killer moves (2 per ply)
- [x] Counter move heuristic
- [x] Probcut pruning
- [x] Quiescence search (in-check evasions; captures-only — quiet checks intentionally dropped)

### Extensions (WORKING)
- [x] Check extensions
- [x] Singular extensions
- [x] Recapture extensions
- [x] Passed pawn extensions

### Evaluation (COMPLETE)
- [x] Material
- [x] Piece-square tables
- [x] King endgame PST (different from middlegame)
- [x] Doubled pawns penalty
- [x] Isolated pawns penalty
- [x] Passed pawns bonus
- [x] Pawn chains bonus
- [x] Pawn storms
- [x] Weak squares
- [x] Bishop pair bonus
- [x] Mobility evaluation (knights, bishops, rooks, queens)
- [x] King safety (pawn shield, open files)
- [x] Knight outposts
- [x] Piece coordination
- [x] Rook on open file (explicit bonus)
- [x] Tapered eval (MG/EG interpolation for king PST)

### Infrastructure
- [x] Opening book (embedded works; external Polyglot broken — see Remaining Known Issues)
- [x] Syzygy tablebase support
- [x] UCI protocol (`stop` wired into recursive search via global atomic flag)
- [x] Time management (`TimeManager.NewIteration` drives soft-stop)

---

## Performance Metrics (Verified)

| Metric | Value | Notes |
|--------|-------|-------|
| NPS | ~350K | At depth 10, middle game position |
| Depth/5s | 10 | From complex position |
| Depth/1s | ~7-8 | Estimated |

---

## Development Phases

### Phase 1: Critical Bug Fixes (COMPLETED)
1. ~~Fix threefold repetition detection~~ ✓
2. ~~Verify game completion works~~ ✓
3. ~~Run baseline ELO assessment~~ ✓ (~1400-1500 ELO after Phase 2)

### Phase 2: Evaluation Features (COMPLETED)
All classical evaluation features now implemented:
- ✓ Bishop pair bonus
- ✓ Mobility evaluation
- ✓ King safety
- ✓ Knight outposts
- ✓ Piece coordination
- ✓ Rook on open file (evaluateRookOnOpenFile)
- ✓ Tapered eval (gamePhase + taperScore for king PST)

### Phase 3: Search Correctness (LARGELY COMPLETE)
- ✓ Aspiration windows fail-low past depth 3 (Pri 0)
- ✓ Quiescence in-check evasions + drop quiet checks (Pri 1 + Pri 5)
- ✓ `TestIterativeDeepening` deterministic (Pri 4)
- ✓ UCI `stop` wired into recursive search (Pri 2)
- ✓ `ShouldStopSearch` iteration-start tracking (Pri 3)
- Deferred: SAN→UCI in `TestEnginePlayingStrength`, Polyglot constants (Pri 6)

### Phase 4: ELO Validation + Tuning (CURRENT)
1. Build the rolling-ELO baseline by running `make smoke` after each change. Per-run Wilson CI is wide (~700+ ELO at 5 games), but the cumulative across 6-10 smokes converges below 300.
2. Tune evaluation/search parameters one knob at a time, gating each with `make smoke`. Revert anything that regresses tactically or drives the cumulative ELO down.
3. When you need a publishable single-shot CI (e.g., before claiming a milestone), run a full 14-game Bayesian on AC power.
4. Optional: address remaining test hygiene + Polyglot constants if external book support is wanted.

---

## Reference Engines (Go)

### Blunder (Go, ~2400 ELO)
- AdaGrad tuning for evaluation
- Queen promotions in quiescence
- Pawn pushes to 6th+ not reduced in LMR

### Zurichess (Go, ~2500 ELO)
- Reached GM level with classical eval
- King-queen tropism
- Hashing in quiescence search
- Razoring

---

## Is GM Level Hard in 2025?

**No.** Classical engines (no NNUE) have reached 2500+ ELO:
- Crafty (~2600)
- Fruit (~2700 in 2005)
- Stockfish pre-NNUE (~3000)

The techniques are well-documented. The challenge is:
1. Getting all the pieces working together correctly
2. Avoiding bugs in complex interactions
3. Proper tuning of parameters

NNUE adds ~300-400 ELO on top of a strong classical base, but is NOT required for GM level.

---

## Commands Reference

```bash
make build           # Build engine
make test            # Run tests with coverage
make test-short      # Run tests without coverage (faster)
make smoke           # DEFAULT verification: tactical + 5-game smoke + rolling ELO (~5 min)
make smoke-tactical  # Tactical-only smoke (~5 sec)

./build/smoke -rolling   # Print rolling-ELO summary without running

# Full Bayesian ELO test (AC power only):
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

---

## Notes

- Run `make test` after every change
- Verify games complete before trusting ELO numbers
- One bug fix at a time, test between each
- Track ELO progress with assessment tool
