# 2026-07-25 Technique gap audit — refilling the queue for the 2800 goal

**Why this exists.** The 2800 target needs roughly **+190 self-play Elo** (+96 CCRL at the measured ~50%
transfer). The existing T-queue's stated priors sum to ~+55 *if every item hit in full*, and three of the last
four returned zero. **The queue is not deep enough to support the goal**, so this audits NGN against the
standard classical-engine technique set to find what is genuinely missing, rather than re-tuning constants
that four straight nulls suggest are tapped.

Method: grep HEAD for each technique, then verify the finding in code. Same premise-verification discipline
that caught the stale `DoubleExtMargin` and LMP-dims claims today.

## PRESENT (confirmed, not gaps)

History tables are well covered — `historyTable`, `continuationHistory` (2-ply), `followupHistory`,
`captureHistory`, and **three** correction histories (pawn / non-pawn / minor). Also present: IID/IIR,
NMP + RFP + futility + LMP + LMR, SEE pruning on both captures and quiets, killers, counter-moves,
singular extensions with negative-extension-lite, probcut, aspiration (T5-modernized), check / recapture /
passed-pawn extensions, delta pruning, qsearch capture TT store, and the T1b+T1e time-management package.
Razoring is deliberately absent (removed by decision).

## MISSING — genuine gaps (0 hits in HEAD)

### T19 — `ttPv` (RECOMMENDED next mined candidate)

**Absent entirely** (`ttPv`/`TTPv` = 0 hits). SF/Ethereal/Stash carry a per-entry flag recording that a node
was once searched as a PV node, and then **reduce it LESS** in LMR.

**Why this fits NGN specifically — it follows the diagnosis instead of fighting it.** NGN's first-move-cutoff
rate is ~84% vs elite ~90%, and **T3a proved that ADDING reduction hurts** (-7.9: more reduction over-reduces
mis-ordered moves, which then fail low silently). Every other item in the T3 LMR family is therefore gated
behind T10 raising FMC. **`ttPv` moves the opposite way** — it reduces *less* at nodes known to have mattered
— so it is aligned with the measured weakness and is **not** blocked by the T3 gate. That makes it the most
diagnosis-consistent unexplored item found in this audit.

**Feasibility — the real blocker is the bit budget, and it is solvable.** `CachedEval` packs into exactly 64
bits with nothing spare (cache.go:76-88):

| Field | Bits | Range used |
|---|---|---|
| hashmove | 28 | full |
| eval | 16 | full |
| depth | 7 | full |
| nodeType | 3 | **only 3 values** (Exact/UpperBound/LowerBound as `1 << iota` flags) |
| age | 10 | compared against `OldAge = 5` |
| **total** | **64** | fully packed |

Two independent ways to free one bit, no entry-size growth (entry size is load-bearing — the 16-byte entry
quadruples slot count at the same MB, which the endgame audit showed materially cuts the middlegame tree):

1. **nodeType 3 bits → 2.** It stores only three values but wastes a bit by encoding them as bit flags rather
   than 0/1/2. Touches every pack/unpack site, so it needs a node-identity proof.
2. **age 10 bits → 9.** 1024 generations against `OldAge = 5` is far more headroom than needed.

Option 2 is the smaller blast radius and is the recommended route.

**Status: NOT nodecheck-gated yet.** Must pass the >=1% behavioral-delta gate before it gets box time.

### `cutoffCnt` — GATED, do not queue yet

Absent. SF tracks per-node cutoff counts and **increases** reduction when a node has produced many. That is an
LMR-reduction *add*, which is exactly the class **T3a falsified and T10b failed to unblock**. It stays behind
the T10/FMC gate. Recorded so a future agent does not mistake it for free Elo.

### NMP verification search — low priority

Absent. NGN guards zugzwang with `hasNonPawnMaterial` only, with no high-depth verification re-search. This is
mostly a correctness/blunder-avoidance item in deep endgames rather than a throughput win; low expected Elo at
10+0.1 bullet, where such positions are rare. Park it.

## The throughput lever (larger than any single item here)

At ~5h per SPRT, +190 self-play is ~135 SPRTs ≈ **~675 box hours ≈ 28 days** of continuous box time. **Halving
that is worth more than any candidate in this file.**

**M6 should be revisited.** 5+0.05 c8 was NOT enabled because its A/A came back penta **-8.3 [-20, +4]** over
1600 games — but **that CI includes zero**. It was rejected for looking lopsided, not for being demonstrated
biased. A properly powered A/A (4000-8000g) would settle it, and if 5+0.05 is clean the entire campaign runs
at ~2x, cutting the 28-day figure to ~14.

This is a mill (M-item) change with no Elo risk of its own — it only changes the ruler, and it is gated by its
own A/A exactly as M5 and M6 were. Per the revamp's "mill first" ordering it should outrank the T-queue.
Recommend it as the next box job after T8's gate, ahead of T12.

## Resulting queue shape

1. **T8** — live, SPSA round 1, then its confirmation SPRT (must gate before T12; it tunes NMP dims).
2. **M6-retest** — powered 5+0.05 A/A. Highest leverage: halves every subsequent run.
3. **T12** — NMP cut-node-only, pre-verified, one line, +9.7 prior
   (`experiments/2026-07-25-t12-nmp-cutnode.md`).
4. **T19** — `ttPv`, per above; needs a freed TT bit + nodecheck gate.
5. **T18** — probcut TT short-circuit + TT store (two sub-items; lane confirmed live, pcut 1622 @kiwipete d12).
6. T15 / T11a / T9b / T10a / T8b — as previously ranked.

---

## T19 `ttPv` — IMPLEMENTED and GATED 2026-07-25 (shelved at `output/t19.patch`, 190 lines)

Built, measured, reverted. Tree restored and re-verified to the exact 346662/149587/765656 baseline; nothing
speculative is committed.

**Implementation (the bit budget resolved as recommended — age 10 → 9 bits):**

- `AGE_MASK` 10 → 9 bits, new `TTPV_MASK` at bit 63. Entry stays 8+8 bytes, so the slot-count property that
  the endgame audit showed is load-bearing is untouched.
- `Pack`/`Unpack`/`Set`/`Get` all carry the flag (4 probe sites + 3 store sites updated).
- The mark is **sticky**: `nodeTTPv := isPV || (ttHit && ttPvEntry)`, re-stored true. That is the point — a
  node that has ever mattered keeps deserving less reduction after it drops out of the PV.
- LMR consumes it: `if nodeTTPv { reduction-- }`.

**Behavioral-delta gate — PASSES DECISIVELY:**

| Position | Base | T19 | Delta |
|---|---|---|---|
| kiwipete d12 | 346662 | 410704 | **+18.5%** |
| mid d12 | 149587 | 236690 | **+58.2%** |
| end d16 | 765656 | 516337 | **-32.6%** |

### BLOCKER: `TestIIDIsWired` fails under T19 — and it is NOT an IID break

`go test -short ./engine` fails with `IID never fired on this FEN` (IID-on == IID-off == 32561 nodes,
`IIDSearches == 0`) on `r2q1rk1/1b1nbppp/p2ppn2/1p6/3NPP2/1BN1B3/PPPQ2PP/2KR3R w - - 0 12` at depth 9.

**Verified rather than assumed** (the failure mode here would be to shrug and edit the test): IID fires
normally under T19 at depth 10 on that *same* FEN — `iid 1` for both HEAD and T19, on kiwipete and on the test
FEN. So the wiring is intact; T19 merely shifted the depth-9 search out of the IID-firing regime.

**This is the fragility the test's own comment already documents** — T4c's minor correction-history term moved
the previous FEN out of that regime once before, which is exactly why the `IIDSearches > 0` assertion was
added. The guard is working as designed: failing loudly beats passing spuriously with on == off.

**Do NOT weaken this test to let T19 through.** It is a durable lock, and editing a guard to accommodate a
speculative candidate is how locks get un-fixed. The correct sequence is:

**DONE 2026-07-25 — the test fix landed (test-only commit, engine untouched).** `TestIIDIsWired` now probes a
SET of four PV-heavy middlegames across depths {9,10} and guards on the first that actually fires, instead of
pinning one position. Strictly stronger than the single-FEN form: defeating the guard now requires EVERY probe
to drift out of regime at once. Verified both directions — on HEAD it still selects the original FEN at depth 9
(so what it guards there is unchanged, IID-on 35600 vs off 35439), and with `output/t19.patch` applied it falls
through the three drifted positions to `2rq1rk1/...` where IID fires and counts still differ (82152 vs 82069),
i.e. the guard stays genuinely exercised rather than passing vacuously. Full suite + `-race` green; nodecheck
unchanged at 346662/149587/765656, confirming the engine was not touched. **T19's blocker is cleared** — it now
needs only the qsearch-mark decision below before it is launch-ready.

Original plan, retained for the record:

1. ~~**Fix the test's robustness on its own merits first**~~ (own commit, justified independently of T19): choose
   a FEN/depth where IID fires reliably, or assert on a construction that cannot drift out of regime. It has
   now tripped on **two** separate tree-changing candidates, so it will keep blocking the queue — this is a
   real, recurring tax, not a one-off.
2. Then re-run T19's gate on a green baseline and launch.

T19 is therefore **NOT launch-ready**, unlike T12. Sequence it behind that test fix.

### RESOLVED 2026-07-25 — the qsearch mark question is IMMATERIAL (measured, not assumed)

Decided deliberately, then measured. Qsearch already probes the TT and was discarding the flag, so preserving
it costs nothing: capture `ttPvEntry` from the existing probe and pass `ttHit && ttPvEntry` to the depth-0
store instead of `false`.

**But the honest result is that it changes nothing measurable.** With the mark preserved, nodecheck is
**identical to the leaking variant on all three canonical positions** — 410704 / 236690 / 516337, exact. The
mechanism explains it: the replacement policy only lets a depth-0 store overwrite a same-position entry when
`depth >= oldDepth-3`, i.e. old depth <= 3, so a qsearch store almost never clobbers the deep entries that
carry a meaningful mark.

**Shipping the preserving variant anyway** — it matches SF, it is free, and it stays correct if the
replacement policy is ever changed (T11a's 4-way clustering would touch exactly that code). But it must **not**
be credited as an improvement: it is a correctness tidy with a measured zero delta, and calling it a fix would
be inventing a gain. The "leak" flagged earlier was real in principle and negligible in practice.

`output/t19.patch` updated to the preserving variant (194 lines). Tests green, tree restored to exact baseline.

### Superseded design note (kept for the record)

The qsearch store (search.go:2440) passes `ttPv = false`, so a qsearch entry replacing a marked node **clears**
the mark. SF preserves ttPv through qsearch stores. Whether NGN should thread it through is a real variant, not
an oversight to paper over — pick one deliberately before launch, since a mark that leaks away weakens the
whole mechanism.
