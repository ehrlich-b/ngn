# 2026-08-03 Amplification ratio — a candidate gate diagnostic, PROPOSED not adopted

**Zero box cost. No code changed.** Arose from a gap in my own gating of T19b: the T18b rule says *"a big
node delta from a low fire rate is a warning, not a green light"*, but nothing in the gate ever **computed**
that as a number, so T19b passed its gate without the check being applied.

## Definition

```
amplification = |nodecheck delta %| / (fire rate %)
```

i.e. **how much tree each affected node controls.** A change touching many nodes shallowly has low
amplification; one touching a handful of nodes that each gate huge subtrees has high amplification.

## The four candidates where both numbers are known

| candidate | \|delta\|% | fire % | **amplification** | outcome |
|---|---|---|---|---|
| T19 — ttPv -> LMR | 58.2 (mid) | 9.7 | **6x** | **KEPT +7.3** (full H1) |
| T12 — NMP cut-node only | 30.1 (kiwipete) | 3.1 | **10x** | **KEPT +2.2** (provisional, since certified) |
| T19b — ttPv -> LMP | 36.1 (mid) | 1.3 | **28x** | **IN FLIGHT** — see caveat |
| T18b — probcut TT store | 25.5 (kiwipete) | 0.47 | **54x** | **-22.5** (largest negative of the campaign) |

**The ordering is monotone: the two keeps sit at 6x and 10x, the worst result of the campaign at 54x.**

**Plausible mechanism, and it generalises T18b's lesson rather than restating it:** high amplification means
the change acts at a few very high-leverage nodes, so an error there propagates across an enormous subtree.
Low amplification means it acts broadly and shallowly, so individual errors average out. That is the same
intuition T18b produced ("0.47% of nodes removed a quarter of the tree = ~50x, which was measuring error
surface, not savings") expressed as a computable quantity instead of a narrative.

## Caveats — this is NOT adopted as a gate, and the reasons matter

1. **Four data points.** Setting a numeric threshold on four points would be exactly the hand-picked-constant
   error this campaign has made repeatedly (T9a's margin, T7's coefficient, T6a's divisor, T15's scaling).
   **No threshold is proposed.**
2. **T19b's row is IN FLIGHT, not a result.** At the time of writing it stands at G3825, elo -0.1, pLLR -0.04
   — a textbook null trajectory, but **the verdict is not in and this table must be re-checked against the
   actual outcome.** If T19b keeps, the monotone ordering breaks and this diagnostic is much weaker.
3. **T19's fire rate is an approximation.** It was measured at LMP sites (9.7% mid) and applied to LMR sites.
   Same node population, different consumer — close, but not the same measurement.
4. **Selection effect.** Both keeps come from this session, both from the same author, and the delta chosen
   per row is the largest-magnitude position rather than a principled aggregate.
5. **Correlation, not causation.** Four candidates that differ in many ways.

## Proposed action — gather, do not gate

**Add to the gating checklist: COMPUTE and RECORD the amplification ratio for every candidate that passes the
>=1% nodecheck gate.** It costs one division once the fire rate is known, and the fire rate is already
measured for most candidates (T12's blocked-attempt probe, T18b's `pcut` count, T19b's ttPv width).

**Do not reject on it.** Revisit once there are ~10 points with real verdicts. If the monotone relationship
survives, it becomes a genuine pre-launch filter; if it does not, this file is the record of a plausible idea
that did not hold.

**One thing it already earns:** the fire rate should be measured **at gate time, not after launch**. For
T19b it was measured only after the run started, which is why the amplification was not available when the
launch decision was made.

```yaml
id: 2026-08-03-amplification-ratio
date: 2026-08-03
change_class: proposed gate diagnostic (not adopted)
result: >
  amplification = |nodecheck delta %| / fire rate %. Across the four candidates where both are known the
  ordering is monotone with outcome: T19 6x KEPT +7.3, T12 10x KEPT +2.2, T19b 28x (in flight, trending
  null), T18b 54x -22.5.
status: GATHER ONLY -- no threshold proposed on 4 points, one of which is not yet decided.
action: compute and record amplification for every candidate passing the >=1% nodecheck gate; measure fire
  rate AT GATE TIME rather than after launch; revisit at ~10 points.
```
