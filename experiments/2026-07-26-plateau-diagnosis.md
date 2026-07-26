# 2026-07-26 Plateau diagnosis — why the mill stalled at 2700, measured

**Trigger.** The pin has moved 2632 → 2704 slowly enough to read as stuck, while Stash sits at **3374 with
zero NNUE** and Counter 4.1 at 3157 as the last pure HCE. The gap is not a classical-engine ceiling. This
measures where it actually goes, because the strategy docs driving the queue turned out to be stale.

## Finding 1 — the EBF lane MOVED, and every recorded number for it was out of date

`scripts/ebfprobe.py`, depth reached at 400K fixed nodes, HEAD vs the script's own standing baseline
(2026-06-11, post-W1):

| Position | Jun-11 baseline | HEAD (2026-07-26) | Δ plies |
|---|---|---|---|
| quiet-mg-QGD | 12 | **14** | +2 |
| quiet-mg-closed | 9 | **12** | +3 |
| lateMg-ph6-11 | 11 | **13** | +2 |
| najdorf-mg | 9 | **14** | +5 |
| rook-eg-R4P | 14 | **17** | +3 |
| pawn-eg | 24 | **32** | +8 |

Implied EBF (N^(1/d)) at HEAD: QGD 2.38, closed 2.73, lateMg 2.70, najdorf 2.51, rook-eg 2.12, pawn-eg 1.49.

**Middlegame EBF is now ~2.4-2.7, NOT the ~3.0 recorded in `project_ebf_lane` and the ebfprobe header.**
Both are stale by ~6 weeks of kept work (T5 aspiration, the corrhist family, T1b/T1e). The lane was never
"exhausted" — it has been quietly delivering, and nobody re-measured.

**The remaining gap is still the whole ballgame.** Elite class is ~2.0. At QGD's 183331 nodes, EBF 2.5 → 2.0
moves depth 14 → **17.5**. With d12-vs-d10 measured at **+185 Elo**, three-plus plies at equal nodes dwarfs
anything the +4-median heuristic mill produces. **2800 is an EBF problem, not an eval problem.**

## Finding 2 — ordering is TT-starved, and that is the specific lever

Measured on HEAD via the `sd_order` / `sd_band` diagnostics:

| Metric | kiwipete d12 | mid d12 |
|---|---|---|
| First-move cutoff rate (fmc/bcut) | 25896/29170 = **88.8%** | 13488/16177 = **83.4%** |
| Move-loop nodes WITH a TT move (ttlist/mln) | 23535/49522 = **47.5%** | 9323/27565 = **33.8%** |
| Cutoffs attributable to the TT move | 12876 = **44%** of cutoffs | 4399 = **27%** |
| Cutoffs from captures | 13656 = 47% | 6802 = 42% |
| Cutoffs from killers / counter / history | 1447 / 292 / 684 = 8.3% | 3210 / 852 / 914 = 30.8% |

**The headline: the TT move is the single strongest ordering signal — 27-44% of all cutoffs — yet it is
present at only 34-47% of move-loop nodes.** Everything that raises TT-move availability raises FMC directly,
and FMC is what gates the entire T3 LMR-reduction family (T3a proved reducing more at 84% FMC costs -7.9).

FMC by remaining-depth band (kiwipete): d1-2 **87.9%**, d3-5 90.3%, d6-9 93.2%, d10+ 96.7%.
mid: d1-2 **83.8%**, d3-5 82.2%, d6-9 82.2%.

**Ordering is WORST at d1-2**, which is where the overwhelming majority of nodes live, so gains there compound
hardest. The quiet/positional `mid` position is ~5 points worse than tactical kiwipete across every band —
consistent with the loss autopsy's "positional slow-bleed" finding.

## Finding 3 — this retro-explains the whole campaign

Sorting every result since the reset by what it touched:

- **Search-efficiency / TT changes**: T5 aspiration **+20.2** (biggest keep of the campaign), T1b **+13.1**,
  T4 corrhist **+9.1**. Plus the strongest gated candidate now queued, T18b, at **-25.5%/-28.5% nodes**.
- **Pruning-constant changes**: T7 **0**, T10b **0**, T9a **-5.2**, T1c **-4.8**, T13 **-0.4**, T16 **-0.6**,
  T3a **-7.9**.

**Every meaningful win came from search efficiency; nearly every null and negative came from tuning pruning
constants.** The mill has been fishing in the wrong pond, and the +4-median model was measured on the wrong
population.

## Strategic consequence — reorder the whole queue around EBF/TT

**Promote (touch TT retention or ordering, i.e. move EBF toward 2.0):**

1. **T18b** — probcut TT store. RUNNING. Adds proved bounds to the TT, which raises ttlist directly. Its
   -25%/-28% node reduction is exactly the predicted signature.
2. **T11a — 4-way TT clustering.** Was DOWNGRADED to "+7.5 prior, aging/depth-preferred already exist". That
   downgrade judged the wrong axis: the question is not replacement policy quality but **retention under
   pressure**, and ttlist at 34-47% is the direct measurement of the deficit. **Re-promote to top structural
   candidate.** Note the fixed-node probe UNDERSTATES this (350K nodes into 8.4M slots is 4% occupancy) —
   real games at 10+0.1 accumulate millions of nodes across a game and actually pressure the table, which is
   why a games gate rather than a probe must decide it.
3. **T19 — ttPv.** Already implemented/gated; also TT-borne.
4. **T10 ordering family** — the FMC lever, and the gate on the entire T3 LMR lane. T10b's uniform-weight
   failure did not test the lane, only that one weighting.

**Demote:** further pruning-constant work (T15, the T7/T9a reopens, cutoffCnt) and the eval-shape lane.
Constant-tuning is where six of seven nulls came from.

**T8 SPSA stays last** — it tunes constants, and by this diagnosis constants are the low-yield axis; it also
must run after the structural queue stops moving the base (see its abort record).

## Actions taken from this diagnosis

- Stale numbers corrected on sight: `scripts/ebfprobe.py` header baseline and the `project_ebf_lane` memory
  both said mg EBF ~3.0 / depths 9-12. Both now record the measured HEAD values.
- Queue reordered in `TODO.md` around the EBF/TT axis rather than prior size.
- **Re-measure cadence added**: ebfprobe + the fmc/ttlist table are cheap (~4 min local, zero box time) and
  must be re-read after every structural keep. Letting them go stale for six weeks is what allowed the "lane
  exhausted" story to persist while the lane was in fact delivering.
