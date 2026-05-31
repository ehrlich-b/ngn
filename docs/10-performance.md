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

### T2 — Lazy / staged move ordering `[BEHAVIOUR]` — est. large (attacks ~14% + cuts SEE)
Today ordering scores **every** move (calling `staticExchangeEvaluation` on every capture) and runs an
**O(n²) selection sort** before the loop tries a single move — wasted whenever an early move cuts.
Competitive engines pick moves incrementally: yield the TT move, then good captures (SEE only when
reached), then killers/counters, then quiets — and stop as soon as a beta cutoff hits. Replaces the
eager sort+SEE with on-demand selection. **Behaviour-affecting** (tie-break order can shift → different
node counts), so it earns a real fixed-time SPRT. Likely the biggest single NPS win after T1, and it
directly reduces the SEE cost folded inside the 14%.

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
