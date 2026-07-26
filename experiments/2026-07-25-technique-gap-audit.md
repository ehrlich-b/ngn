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
