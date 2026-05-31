# 10 — Performance / NPS (the speed lane)

*Profiled 2026-05-31 @ `87714a1` on E-cores (`taskpolicy -b`). Supersedes the stale
`docs/eliminate_alloc.md` (Feb 2026; its line numbers and 336-allocs/op figure are out of date).*

---

## Why this matters

Measured head-to-head on Kiwipete, equal time, both engines on E-cores:

| | NGN (HEAD) | Blunder 7.4.0 (CCRL 2532) |
|---|---|---|
| NPS | **~267K** | **~640–750K** (~2.5x) |
| Depth reached | 11 @ 3.6s | 13 @ 2.0s |

Blunder is ~2.5x faster per node and ~2–3 plies deeper at equal wall-clock. At ~40–60 ELO/ply
that depth deficit ≈ 100–140 ELO — comparable to the entire measured CCRL gap (2532 − 2380).
**NPS is the single biggest lever on NGN's strength, larger than any individual search bug.**
Both engines roughly agree on the eval of the position (NGN −136 cp @ d11, Blunder −86 cp @ d13),
so this is a *speed/depth* deficit, not an eval-sign disaster.

---

## The profile (verified, not estimated)

CPU profile of `BenchmarkSearchDeep`, 5.31s of samples. **Cumulative** share of total CPU by
subsystem (where the time actually goes):

| Subsystem | % CPU | Note |
|---|---|---|
| `EvaluateForPlayer` (PeSTO + pawn structure + correction) | ~15% | spread across many small fns |
| `isInCheck` | ~15% | called 2–3× per move (node-in-check, post-make legality, gives-check) |
| `orderMovesSinglePassIntoBuffer` | ~14% | scores ALL moves (SEE per capture) + **O(n²) selection sort** |
| `GenerateMovesIntoBuffer` | ~10% | full pseudo-legal gen every node |
| **`runtime.memclrNoHeapPointers`** | **~8%** | **zeroing per-node `[256]Move`/`[256]int` arrays** |
| **`runtime.madvise`** | **~9%** | **GC returning memory — driven by hot-path heap allocation** |
| **`runtime.(*_panic).nextDefer`** | **~3.5%** | **the per-node `defer` for the RepStack pop** |
| make/unmake, hashing, recursion | remainder | |

Allocation profile (`-alloc_objects`): **`alphaBetaPV` = 62% of allocations, `quiescenceWithDepth`
= 33%.** Current rate ~77–207 allocs/op (`go test -benchmem`; the Feb doc's 336 has improved but
the hot path still allocates heavily).

### Root cause: the zero-alloc infrastructure was half-built and abandoned

`SearchInfo` pre-allocates four reusable buffers (`MoveBuffer`, `OrderedBuffer`, `CaptureBuffer`,
`SEEGains`, search.go:478). **Only `CaptureBuffer` is actually used** (qsearch :1816). `MoveBuffer`,
`OrderedBuffer`, and `SEEGains` are **dead fields.** Meanwhile the hot path stack-allocates fresh
arrays *every node*:

| Site | Array | Bytes/node | Lifetime |
|---|---|---|---|
| search.go:1251 | `var moveBuffer [256]Move` (raw gen) | 1 KB | transient (gen→order→discard) |
| search.go:1270 | `var ordered [256]Move` | 1 KB | **persists across the move loop's recursion** |
| search.go:1278/1280 | `searchedQuiets`/`searchedCaptures [64]Move` | 0.5 KB | persists across loop |
| moveorder.go:321 | `var scores [256]int` | 1 KB | transient (inside ordering) |
| qsearch :1730/:1783/:1817 | `[256]Move`×2 + `[64]Move` | ~1.6 KB | mixed |

≈ 3.5 KB allocated + zeroed per main-search node. The compiler can't prove these don't escape (the
slice is handed to non-inlined `GenerateMovesIntoBuffer`), so they land on the heap → the ~8% memclr
(zeroing) + ~9% madvise (GC) + the 62%/33% alloc split above.

---

## Ranked attack plan

Each target tagged **[NEUTRAL]** (must be behaviour-identical — verify by depth-10 Kiwipete
**node-count + bestmove identity** vs clean HEAD, then measure NPS) or **[BEHAVIOUR]** (changes the
tree — needs a fixed-time SPRT, not just a node check).

### T1 — Reuse hot-path buffers via a per-FRAME pool `[NEUTRAL]` — SHIPPED 2026-05-31, KEPT
**Result: behaviour-neutral (proven) + clear improvement → KEPT.** Implemented as a recursion-frame-counter
pool (`searchFrame{moveBuffer,ordered,quiets,captures}`, balanced by the existing RepStack defer, stack
fallback on pool exhaustion). It cut allocations ~75% (`BenchmarkSearch` 77→18 allocs/op) and produced
**identical node counts + bestmoves at fixed depth on three positions** (Kiwipete d12 = 1,339,599 / d5e6;
middlegame d12 = 859,325 / d2c2; endgame d16 = 848,103 / b4f4) — proving zero strength impact. Real-search
speed measured ~0–2% (within E-core run-to-run noise), below the ~15–20% this estimate promised — but **a
clear improvement (75% fewer allocs, never slower) with ZERO regression is KEPT.** The keep bar is
"improvement + no regression," NOT "a speedup big enough to resolve." The +655KB/search is trivial against the
128MB TT, so there is no concrete cost. (Reject applies only to non-correctness changes WITH a concrete cost —
e.g. C5's 2× nodes; T1 has none.)

> **Measurement lesson (the real takeaway): the benchmark profile overstated allocation/GC.**
> The ~17% in `memclr` + `madvise` came from `BenchmarkSearchDeep` running *thousands of short searches in a
> tight loop* — that churns the allocator and GC. A single game search allocates its buffers once and GC
> barely runs, so removing 75% of allocations moved real NPS <2%. **Allocation is NOT this engine's
> single-search bottleneck.** Trust the profile's *flat compute* hotspots (ordering, isInCheck, eval,
> movegen — workload-independent), distrust its *alloc/GC/madvise cum* share for a single-search workload.
> Verify NPS on a real `go depth N` search, never on a benchmark-loop profile alone.

*(T2 and T4 target the trustworthy per-node compute and are the next levers — larger than T1.)*

#### (original T1 plan, for reference)
Kill the per-node `[256]Move`/`[256]int` allocation+zeroing. **The trap (this is a joint — see doc 09
J3):** the persistent buffers (`ordered`, `searchedQuiets`, `searchedCaptures`) are live across the
move loop's recursive calls, so they **cannot** be indexed by `ply` — IID (:1075) and singular
verification (:1459) re-enter `alphaBetaPV` at the *same ply*, and singular is called *mid-loop* while
`ordered` is being iterated. A naive per-ply buffer would clobber the parent's move list and
reintroduce a re-entrancy corruption. **Fix shape:** index a buffer pool in `SearchInfo` by a
*recursion-frame counter* that increments on every entry and decrements on exit (like `RepStackLen`),
sized with headroom for the re-entrant frames (≈ `2·MaximumDepth + EXTENSION_BUDGET`). The transient
buffers (raw gen `moveBuffer`, `scores`) are safe to share (no recursion during their lifetime) — point
them at single `SearchInfo` scratch fields. Verify: node/bestmove identity at depth 10–12 on Kiwipete +
a quiet middlegame, then NPS delta.

### T2a — Incremental (deferred) selection sort `[NEUTRAL]` — SHIPPED 2026-05-31, KEPT
**Result: behaviour-neutral (proven) + ~4–7% faster → KEPT.** The eager path scored every move then ran
a full O(n²) selection sort *before trying a single move* — the whole sort tail past a beta cutoff was
wasted. Split `orderMovesSinglePassIntoBuffer` into a score-only pass (`scoreMovesIntoBuffer`, moveorder.go)
plus a single-step `selectNextMove`, and drive the selection from the `alphaBetaPV` move loop (search.go
~1311/1325): pull the next-best move into `ordered[oi]` per iteration, so a cutoff at move *k* never sorts
moves *k+1..n*. A full selection sort and incrementally-deferred selection produce the **identical order**
(strict `>` keeps ties at the earlier index; positions `[0..i)` are finalized and untouched), so it is
**node-identical** — verified: Kiwipete d12 = 1,339,599 / d5e6, middlegame d12 = 1,165,258 / a2a3, endgame
d16 = 848,103 / b4f4, all matching clean HEAD exactly (nodes + seldepth + score + bestmove). `go test
-short ./engine/` green. Measured ~4–7% faster (288/305/286K → 317/305/312K NPS, same node count), never
slower → KEPT on "improvement + no regression." Scores stored in a frame-resident `searchFrame.scores`
(same re-entrancy constraint as `ordered` — IID/singular re-enter at the same ply). **Saves SORT work, not
SEE work** — SEE is still computed for every capture up front (needed to know the order). That is T2b.

### N1 — Incremental material+PST+phase eval `[NEUTRAL]` — SHIPPED 2026-05-31, KEPT
**Result: behaviour-neutral (proven) + ~3.6% faster → KEPT.** `evaluatePeSTO` and `gamePhase` each rescanned
the whole board every eval call (up to 32 piece lookups + 8 popcounts), and eval runs at internal nodes too
(RFP/futility/NMP/improving), so the scan spanned the whole tree. Replaced the scan with a running accumulator:
three ints on `Bitboard` (`accMG`, `accEG`, `accPhase`) maintained by the only three board-mutation primitives —
`Clear` (subtract the removed piece), `UpdateSquare`/`Move` (add the placed piece). Because *every* make/unmake
path funnels through those three (verified: no raw bitboard-field writes exist outside `bitboard.go`), the delta
is automatically exact for every move type — normal, capture, castle (the recursive rook `Move` self-maintains),
en passant, promotion, capture-promotion — with **zero per-move-type code**. `evaluatePeSTO` now just reads the
accumulator (clamping phase to `totalPhase` exactly as `gamePhase` did, so output is bit-identical); the dead
`gamePhase` was removed. The sums are plain integer adds → order-independent → **bit-exact with a full scan** →
**node-identical**: Kiwipete d12 = 1,339,599 / sd32 / d5e6, middlegame d12 = 1,165,258 / a2a3, endgame d16 =
848,103 / b4f4 — all matching clean HEAD exactly (nodes + seldepth + score + bestmove). `TestEvalAccumulatorUnderMakeUnmake`
walks a depth-3 tree over 7 FENs (incl. castling/EP/promo/capture-promo) asserting the accumulator equals a fresh
`recomputeAccumulator()` scan at every node and after every unmake. Measured ~+3.6% NPS over 8 interleaved Kiwipete
rounds (334.7K→346.8K) + endgame (645K→669K), faster in the clear majority, never a real regression → KEPT on
"improvement + no regression." **Does NOT contradict the "don't trim eval" rule below** — eval is exact, not
trimmed; only the *scan* was eliminated. The accumulator depends on the PST tables, so the Texel tuner (which
mutates them via `RebuildPST` between passes) calls `recomputeAccumulator()` per sample before evaluating — the
same O(32) scan it already paid, no tuner regression. *Caveat:* the accumulator adds ~2-4 table-indexed adds per
make/unmake (incl. the movegen-legality path), which eats into the eval saving — that tradeoff is why the net is
~3.6%, not the full eval slice. A future packed `int` score (mg/eg in one word, à la Stockfish `make_score`) would
halve that overhead; not done now.

### T2b — Lazy SEE / staged generation `[BEHAVIOUR]` — est. large (cuts the SEE inside the ~14%)
Today scoring still calls `staticExchangeEvaluation` on **every** capture before the loop runs. Competitive
engines stage it: yield the TT move, then captures ordered by MVV-LVA with **SEE computed only when a
capture is actually reached**, then killers/counters/quiets — stopping as soon as a beta cutoff hits.
Changes the capture tie-break order ⇒ node counts shift ⇒ **NOT node-identical** ⇒ needs a fixed-time SPRT
(keep iff not a regression). Bigger SEE-cost win than T2a, but behaviour-affecting — do it after T2a.

### T3 — Drop the per-node RepStack `defer` `[NEUTRAL]` — est. +3%
`defer func(){ info.RepStackLen-- }()` (search.go:1026) runs on every node; Go's defer machinery shows
as ~3.5% (`nextDefer`). Replace with an explicit `info.RepStackLen--` at each return path. **Caution:**
this is the very invariant doc 09 INV-A3/J3 guards — every `return` in `alphaBetaPV` must decrement
exactly once. Pairs naturally with the J3 defensive fix (guard the pop). Verify: node/bestmove identity
+ the RepStack-balance reasoning re-checked at each return.

### T4 — Cut redundant check detection `[NEUTRAL/BEHAVIOUR]` — est. part of ~15%
`isInCheck` runs up to 3× per move (node-in-check :1100, post-make legality :1387, gives-check :1406).
The post-make legality check and the gives-check probe both recompute king attacks. Investigate caching
the king-attack computation, or deriving `givesCheck` from the move + a cheap incremental test. Some of
this is genuinely behaviour-neutral (caching); a cheaper gives-check approximation would be behaviour-
affecting. Lower priority — measure after T1/T2.

### Not now
- **Eval cost (~15%)** is in-line with peers and is where strength lives — do NOT trim eval terms for
  speed before T1–T3; the overhead targets are free, eval is not.
- **Lazy SMP / multi-threading** is the other large lever but is a separate project (the `Threads`
  option is a stub; INV-A7 — every global heuristic table becomes a race) and is out of scope here.

---

## Sequencing & gate

Do T1 first (biggest **safe** win, behaviour-neutral, no SPRT needed — just node-identity + an NPS
measurement). Then T2 (the big algorithmic win, gated by a fixed-time SPRT). T3/T4 as cleanup. Re-run
the Kiwipete vs-Blunder depth comparison after T1+T2 to quantify how much of the 2.5x closed.

**Every [NEUTRAL] target must produce identical node counts and bestmoves at fixed depth** — that is the
proof it changed only speed, not strength. If node counts move, it's secretly a [BEHAVIOUR] change and
needs an SPRT. This is the speed-lane analogue of the keep-correctness-fixes rule (doc 09).
