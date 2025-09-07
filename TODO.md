# NGN Chess Engine — Overhauled Roadmap

This roadmap focuses on achieving approximately 2x speed and 2x effective depth through a mix of raw throughput and stronger pruning/ordering. It replaces prior tactical fixes with a performance-first plan and clear validation.

## Objectives
- 2x wall-clock throughput on single thread (NPS) on a mixed position set.
- +1 to +2 ply deeper on typical midgame searches at fixed time.
- No regressions on legality, stability, or correctness tests.

## Key Metrics (track per commit)
- NPS on a 100–500 position mixed suite (midgame, tactical, quiet).
- Nodes searched to depth X on fixed positions; depth reached at fixed time.
- TT hit rate, cut node ratio, qsearch node share.
- Correctness: perft spot-checks, move legality, tactical regression tests.

## Recently Fixed
- Sliding move buffer wrap bug causing illegal rook moves (e1a2) fixed in `generateSlidingMovesIntoBuffer` and `generateSlidingCapturesIntoBuffer`.
- Root legality checks enforced in search to avoid returning illegal moves.

## Phase 1 — Profiling & Benchmarks (1–2 days)
- [ ] Add a deterministic benchmark harness (CLI or test) to report: NPS, nodes, depth per position.
- [ ] Add CPU profiles on representative searches; store flamegraphs for before/after comparisons.
- [ ] Lock a corpus: 100–500 FENs across openings/midgames/endgames; include a small tactics suite.

## Phase 2 — Core Throughput (3–5 days)
- [ ] Implement magic/lookup sliding attacks (rook/bishop/queen) and use in:
  - [ ] Move generation (all variants, including buffer-based).
  - [ ] `isInCheck`/attack queries and pin/line-of-sight checks.
- [ ] Minimize hot-path work:
  - [ ] Reduce `PieceAt` calls inside loops (prefer bitboards/attacks). 
  - [ ] Inline trivial helpers; trim branches in Make/UnMake paths.
- Expected: 30–60% speedup depending on current hotspots.

## Phase 3 — Pruning & Ordering Gains (4–7 days)
- [ ] Implement SEE (Static Exchange Evaluation):
  - [ ] Order captures by SEE; prefer profitable or neutral exchanges.
  - [ ] Quiescence: prune losing captures; tighten delta margins based on SEE.
  - [ ] Main search: light SEE pruning for late captures in non-PV nodes.
- [ ] LMR: switch to dynamic reduction formula based on depth and move index; boost reductions for bad-history moves; reduce for good-history, checks, and captures.
- [ ] IID: if no TT move at PV nodes, do a reduced-depth probe to seed ordering.
- [ ] Null-move: adaptive R (2–3) based on depth and phase; zugzwang guards in likely endgames.
- Expected: 20–40% node reduction; better move ordering stability.

## Phase 4 — Transposition Table (2–4 days)
- [ ] Clustered TT buckets (e.g., 4 entries) with replacement by depth/age; keep current aging.
- [ ] Make table size configurable at runtime; add lightweight stats (hit, store, replace).
- Expected: 5–15% node reduction via higher hit-rate.

## Phase 5 — SMP (Parallel Search) (5–10 days)
- [ ] Root-split parallel search (2–N workers) with shared TT.
- [ ] Optional split points on deeper PV nodes after stability.
- [ ] Determinism toggle for testing; integrate with time manager.
- Expected: 1.5–1.8x wall-clock speedup on 2–4 cores.

## Phase 6 — Eval Throughput (3–6 days)
- [ ] Incremental evaluation: cache material/PST/selected terms; update in Make/UnMake.
- [ ] Gate expensive terms by depth or PV-only to reduce average eval cost.
- Expected: 10–25% time saved if eval is a hotspot.

## Guardrails
- Never regress legality: keep perft spot-checks and legality tests on every movegen change.
- Quiescence stability: cap qDepth; only admit a limited set of non-capturing checks; SEE-based pruning required.
- Maintain zero allocations in hot paths (search, movegen, ordering).

## Validation Plan
- [ ] Continuous benchmark: report NPS, depths, nodes, TT hit-rate vs. baseline.
- [ ] Tactical suite pass rate; ensure no pruning-induced tactical misses grow over baseline.
- [ ] Nodes-to-depth comparison for key test FENs before/after each phase.

## Stretch Goals
- [ ] Specialized endgame pruning (e.g., late endgame futility/razoring variants with safe guards).
- [ ] Extended history heuristics (butterfly history, continuation history).
- [ ] Tapered and or phase-aware evaluation refinements once speed is achieved.

## Implementation Pointers
- Magic attacks: add a new `attacks/` module or embed into `global.go`; expose `RookAttacks(bb, sq)`, `BishopAttacks(bb, sq)` using blockers.
- SEE: add a lightweight, non-allocating SEE; avoid recursion/alloc in hot paths.
- LMR/IID/null-move: update `search.go` with dynamic formulas; keep guardrails to ensure depth strictly decreases.
- TT buckets: extend `Cache` in `cache.go` to support clustering and depth/age-prefer replacement.

## Status Snapshot
- [x] Illegal sliding wrap fixed in buffer movegen.
- [x] Root legality checks to avoid illegal best moves.
- [ ] Magic attacks
- [ ] SEE + qsearch pruning
- [ ] Dynamic LMR + IID
- [ ] TT clustering
- [ ] SMP root split

This plan is designed to compound speed and depth gains. If single-thread plateaus short of 2x, SMP provides the remaining wall-clock improvement without compromising tactical strength.

## Benchmark Findings (2025-09-06)

Current targeted benchmark results and profiles for guidance. Values are approximate; keep snapshots per commit for tracking.

- BenchmarkEvaluation: ~370–400 ns/op, 0 allocs/op (Eval is cheap).
- BenchmarkMoveGeneration (starting pos): ~2.4 µs/op, 34 allocs/op (non-buffer API; search uses buffer APIs).
- BenchmarkLegalMoveGeneration: ~5.6 µs/op, 35 allocs/op (non-buffer API).
- BenchmarkCriticalPath (SearchFixed depth 4): ~1.36–1.50 ms/op, ~263 KB/op, ~350 allocs/op.

CPU hotspots (depth-4 critical path):
- isInCheck: ~16–18% CPU
- orderMovesIntoBufferWithDepthPreviousAndTTMove: ~14% CPU
- GenerateMovesIntoBuffer (total across pieces): ~16–19% CPU
- Evaluate/EvaluateForPlayer: ~12–17% CPU combined
- TT atomics (atomic.Store/Load in Cache.Set/Get): ~12–20% CPU cumulative
- QuiescenceWithDepth: consumes ~44–52% of time cumulatively

Memory hotspots (alloc_space / alloc_objects):
- One-time: TranspositionTable init (NewCache): 128 MB
- Per-search allocations dominated by:
  - quiescenceWithDepth: ~25–27% of bytes; ~100k+ alloc objects
  - alphaBeta: ~22–23% of bytes; ~40–50k alloc objects

Notes:
- Removing closure from eval PST loop and avoiding defer in WrapEvaluation yielded small wins but did not change the dominance of qsearch + ordering + inCheck.
- Biggest returns will come from cutting nodes (qsearch pruning), improving ordering passes, and speeding inCheck.

## Immediate Actions (High ROI next)

1) QSearch Node Reduction (primary)
- [ ] Implement SEE (Static Exchange Evaluation) with no allocations.
- [ ] Use SEE to order captures and to prune losing captures in qsearch.
- [ ] Tighten delta pruning margins using SEE-informed bounds.
- [ ] Restrict non-capturing checks admitted in qsearch to a very small, high-value set.

2) Move Ordering CPU Reduction
- [ ] Replace multi-scan staged ordering with a single-pass classifier:
  - [ ] Emit TT move early; then classify into captures (good/bad), promotions, castles, killers, counters, quiets.
  - [ ] For remaining quiets, one scoring pass (history/counter/killer) and selection without multiple rescans.

3) isInCheck Speedup (no magic requirement)
- [ ] Switch sliding attacks in isInCheck to ray masks + straight bit scans (no per-step loops).
- [ ] Keep pawn/knight/king bitboard checks as-is.

4) TT Atomic Overhead (single-thread mode)
- [ ] Add build tag or runtime option to use non-atomic TT operations when running single-threaded.
- [ ] Optionally restrict TT stores (e.g., only at Cut/Exact nodes) to reduce writes.

## Short-Term Follow-ups (after above lands)
- [ ] SEE in main search for late captures at non-PV nodes (light pruning).
- [ ] Dynamic LMR formula (depth, move index, history) and add IID at PV nodes when TT has no move.
- [ ] Revisit GenerateMovesIntoBuffer micro-opts: reduce PieceAt calls; prefer masks; tighten branches in pawn and sliding generators.

## Not Doing Now (based on experience + data)
- Full magic bitboards for sliding: previously attempted and deemed not worth cost; profiles show higher ROI in qsearch pruning, ordering, and inCheck optimization without full magic infra.

## Validation & KPIs
- Always run: BenchmarkCriticalPath, BenchmarkEvaluation, BenchmarkMoveGeneration, and a small mixed FEN suite for NPS/depth tracking.
- Track per-commit:
  - NPS and nodes for depth-3/4 searches over a 100–500 FEN corpus.
  - TT hit rate, qsearch node share, cutoff rate, and ordering effectiveness (fail-high rate early).
  - Allocation counts from qsearch and alphaBeta (goal: shrink both meaningfully with SEE + pruning).

## Execution Order (Pragmatic)
- [ ] Implement SEE + qsearch pruning and integrate with capture ordering.
- [ ] Refactor move ordering to single-pass classifier + quiets selection.
- [ ] Optimize isInCheck with ray scans for sliding attacks.
- [ ] Add non-atomic TT option for single-thread bench builds; measure impact.
- [ ] Iterate on LMR/IID and further pruning once above is stable.
