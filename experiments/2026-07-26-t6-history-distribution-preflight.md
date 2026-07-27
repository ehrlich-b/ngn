# 2026-07-26 T6 pre-flight — the live history distribution, and why the revamp doc's T6 spec was wrong

Measured locally while T20 held the box. **Zero box cost.** This is a pre-flight measurement, not a game
result. Instrumentation fully reverted; `go test -short ./engine` green on the restored tree. Shelf copies:
`output/t6-histprobe.patch`, `output/t6-histprobe.go.txt`, `output/t6-histprobe_test.go.txt`.

## Why this was run

The revamp doc (`2026-07-01-method-revamp.md:265-272`) specifies T6 as:

> CounterGo uses `r -= clamp(histSum/5000, -2, +2)` where its EMA-form history saturates entries toward
> ±16384 (sum range ±49k). **NGN's gravity-form entries cluster near 0 (equilibrium), so the tested `/2048`
> truncated to 0 almost everywhere** (the recorded "int-trunc no-op") and `/256` over-swung — **the
> 2026-06-28 rejection never tested a divisor matched to NGN's actual value distribution WITH a ±2 clamp.**
> Retest: measure the live histSum distribution, **pick divisor ≈ P90/2**, clamp ±2.

The spec names its own prerequisite — measure the distribution — so that was done first.

## Premise verified from code

`search.go:1944-1954` — NGN's current history-LMR term is a crude asymmetric step function:

```go
histScore := GetHistoryScore(move, savedLast, prev2)   // main + continuation + followup
if histScore < -500 {
    reduction++          // bad history -> reduce MORE
} else if histScore > 1000 {
    reduction--          // good history -> reduce LESS
}
```

`GetHistoryScore` (moveorder.go:284) sums three gravity-bounded tables, each capped at
`historyMax = 8192` ⇒ theoretical range **±24576**.

## Measurement

200 corpus openings replayed to their end position, one fixed-depth-11 search each,
**3.8M LMR history evaluations** sampled at the decision point itself.

### The current rule is highly active — not a dead term

| bucket | count | share |
|---|---|---|
| `histScore < -500` (reduce MORE) | 1931916 | **50.76%** |
| `histScore > +1000` (reduce LESS) | 928573 | **24.40%** |
| dead zone (no adjustment) | 945827 | 24.85% |

### Live |histScore| distribution

| percentile | value |
|---|---|
| P50 | 1450 |
| P75 | 2550 |
| P90 | **3950** |
| P95 | 5650 |
| P99 | 9550 |

## FINDING 1 — the revamp doc's stated rationale for reopening T6 is FALSE on both counts

**(a) `/2048` did NOT "truncate to 0 almost everywhere."** Measured, at divisor 2048 with the ±2 clamp the
continuous scheme produces a **nonzero adjustment on 37.20%** of LMR evaluations and **differs from the
current rule on 47.44%** of them. That is a live, large behavioral change — not an int-truncation no-op.

**(b) The 2026-06-28 test DID use the ±2 clamp.** Reading the actual run record
(`experiments/2026-06-28-lmr-conthist.md`) rather than the summary of it, the tested formula was:

```text
reduction -= clamp(histScore/2048, -2, +2)
```

— character-for-character the change T6 proposes. The claim that the prior rejection "never tested a divisor
matched to NGN's actual value distribution WITH a ±2 clamp" is wrong about the clamp, and the distribution
measurement above shows /2048 was in fact well inside the live value range.

**Both reopen rationales in the T6 spec are therefore void.** The "int-trunc no-op" phrase originated in the
2026-06-28 result line itself and was carried forward into the revamp doc as if it were a diagnosis; it was a
guess, and it is now measured false.

## FINDING 2 — but the prior evidence is statistically empty, so the lane is still open

The 2026-06-28 verdict reads "REJECT ... worse both ways", but the numbers behind it are:

| variant | result | games |
|---|---|---|
| `/2048` | -8 Elo **[-44, +28]** | **358** |
| `/256` | -45 Elo [-88, -2] | **254** |

A ±36-wide CI at 358 games cannot distinguish a +5 keep from a -20 loss; it is compatible with essentially
any outcome in the gate's range. It is also **pre-reset** (base `2dadef8`, dated the reset day), and per
CLAUDE.md pre-reset game verdicts are historical evidence only and cannot close a lane.

So T6 stays open — **not for the reasons the revamp doc gave, but because the prior test measured nothing.**
That is a weaker and more honest reopen than the one on the books.

## FINDING 3 — the doc's prescribed divisor would have silently reversed the term's direction

This is the load-bearing result. Mean signed reduction delta, positive = reduce MORE = shallower:

```text
CURRENT rule (+/-1 @ -500/+1000) : +0.2635 plies per LMR node
```

| divisor | nonzero% | differs from current% | mean delta | vs current |
|---|---|---|---|---|
| 512 | 79.49% | 67.40% | +0.3629 | +0.0993 |
| 650 | 75.86% | 61.27% | +0.3122 | +0.0487 |
| 700 | 72.47% | 60.22% | +0.2837 | +0.0201 |
| **750** | **72.47%** | **57.55%** | **+0.2701** | **+0.0066** |
| 800 | 69.24% | 56.41% | +0.2410 | -0.0226 |
| 1024 | 63.07% | 49.29% | +0.1568 | -0.1067 |
| **1975 (doc's P90/2)** | ~38% | ~47% | **~-0.07** | **~-0.33** |
| 2048 | 37.20% | 47.44% | -0.0692 | -0.3328 |
| 5000 (Counter's) | 6.42% | 69.53% | -0.0720 | -0.3356 |

**The doc's prescribed `divisor ≈ P90/2 = 1975` lands essentially on the already-tested /2048** — and at that
divisor the change is not a smoothness refinement at all. It **reverses the net direction** of NGN's
history-LMR term, from net-reduce-more (+0.264) to net-reduce-less (-0.069): a -0.33 plies/LMR-node swing.

So T6 as specified conflates two independent changes:

- **(a) shape** — step function to continuous ramp (the stated hypothesis);
- **(b) level** — a large net shift in how much LMR reduces overall (unstated, and the dominant effect).

Run together, neither a positive nor a negative verdict is interpretable. This is the same failure mode
T18b and T20 both taught: porting a donor formula imports the donor's calibration along with it. Counter's
`/5000` is matched to Counter's ±49k EMA range, its own LMR base table, and its own ordering quality.

**Both prior data points are consistent with the T3a mechanism, which explains the old result without any
int-truncation story:** `/256` (far MORE reduction than current) measured **-45**, and T3a (more reduction
via cutNode) measured **-7.9**, both matching the recorded diagnosis that at FMC ~84% extra reduction
over-reduces mis-ordered moves that then fail low silently. `/2048` (net LESS reduction) measured -8, i.e.
flat within its huge CI. The old experiment bracketed the *level* axis and never isolated the *shape* axis.

## The re-derived candidate — T6a, level-matched continuous scaling

**`reduction -= clamp(histScore/750, -2, +2)`**, replacing the -500/+1000 step rule.

Divisor **750** is chosen because it holds the mean reduction delta at **+0.2701** against the current
**+0.2635** — a residual of +0.0066 plies/LMR-node, i.e. the net reduction level is preserved and the
experiment isolates **shape alone**. It is not a donor constant and not a percentile heuristic; it is fitted
to NGN's own measured distribution to null out the confound.

Behavioral delta is large — nonzero on **72.47%** of LMR evaluations, differing from the current rule on
**57.55%** — so this is nowhere near the T7/T1f inertness failure mode, and unlike a time-management change
it **is** measurable by the standard fixed-depth nodecheck gate (LMR runs at fixed depth). Nodecheck must be
run and must move ≥~1% before this earns box time.

**Honest expectation: low single digits, plausibly zero.** Smoothing a step function that already fires on
75% of nodes is a second-order refinement, the donor prior does not transfer (different history form,
different LMR base, different ordering quality), and the one prior measurement — however underpowered —
did not point positive. It ranks where it does because the alternatives ranked lower after the audit
correction, not because it has a strong prior.

**A cheap companion falls out of the same table and should NOT be bundled:** the *level* axis has never been
tested in the de-reduction direction on a post-reset base. T3a proved more reduction hurts at 84% FMC; the
complement — less reduction — is a one-constant edit to the existing thresholds. If T6a is inconclusive, that
is the follow-up, as a separate change.

## Caveats

1. Fixed depth 11 on 200 corpus positions, single-threaded, local machine. Real games at 10+0.1 reach
   varying depths and the history tables warm differently across a full game; the distribution could shift.
   The direction-reversal result is driven by the bulk of the distribution and is unlikely to flip, but the
   exact level-matching divisor (750) is a point estimate from this sample, not a tuned constant.
2. History tables here warm from a single search per position rather than across a game's move sequence, so
   early-search evaluations are over-represented relative to real play.
3. Mean reduction delta is a per-LMR-node average; it does not account for where in the tree the change
   lands, and a ply of reduction at depth 3 is not worth a ply at depth 12.
