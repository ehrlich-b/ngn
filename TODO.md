# NGN Chess Engine - Development Roadmap

## Goal: 2500+ ELO (Grandmaster Level), with 2200 as the next milestone

**Current Status (2026-05-24):** Built the measurement instrument that unblocks
everything from here — `cmd/sprt`, an NGN-vs-NGN self-play SPRT. The 5-game
vs-Stockfish smoke provably can't resolve sub-100-ELO changes (identical binary
swung 80%→20%), which is why every eval/ordering tweak kept coming back "neutral"
in the noise. sprt fixes that: **fixed-depth games by default** (every move
reaches real depth — directly answers the "hyperbullet won't get enough depth"
worry — and removes time noise), authoritative game-end detection via the engine
package (checkmate/stalemate/threefold/50-move; the elo-assess loop mis-scored
stalemate as a loss), reversed-color opening pairs for variance reduction,
`-genbook` to scale openings, `-basedepth` for a positive control, and a GSPRT
LLR stop. This is the prerequisite for Stages 2-7 (eval features, Texel,
search-efficiency) — none of which the smoke can measure. Validated end-to-end:
null (identical binaries) = exactly 50.0% (20W-8D-20L), LLR -0.10, no false
positive; positive control (new depth 7 vs base depth 3) correctly tripped H1
ACCEPTED at +280 elo. First real use is the next eval/ordering change.

Also closed the bugs-first gate: dissected the worst residual blunder (the
29301cp mate-walk) with a deterministic depth-1→16 scan — it's a **false-perpetual
horizon illusion** (NGN scores it 0.00/draw at depths 4-8, then sees white escapes
the perpetual at depth 9-10 and switches to SF's move), NOT a bug. The smoking gun
for *why* movetime stalls at the blundering depth: depth 9 = 289K nodes but depth
10 = **35.5M nodes** (120× explosion). Confirms search efficiency in sharp
positions is the remaining lever — exactly the tuning-shaped work sprt now measures.

**Current Status (2026-05-23):** ~1711 rolling-smoke estimate, 95% CI [1555, 1867]
(commit-decayed, NGN 1000ms vs SF1500 200ms), up +243 from 1468 at the start of today's
blunder work and still climbing as pre-fix runs decay. Post-fix smokes went 19W-0D-1L
(95%, up from ~50%); trajectory 1468 → 1585 → 1660 → 1711. Tactical 30/30. Four search
fixes today (partial-commit, mate-found stop, losing-mate UCI formula, + the diagnostic
tool); the dominant gain was the partial-commit fix below.

MAJOR FIX (commit 362fd24): the #1 strength bottleneck was NOT eval — it was a SEARCH
ROBUSTNESS bug. The iterative-deepening loop committed a *half-searched* best move when
the clock interrupted an iteration mid-move-loop (the classifier didn't check
`info.Stopped`), so under time pressure NGN played catastrophic blunders (hung
queens/rooks, walked into mates). Fixed with `if info.Stopped { break }` before the
classifier (search.go ~624). Verified three ways: oracle blunder rate 9.7%→3.2%, tactical
29→30/30, and two post-fix smokes went 9W-0D-1L (90%) lifting rolling 1468→1585.

How it was found: built `cmd/blunder-finder` (Stockfish-oracle centipawn-loss detector;
see -log/-in, -reset-tt, -depth modes). Dissection proved it was search-not-eval (static
evals of blunder positions were fine), and a fixed-depth-vs-movetime + reset-tt-vs-carry
A/B isolated the cause to the time-interrupt path. NPS also VERIFIED ~750K-1M (not the
stale "350K"), so Stage 6 is already met. The 5-game smoke is still a coarse instrument
(swung 80%→20% on an identical binary) — fine for catching big effects like this fix,
useless for <100-ELO tuning.

Mate hygiene DONE (commits e8d5eda): added a mate-found stop to the ID loop (no more
depth-100 spin; tactical suite 4.2s → 0.2s) and fixed the losing-mate UCI formula
("mate -501" → correct distances). Both gated by smoke-tactical 30/30, neutral on play
(same move, just stops deepening) — expected, since the mate spin was post-fix mostly
cosmetic.

Remaining blunders (~3% at movetime, down from ~9%): all HORIZON misses — NGN blunders at
depth 7-9 but finds the right move by depth 10-13 (verified by depth-scaling). NOT discrete
bugs. The lever is reaching more depth in sharp positions, where the search explodes
(~35M nodes to reach depth 10 in an 8-piece position). So further blunder reduction =
search EFFICIENCY work (move ordering, aspiration windows, extension/qsearch discipline)
— diminishing returns and tuning-shaped, so it needs the SPRT self-play instrument
(deferred) to measure. Eval/Texel stays premature. Highest-value next: either build the
SPRT harness (TC ≈ current smoke depth, NOT hyperbullet) to unlock fine tuning, or keep
using cmd/blunder-finder for any remaining gross bugs.

**Prior status (2026-05-03):** 1580 ELO rolling-smoke estimate, 95% CI [1446, 1714]
(width 268, commit-decayed sample of 26.0 games across 11 runs at NGN 1000ms vs SF1500
200ms). Tactical 29/30. Trajectory this session: 1477 → 1533 → 1582 (2 post-audit
smokes) → 1553 (hanging-pieces tried/reverted) → 1559 ([piece][to] history) → 1564
(continuation history) → 1599 (LMR log-formula) → 1580 (mate-distance ply fix; tactical
28→29/30, rolling drift within noise). +103 ELO net since session start, all changes
gated by smoke-tactical and committed individually. Stage 1 ladder target 1700 within
reach: CI lower bound 1446 vs 1700 target.

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

### Pri 1: Mate distance pruning uses stale `info.Depth` (FIXED)
- `engine/search.go:1081` and `:1397` (alphaBetaPV checkmate paths) used
  `info.Depth - depth`, which is stale during the tree walk.
- Fix: both sites now use `ply` directly. Qsearch path (line 1516) and the
  top-of-alphaBetaPV mate-distance pruning (line 946) were already on `ply`.
- Commit b50aa23. Tactical 28 → 29/30; rolling 1599 → 1580 (drift within noise).

### Pri 2: SEE and delta pruning ignore promotion gain (FIXED in 5b0032b)
- Both initial SEE (`engine/search.go:139-143`) and qsearch delta pruning (`:1579-1583`)
  now add `promoPiece.seeWeight() - 100` (pawn weight) when `move.PromoType() != NoType`.

### Pri 3: Singular extension reduction is too aggressive (FIXED)
- Replaced `depth/2` verification depth with `max(1, depth-4)` per Stockfish/Ethereal.
- Commit 8269da1. Tactical 29/30 held; rolling 1580 → 1551 (drift within noise).

### Pri 4: Qsearch discards TT move + searches captures unordered (FIXED in 5b0032b)
- Both in-check evasion branch (`:1477-1484`) and captures branch (`:1564-1571`) now
  swap ttMove to index 0 if present. Avoids the -129 ELO upfront-MVV/LVA sort regression
  while still letting TT-suggested moves cut early.

### Pri 5: `info.SelDepth` not reset between iterations (FIXED in 5b0032b)
- `info.SelDepth = currentDepth` set at top of each iteration loop in
  `searchIterativeDeepeningUnsafe` (`engine/search.go:513`).

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

**Stage 1 — Audit-batch fixes (target 1700, +50-100 ELO) — COMPLETE**
- All Pri 1-5 done. Tactical 28 → 29/30 (mate-distance ply fix). Rolling drifted
  1599 → 1551 across two changes — within Wilson noise; CI lower bound 1414 vs
  the 1700 ladder target.

**Stage 2 — Texel tune existing magnitudes (target 1800, +50-100 ELO)**
- `engine/texel.go` exists. The wired components (mobility baselines, king-safety quadratic
  curve, pawn-structure penalties, PST blends, tempo) are hand-tuned or sub-agent-guessed.
- Run a real Texel session against ~50K-100K labeled positions. Expect biggest movers in
  king-safety weights and mobility baselines.
- After tuning, re-evaluate orphans (`evaluatePawnStorms`, `evaluatePieceCoordination`) at
  freshly-tuned magnitudes — they may be salvageable post-recalibration.

**Stage 3 — Better history (target 1850, +30-50 ELO)**
- [x] Replace `[from][to]` history with `[piece][to]` (commit 8b04b85, 2026-05-03).
  Tactical 27/30 → 28/30; rolling 1553 → 1559 (+6 ELO).
- [x] Add **continuation history**: indexed by `[prev-piece][prev-to][curr-piece][curr-to]`
  (commit 7a309c3, 2026-05-03). Tactical 28/30 held; rolling 1559 → 1564 (+5 ELO);
  CI lower bound +7. Combined Stage 3 effect: +11 ELO rolling, +16 lower bound.
- [x] LMR formula change: `log(depth) * log(moveNum) / 2` precomputed table replaces
  piecewise base + legalTried thresholds + depth>6 bump (commit 2351daf, 2026-05-03).
  Tactical 28/30 held; rolling 1564 → 1599 (+35 ELO); CI lower bound +26.

**Stage 4 — Eval features that move the needle (target 1950-2000, +120-170 ELO)**
- Hanging/loose pieces: TRIED 2026-05-03 twice. V1 at piece_value/4 regressed -29 ELO
  (rolling 1582 → 1553) — V/4 double-counts material. V2 at flat 30cp was neutral
  (3W-0D-2L, rolling 1599 → 1598, tactical 27/30 held). Both reverted. Real fix:
  defer to Stage 2 Texel so magnitudes co-tune with mobility/king-safety weights.
  Future retry could try flat 50-60cp (Stockfish's range) but only after Texel.
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
  TRIED 2026-05-03 with `if ttMove == EmptyMove && depth >= 4 && !isPV { depth-- }`
  right after the IID block. Tactical 29/30 held but 5-game smoke went 0W-0D-5L
  (all 5 losses to checkmate); rolling 1551 → 1460. Reverted (no commit).
  Hypothesis: depth >= 4 threshold is too aggressive in our codebase — most non-PV
  search nodes early in iterative deepening have empty TT, so IIR cascades through
  the entire shallow tree and effectively reduces full-game search by 1+ plies.
  Stockfish has many other compensating mechanisms (multi-cut, IIR depth >= 6,
  tt-aware reduction caps). Retry candidates: (a) raise threshold to depth >= 6 or
  >= 8, (b) only fire if depth - ttDepth >= 4 (skip IIR when TT has stale entry),
  (c) defer until after Texel + multi-cut so the search tree is more robust.

**Stage 6 — NPS (LARGELY DONE — verified 2026-05-23)**
- The "350K NPS is low" premise is STALE. Measured in-process: ~750K-1M NPS in real
  middlegame/tactical positions, 1.47M on startpos, depth 20 in 5s. The 350K figure came
  from the Feb 5 profiling files before optimizations.
- Already at/above the "realistic Go-bitboard target 800K-1M" this stage set as the goal.
  Search speed is NOT the bottleneck; do not spend time here. Re-profile only if a future
  position-mix shows a hot path, but expect single-digit ELO, not +50.

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
