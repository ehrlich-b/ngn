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

### Threefold Repetition Detection (FIXED)
- `handlePosition` used `MakeMove` instead of `GameMakeMove` — position history wasn't tracked.
- Root search didn't check `IsFIDEDrawRule()` before returning moves.

---

## Critical Search Bugs (Confirmed 2026-04-26 by external review + code inspection)

These are pre-existing bugs not yet fixed. They likely cap strength regardless of further evaluation tuning. Fix in priority order before tuning anything.

### Pri 0: Aspiration windows treat completed iterations as fail-low past depth 3
- `engine/search.go:509-532` — root loop raises `alpha` to the best score during the move loop, then classifies the iteration with `iterationBestScore <= alpha` (line 520).
- Because `alpha` was just raised to `iterationBestScore`, the check fires for normal completed searches at depth > 3, marking them fail-low.
- The engine then exhausts `maxAttempts` without accepting the new result, but still sets `info.Depth = currentDepth` (line 537) and emits UCI info with the previous depth's best move.
- Symptom: deep search announced, shallow play actually used. Plausibly worth >100 ELO.
- Fix: snapshot `windowAlpha`/`windowBeta` before the loop and classify against the original window, not the mutated `alpha`. Only set `info.Depth` and call the callback when the iteration actually completed.
- Quick test: disable aspiration windows entirely (use `[-INFINITY, INFINITY]`) and re-run the asymmetric ELO test. If ELO jumps, this confirms the bug's impact.

### Pri 1: Quiescence allows stand-pat while in check
- `engine/search.go:1319-1325` — qsearch unconditionally evaluates stand-pat even when side-to-move is in check.
- In check, the side must make an evasion; stand-pat causes false beta cutoffs and missed tactics at the horizon.
- Fix: at qsearch entry, compute `inCheck := isInCheck(pos, pos.Turn())`. If in check: skip stand-pat, generate all legal evasions (not just captures/checks), and return mate score if no legal evasion exists.

### Pri 2: UCI `stop` does not stop recursive search
- `engine/uci.go:626-633` — `stopChan` is checked only inside the depth-completion callback, not in `alphaBetaPV`/qsearch.
- `engine/time.go:281-282` — `Infinite` mode explicitly never stops (`tm.shouldStop = false`).
- Effect: GUI `go infinite` + `stop` hangs until max depth or hard timeout.
- Fix: add a stop flag/context to `SearchInfo`, have `handleStop` set it, check it in the recursive search root loops and qsearch. Don't rely on the depth callback.

### Pri 3: ShouldStopSearch always sees iterationTime ≈ 0
- `engine/search.go:423,463,477,599,661,933` — every caller passes `time.Now()` where iteration-start is expected.
- `engine/time.go` (Tournament path) computes `iterationTime := time.Since(iterationStartTime)` for soft-stop and `nextIterationEstimate := iterationTime * 4`. With `time.Now()` as the argument, `iterationTime` is ~0 and the soft-stop logic is dead.
- Fix: record `iterationStart := time.Now()` once at the top of each iterative-deepening depth and thread it through, or remove the parameter and let `TimeManager` own iteration-start state.

### Pri 4: TestIterativeDeepening fails — ID and fixed-depth disagree at depth 3
- `engine/search_test.go` — fails reproducibly: `ID: b1c3, Fixed: g1f3` at depth 3.
- This is below the aspiration-window depth>3 threshold, so it points at a separate instability: TT reuse, root handling, move-ordering state, or node-type storage/probing.
- Fix path: (1) reset global state (`TranspositionTable`, history/killer/counter) inside the test; (2) if it still fails, instrument root move scores at fixed depth and ID depth 3 to find the first divergence.
- Do not tune evaluation while this fails — it indicates non-deterministic search.

### Pri 5: moveGivesCheck misses discovered/EP/promotion checks
- `engine/movegen.go:869-935` — only tests whether the moving piece attacks the king from its destination square.
- Misses: discovered checks (mover unblocks an attacker), en-passant discovered checks, promotion checks (Pawn case checks pawn attacks from destination, not the promoted piece's attacks).
- Used by qsearch to add quiet checks. Discovered checks are exactly the quiet tactical moves qsearch is supposed to include.
- Fix options: (a) make/unmake candidate quiet checks and call `isInCheck`; (b) implement a proper discovered-check detector; (c) until fixed, drop quiet checks from qsearch — captures-only is better than incomplete-quiet-checks.

### Pri 6: External Polyglot books are silently broken
- `engine/polyglot.go:33-55` — `polyglotRandom[781]` is generated from xorshift seed=1 instead of the spec-defined constants.
- A real `.bin` polyglot book will never match positions. Embedded book still works; UCI falls back to it.
- Test currently skipped: `engine/book_test.go:72-78`.
- Fix: embed the official 781 constants and add a starting-position hash check against `0x463b96181691fc9c`.

### Test suite hygiene (lower priority, but blocks ELO signal)
- `engine/play_quality_test.go` (`TestEnginePlayingStrength`) compares `Move.ToString()` UCI output (`b1c3`, `e1g1`) to expected SAN (`Nc3`, `O-O`). Engine plays the right move, test marks it neutral. Fix: store expected moves in UCI or parse to `Move` and compare moves not strings.
- Same file: `Rook_vs_Bishop_Knight` FEN `8/8/8/8/8/8/2R5/2BNK2k w - - 0 1` is white R+B+N vs lone king — not "roughly equal". Fix the FEN or remove the assertion.
- `engine/free_piece_test.go` (`TestFreePieceCapture`) logs "CRITICAL ERROR" but never `t.Fail`s; the test position isn't a clean free-pawn at deeper depths.
- `engine/crash_stress_test.go:250-278` (`TestConcurrencyStress`) shares one mutable `*Position` across goroutines — `GenerateLegalMoves` make/unmakes on it, so the failure is expected. Either copy per-goroutine or document non-thread-safety.

---

## Assessment Strategy

### Standard Cadence (15-30 min, E-cores, depth 6-7 reached)
```bash
./build/elo-assess -elo 1500 -movetime 2500 -min 6 -games 14 -ci 300
```
Asymmetric (`-sfmovetime 200`) is the variant used to verify the recent +160 ELO win.
14 games with Bayesian early-stop. CI width ~300-400 ELO — detects ±80 ELO swings.

### Quick Smoke Test (3-5 min, depth 4 only)
```bash
./build/elo-assess -elo 1320 -movetime 200 -min 6 -games 14 -ci 300
```

### Legacy Presets (hotter)
```bash
./build/elo-assess -level 1   # 8 games @ 250ms
./build/elo-assess -level 5   # 20 games @ 400ms
./build/elo-assess -level 10  # 40 games @ 600ms
```

---

## Verified Implementation Status

### Search Techniques (implemented — but see "Critical Search Bugs" above)
- [x] Iterative deepening — works, but ID disagrees with fixed-depth (Pri 4)
- [~] Aspiration windows — broken past depth 3 (Pri 0)
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
- [~] Quiescence search — illegal stand-pat in check (Pri 1); incomplete quiet-check detection (Pri 5)

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

### Infrastructure (implemented — but see "Critical Search Bugs" above)
- [x] Opening book (embedded works; external Polyglot broken — Pri 6)
- [x] Syzygy tablebase support
- [~] UCI protocol — `stop` not wired into recursive search (Pri 2)
- [~] Time management — `ShouldStopSearch` iteration-start argument is dead (Pri 3)

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

### Phase 3: Search Correctness (CURRENT)
Fix the seven "Critical Search Bugs" listed above before any tuning. Recommended order from review:
1. Aspiration windows (Pri 0) — likely largest single ELO win
2. Quiescence in-check evasions (Pri 1)
3. Make `TestIterativeDeepening` deterministic (Pri 4)
4. Wire UCI `stop` into recursive search (Pri 2)
5. Fix `ShouldStopSearch` iteration-start (Pri 3)
6. Repair misleading "quality" tests (notation mismatches, bad FENs) before treating ELO as signal
7. moveGivesCheck completeness (Pri 5) — or drop quiet checks from qsearch
8. Polyglot constants (Pri 6) — non-blocking for strength

### Phase 4: Tuning (after Phase 3 complete)
1. Run Texel tuning on evaluation parameters
2. Tune search parameters
3. Re-assess ELO after each change

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
make build       # Build engine
make test        # Run tests with coverage
make test-short  # Run tests without coverage (faster; bot review notes some non-short tests fail)
make eval        # 2-minute comprehensive eval (use sparingly)

# Asymmetric Bayesian ELO test (preferred verification cadence):
./build/elo-assess -elo 1500 -movetime 2500 -sfmovetime 200 -min 6 -games 14 -ci 300
```

---

## Notes

- Run `make test` after every change
- Verify games complete before trusting ELO numbers
- One bug fix at a time, test between each
- Track ELO progress with assessment tool
