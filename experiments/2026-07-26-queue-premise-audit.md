# 2026-07-26 Queue-premise audit — where NGN's Elo has actually come from

**Written while T20 is live, deliberately before its verdict**, so the analysis cannot be reverse-engineered
from that result. Zero box cost. This is an audit of *candidate selection*, not of the instrument.

The instrument question is **already closed and must not be re-ground**: the Batch-2 composite measured
**+35.1 [+19,+51]** against a claimed +26.8 on 2026-07-25 (`2026-07-24-batch2-cert.md`), Batch 1 measured
+43.2 against a claimed +34.3, and both exceeded claim. Self-play keeps transfer; the ruler is sound. So the
six-candidate dry streak is a candidate-quality fact and the question is *which candidates*.

## Every certified keep, by family

From `batch-ledger.md` (Batch 1 + Batch 2, all real-clock 10+0.1 c8):

| Change | Penta Elo | Family |
|---|---|---|
| T5 — aspiration modernization | **+20.2** | root search control |
| T1b — stability-scaling of soft time budget | **+13.1** | time management |
| T4 — pawn correction-history re-plumb | **+9.1** | eval correction (corrhist) |
| T1e — node-fraction best-move effort scaling | **+6.6** | time management |
| T4np — non-pawn correction-history | **+6.4** | eval correction (corrhist) |
| T4c — minor-piece correction-history | **+3.7** | eval correction (corrhist) |
| iir3 — phase-guarded cutNode-IIR | **+2.0** | **node-level search heuristic** |

## Every shelve/reject, by family

| Change | Result | Family |
|---|---|---|
| T18b — probcut TT store | **-22.5** | node-level |
| T1a — drop soft-stop projection | -7.8 | time management |
| T9a — singular double extension | -5.2 | node-level |
| T3b — capture LMR | -3.1 | node-level |
| T10b — 4-ply continuation history | -2.4 | node-level |
| T17 — `improving` reference fix | H0 reject | node-level |
| T16 — s7 depth gate | -0.6 | node-level |
| T13 — qsearch stand-pat TT store | -0.4 | node-level |
| T7 — history pruning depth<=6 | 0 | node-level |
| T3a — LMR cutNode | reject | node-level |
| T1c — score regimes | H0 reject | time management |

## The finding

| Family | Record | Total Elo |
|---|---|---|
| **node-level pruning/reduction/ordering** | **1 keep / 10 attempts** | **+2.0** |
| time management | 2 / 4 | +19.7 |
| correction history | 3 / 3 | +19.2 |
| root search control | 1 / 1 | +20.2 |
| **everything except node-level** | **6 / 8** | **+59.1** |

**The campaign's entire node-level transplant program has produced +2.0 Elo across ten attempts.** Roughly
97% of certified Elo came from three families the queue has largely stopped drawing from. And the current
queue — T15, T12, T11a, T19, plus the live T20 — is **almost entirely node-level**.

## Why — the mechanism, not just the correlation

This is not "node-level techniques are bad." They are what strong engines are made of. It is that **NGN's
node-level stack is already dense, and density is the thing that saturates.**

The corroborating measurement is independent and already in `TODO.md`: **Counter 3.8 at CCRL 2994 — ~290 Elo
above NGN — runs a LEANER node-level stack than NGN does** (no corrhist, no capture-history, no multicut, no
double-extension, no verification search; IID rather than IIR). Its measured differential against NGN is the
**three-regime stability/score-drop time manager**, LMR *shape*, and draw scaling.

So the engine 290 Elo ahead of us got there with *fewer* node-level rules and a *better* time manager. Every
node-level prune/reduce rule competes for the same nodes: probcut, history pruning, continuation history,
capture-LMR, double extensions and SEE pruning all decide what to cut in overlapping regions of the same
tree. Adding the ninth such rule to an already-dense stack has little left to cut — which is exactly the
observed pattern of near-zero capped nulls (T7 = 0, T13 = -0.4, T16 = -0.6, T10b = -2.4, T3b = -3.1), all
with envelopes that never approached a bound. Those are not near-misses. That is the signature of a
saturated mechanism.

T18b's **-22.5** is the same story at the other extreme: when a dense stack does have room, what fits is
usually something that *persists speculative information*, and it back-fires hard.

## Honest caveats — where this argument is weaker than it looks

1. **Ordering confound.** The node-level items ran *later*, against a base already carrying the time-mgmt and
   corrhist gains. A tougher base does not by itself shrink a true +5 (SPRT measures the delta), but
   **redundancy does** — and redundancy is precisely the proposed mechanism, so this confound and the
   hypothesis are not independent. It cannot be fully separated with existing data.
2. **Small samples per family.** Root search control is 1-for-1. That is not a family, it is one result.
3. **Time management is not automatic** — 2-for-4, with T1a at -7.8 and T1c a clean reject. "Go back to the
   time manager" is not a guaranteed lane, it is a better-odds one.
4. **Selection effects in what got queued.** Cheap/tractable node-level items may have been preferred
   precisely because they were easy to specify, independent of their prior.

## What this implies for the queue — NOT yet applied

No queue change is made in this document. T20 owns the box and its verdict comes first; if T20 keeps, the
node-level lane has a live counterexample and this audit weakens considerably.

If T20 caps null, the proposed reorder is:

1. **Counter's three-regime stability/score-drop time manager** (exact spec in the revamp doc's T1b section)
   — the measured differential against a 2994-rated engine, in the family with the campaign's second-best
   record, and TC-gated so it is real-clock testable by construction.
2. **T6 — LMR shape with Counter's saturating-EMA continuous history scaling.** The spec explains *why*
   NGN's `/2048` retest no-op'd (gravity-equilibrium history values vs Counter's saturating EMA), so this is
   a shape fix with a diagnosed prior failure, not another prune-rule transplant.
3. **T11a (TT retention) survives the audit on mechanism** even though it is nominally node-level: it raises
   TT-move *availability* (`ttlist`), which the plateau diagnosis identified as the specific deficit
   (TT move drives 27-44% of cutoffs from only 34-47% availability). That is information retention, not
   another rule competing to cut the same nodes.
4. **T15/T12/T19 drop below all of the above** — they are additional node-level rules in a saturated stack.

The one thing this audit should NOT be used for: closing the node-level lane. Ten attempts with one small
keep is evidence of low yield, not of impossibility, and `iir3` proves the lane can produce. It is a
ranking argument, not a closure.
