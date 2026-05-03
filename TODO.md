# NGN Chess Engine - Development Roadmap

## Goal: 2500+ ELO (Grandmaster Level), with 2200 as the next milestone

**Current Status (2026-05-03):** 1553 ELO rolling-smoke estimate, 95% CI [1422, 1684]
(width 261, commit-decayed sample of 26.5 games across 10 runs at NGN 1000ms vs SF1500
200ms). Climbed from 1477 → 1533 → 1582 over two post-audit-batch smokes (both 4W-1L),
then a tried-and-reverted hanging-pieces experiment slipped it to 1553 (2W-3L). Code
reverted to audit-batch baseline; the regressed data point will decay out as future
smokes accumulate. Stage 1 of the ladder targets 1700.

---

## Recent Fixes (2026-04-26)

### Attack-table king safety (FIXED)
- `evaluateKingAttackPatterns` rewritten to use real piece attack bitboards
  intersected with a king zone (3x3 around king + one-rank extension toward
  the enemy half) instead of just counting opponent pieces in a 5x5 proximity
  zone. Per-piece attack-square hits weighted (N=2, B=2, R=3, Q=5).
  2+ attackers fed through quadratic penalty curve (units²/8 capped at 500cp);
  0-1 attackers get a small linear penalty (units/2). The old version flagged
  pieces NEAR the king as dangerous even when blockaded; the new version only
  credits real threat squares.
- 5W-0D-0L smoke. Rolling ELO 1605 → 1653 (CI lower bound +54 to 1519).

### Mate score TT encoding (FIXED)
- `alphaBetaPV` and `quiescenceWithDepth` now take an explicit `ply` parameter.
- New `scoreToTT(score, ply)` and `scoreFromTT(score, ply)` helpers convert between
  root-relative search scores and node-relative TT-stored scores. Without this, mate
  scores stored at one ply and retrieved via the same hash at a different ply reported
  the wrong distance to mate.
- First attempt failed because `info.Depth - depth` is NOT a valid ply (info.Depth is
  initialized to maxDepth and only updates AFTER each iteration completes). Proper fix
  threads ply through every recursive call site (root entries, IID, null move, probcut,
  singular, LMR, qsearch evasions/captures).

### Qsearch stalemate score (FIXED)
- `quiescenceWithDepth` returned `-500` when stalemated while winning (>+300cp eval).
  Stalemate is a draw regardless of material; replaced with `return 0`.

### Tempo bonus added
- `EvaluateForPlayer` now applies a 10cp bonus to side-to-move (eval.go).
  Standard small classical-eval feature, claimed +10–20 ELO.

### Eval/search micro-optimizations
- `pos.Hash()` cached once at the top of `alphaBetaPV` instead of recomputing in
  the repetition block and TT probe block.
- King-safety pawn lookups hoisted out of the file loop.
- `evaluateMobility` caches `GetWhitePieces`/`GetBlackPieces`.

### Rolling-ELO commit-distance decay
- `cmd/smoke/main.go` now weights historical runs by `git rev-list --count <sha>..HEAD`:
  full weight at distance 0, linear taper to 0.5 at distance 5, then to 0 at distance 15.
  After 15 commits without a smoke, that data point falls off entirely.

### Hash Corruption on UnMakeMove (FIXED)

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

## 2026-05-02 audit batch — verified bugs

Six parallel sub-agent audits of search/eval/movegen/qsearch+SEE/TT+hash/UCI+time produced
findings; the items below were verified against current code. Apply one at a time gated by
`make smoke-tactical`.

### Pri 1: Mate distance pruning uses stale `info.Depth` (CONFIRMED)
- `engine/search.go:916`: `currentDepth := info.Depth - depth` is wrong. `info.Depth` starts
  at `maxDepth` (line 448) and is only updated on iteration completion (line 618). During
  the tree walk it's stale; can even go negative on iteration 2+.
- `engine/search.go:1482`: qsearch checkmate path computes `mateDistance := info.Depth + qDepth`
  with the same staleness; should be `ply`.
- Fix: replace both with `ply` (already threaded after the mate-TT fix).
- Impact: ~5-15 ELO. Mate distance pruning is small even when working; qsearch mate-detection
  inside captures is rare. Cheap fix, no regression risk.

### Pri 2: SEE and delta pruning ignore promotion gain (CONFIRMED)
- `engine/search.go:121`: `gain[depth] = int(capturedValue)` — initial SEE gain misses the
  promotion bonus. For Pxf8=Q the SEE thinks gain is rook (~500) when it's rook + queen-pawn
  (~1300). Causes SEE pruning to skip winning promotion-captures and move ordering to misrank.
- `engine/search.go:1534`: delta pruning has the same flaw — `gain := move.CapturedPiece().seeWeight()`
  doesn't add promo bonus, so legitimately winning promo-captures get pruned.
- Fix: when `move.PromoType() != NoType`, add `GetPiece(promoType, color).Weight() - PawnValue`
  to gain at both call sites.
- Impact: ~5-15 ELO. Promotion-captures are rare in middlegame, more impactful in endgames
  with passed pawns.

### Pri 3: Singular extension reduction is too aggressive (CONFIRMED)
- `engine/search.go` singular search uses `depth/2`; at depth 6 that's a depth-3 verification.
  Standard practice is `depth - 4` or `depth - 3` (Stockfish, Ethereal). Current setting
  misses real singular moves.
- Fix: replace `depth/2` with `max(1, depth-4)`.
- Impact: ~10-15 ELO from extra extensions at correct depths.

### Pri 4: Qsearch discards TT move + searches captures unordered (CONFIRMED)
- `engine/search.go:1426`: TT.Get returns ttMove but qsearch underscores it.
- `engine/search.go:1530`: captures looped in raw `GenerateCapturesIntoBuffer` order; no
  SEE/MVV-LVA preference. Note: full upfront sort was tried (commit 2f61d3a) and reverted at
  -129 ELO. **Pick-best-on-step (single O(n) max-find before each move)** has not been tried
  and is the right next experiment — only pays for moves searched before cutoff.
- `engine/search.go:1450-1478`: in-check evasions also raw-order. Could try TT-move-first +
  captures-first split.
- Fix: capture ttMove from TT.Get; pick-best-on-step ordering for both branches.
- Impact: speculative ~10-25 ELO. Smoke-gate carefully; revert on regression.

### Pri 5: `info.SelDepth` not reset between iterations (cosmetic)
- The pending diff in `engine/search.go` adds SelDepth tracking but never resets it at the
  top of each iterative-deepening depth, so iteration N reports SelDepth from the deepest
  prior iteration.
- Fix: assign `info.SelDepth = currentDepth` at top of loop body in `searchIterativeDeepeningUnsafe`.
- Impact: 0 ELO; UCI cosmetic only. But ship it correct since the diff is open.

### Audit findings flagged but FALSE-POSITIVE
- **King centrality "negative for corners"** (`eval.go:1476-1484`): agent claimed HIGH/+50 ELO.
  Verified: `whiteCentrality = 6 - centerDistance` ranges [-1, +5] (slightly asymmetric, off
  by one). Term enters as `(whiteCentrality - blackCentrality) * 3` so the asymmetry cancels
  when both kings have similar status. Worst-case ~3cp calibration smell, not a real bug.
- **Cache.go atomic race / replacement strategy** (`cache.go:107-113, 157-164`): agent claimed
  partial-update visibility. This is the textbook XOR-verifier (lockless) pattern; mismatch
  detection rejects partial updates safely.

### Audit findings UNVERIFIED (defense-in-depth, low urgency)
- White EP missing rank validation (`movegen.go:198-214`) — perft passes; defense-in-depth fix
  if EP could ever be set to invalid rank by some other bug. Cheap to add.
- Movetime + 50ms overhead floor (`time.go:265-266`) — claim is plausible only at very low
  movetime (<100ms), uncommon in our smoke (1000ms). Verify if real.
- Soft-stop volatility burning time (`time.go:370-381`) — speculative; would need targeted A/B.

## Older deferred items

### External Polyglot books silently broken
- `engine/polyglot.go:33-55` — `polyglotRandom[781]` is generated from xorshift seed=1 instead
  of the spec-defined constants. `PolyglotHash` of starting position yields
  `0x4b9aa9e5e768fe35`; spec says `0x463b96181691fc9c`.
- A real `.bin` polyglot book will never match positions. Embedded book still works.

### Test hygiene (cosmetic)
- `engine/play_quality_test.go` (`TestEnginePlayingStrength`) compares `Move.ToString()` UCI
  output (`b1c3`, `e1g1`) to expected SAN (`Nc3`, `O-O`). Engine plays the right move, test
  marks it neutral.
- `engine/crash_stress_test.go:250-278` (`TestConcurrencyStress`) shares one mutable
  `*Position` across goroutines.

### Qsearch quiet-check detection (deferred — low priority)
- Currently qsearch is captures-only (quiet checks intentionally dropped after the
  `moveGivesCheck` direct-attack-only bug). If quiet-check coverage is desired later, use a
  proper discovered-check detector or make/unmake + `isInCheck`.

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

### Evaluation

Currently wired into `evaluateUnsafe`:
- [x] Material + PeSTO tapered MG/EG piece-square tables (all pieces, not just king)
- [x] Passed pawns bonus
- [x] Doubled pawns penalty
- [x] Isolated pawns penalty
- [x] Pawn chains bonus (re-wired commit a609f94 — was orphaned by PeSTO swap c6675ff)
- [x] Bishop pair bonus
- [x] Mobility (knights/bishops/rooks/queens, enemy-pawn-attacked squares excluded)
- [x] Rook on open / semi-open file
- [x] Outposts (knight/bishop on pawn-supported, unattackable enemy-territory squares; re-wired commit 94c4316)
- [x] King safety (pawn shield + attack-table king-zone scoring; middlegame only, phase ≥ 12)
- [x] King activity (centralization + king-near-pawns; deep endgame only, phase < 6; re-wired commit c0f6bbb)
- [x] Tempo bonus (10cp side-to-move, applied in `EvaluateForPlayer`)

Orphaned (function defined but never called from `evaluateUnsafe`; left as dead code by the PeSTO swap):
- [ ] `evaluatePawnStorms` (eval.go:561) — TRIED AND REVERTED at full magnitude in commit 7e8702b (reverted by 8c773eb). Tactical held at 27/30 but 5-game smoke went 0W-5L vs SF1500 (rolling 1524 → 1430). Hypothesis: full-magnitude storm bonus (28cp peak in Italian-style positions) over-rewards pawn pushes that weaken own king's pawn shield, especially when our king is also uncastled. If retried, scale magnitude down ~50% AND add a "only fire when own king is castled OR opponent king is castled" guard so we don't push our own structure apart against a king still in the center.
- [ ] `evaluatePieceCoordination` (eval.go:1517) — large/inflated magnitudes (queen-protection ×8 etc.); inline comments admit values were already tuned-down once. High risk to re-enable without recalibration
- [ ] `evaluateWeakSquares` (eval.go:450) — short-circuits `return 0`; real logic commented out for performance. Effectively a re-implementation
- [ ] `evaluateBasicEndgames` (eval.go:1256) and `evaluateKingPawnOpposition` (eval.go:1304) — `+800/+600` magnitudes mis-calibrated against PeSTO; deferred per project_bugs memory

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

## Realistic non-NN path to 2200 ELO

You're at ~1450-1650 rolling. Target +550-750 ELO. Reference points: Crafty ~2600, Fruit
~2700, pre-NNUE Stockfish ~3000 — all classical. The route is staged, each rung gated by
smoke. Sized in expected ELO buckets per stage; reality will vary.

**Stage 1 — Audit-batch fixes (target 1700, +50-100 ELO)**
- Pri 1-5 above: mate distance ply, SEE/delta promotion, singular reduction, qsearch TT+ordering.
- Tactical 27/30 → 29/30 expected from #2 + #4 alone.

**Stage 2 — Texel tune existing magnitudes (target 1800, +50-100 ELO)**
- `engine/texel.go` exists. The wired components (mobility baselines, king-safety quadratic
  curve, pawn-structure penalties, PST blends, tempo) are hand-tuned or sub-agent-guessed.
- Run a real Texel session against ~50K-100K labeled positions. Expect biggest movers in
  king-safety weights and mobility baselines.
- After tuning, re-evaluate orphans (`evaluatePawnStorms`, `evaluatePieceCoordination`) at
  freshly-tuned magnitudes — they may be salvageable post-recalibration.

**Stage 3 — Better history (target 1850, +30-50 ELO)**
- Replace `[from][to]` history with `[piece][to]` (color/piece-aware).
- Add **continuation history**: indexed by `[prev-piece][prev-to][curr-piece][curr-to]`.
  Known +30-50 ELO from Stockfish/Ethereal experiments.
- LMR formula change: `log(depth) * log(moveNum) / 2` (Stockfish-style) instead of current
  piecewise.

**Stage 4 — Eval features that move the needle (target 1950-2000, +120-170 ELO)**
- Hanging/loose pieces: TRIED 2026-05-03 at piece_value/4 magnitude — single-shot smoke
  2W-3L vs SF1500, rolling 1582 → 1553 (-29 ELO). Tactical 27/30 held. Reverted same
  session. Hypothesis for retry: piece_value/4 (P=25, N=80, B=82, R=131, Q=250) double-
  counts material — the static eval already shows +V if we have the piece, then deducts
  V/4 because it might be lost. Stockfish uses a flat ~60cp per hanging piece, not value-
  scaled. Retry with flat 25-40cp penalty AND/OR gate magnitude on whose-turn-it-is.
  Better: defer to post-Texel when magnitudes can be tuned automatically.
- King safety upgrade: open/half-open files in front of king + pawn shelter ratio (the
  attack-table now exists; pair with pawn-shield strength).
- Threats: piece attacks higher-value piece, discovered-attack potential, pinned-piece bonus.
- Endgame heuristics that don't require tablebase: KPK rule-of-the-square, KBNK corner
  driving, opposition.
- Space evaluation in middlegame.

**Stage 5 — Search depth (target 2050-2100, +50-100 ELO)**
- Multi-cut after null move (search 2-3 moves at reduced depth before accepting null fail-high).
- Probcut margin / depth re-tune.
- Razoring depth/margin re-tune.
- Internal iterative reductions for non-PV nodes without TT move (cheaper than IID).

**Stage 6 — NPS (target 2150, +50 ELO)**
- 350K NPS is low. Profile via `cpu.prof`. Movegen at 47KB has redundant paths; hot path
  probably re-traverses bitboards.
- Realistic Go-bitboard target is 800K-1M NPS. 2× NPS ≈ +50 ELO via deeper search.

**Stage 7 — TT, time, polish (target 2200, +50 ELO)**
- 2-bucket TT (depth-preferred + always-replace).
- Wire UCI Hash option (currently ignored).
- Smarter time management: easy-move detection, reduce on book/forced moves, extend on score drop.
- Aspiration window: smaller initial (16cp), exponential growth.

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
