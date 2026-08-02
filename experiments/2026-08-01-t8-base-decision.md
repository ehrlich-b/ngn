# 2026-08-01 T8 base decision — RESOLVED: run SPSA on the T17-FIXED base (T17b)

**DECIDED AND PREPARED. Zero box hours. Tuning base built, gated, staged, on-box hash verified. Tree clean.**

This decision is **time-critical and one-way**: once T8 converges on the unfixed base, its vector is
entangled with a proven defect and T17 can never afterward be isolated. It is resolved **now**, while T12
holds the box, rather than at launch time.

## The three premises — RE-VERIFIED against HEAD, not trusted from the queue

Two queue claims already went stale tonight (T12's shelved-patch pointer did not exist; the M6 throughput
paragraph was obsolete), so every premise was re-checked:

1. **`output/t17.patch` still applies clean** to HEAD. Yes — 2 files, +102/-2.
2. **All five `improving` consumers are live SPSA dims.** Verified against the registry in `search.go`,
   which is **14 dims**: `NullMoveR, NullMoveMinDepth, LMRFullDepth, LMRMoveThreshold, FutilityMargin,
   FutilityMaxDepth, ReverseFutilityMargin, DeltaMargin, SingularDepth, SingularMargin, ExtensionBudget,
   AspInit, AspMult, LMPBase`. The five consumers — **LMRFullDepth, LMRMoveThreshold, FutilityMargin,
   ReverseFutilityMargin, LMPBase** — are all present.
3. **`cmd/spsa` has no dim-subset flag** and iterates the whole registry (`cmd/spsa/main.go:84`,
   `for _, tp := range engine.TunableSearchParams`). Confirmed: the flag set is
   engine/tc/iters/single/pairs/openings/out/resume/rend/afrac/report/seed/lowpower/selftest — **no `-dims`**.

**Therefore running the EXISTING 14-dim T8 on the FIXED base IS the interaction test T17 pre-declared**
("fix + SPSA over the four margins as ONE explicit interaction test"), needs **no new code**, and costs
**exactly the same** as the T8 round already planned. Only the base changes.

## Decision: FIXED base

**Reasoning, in order of weight:**

1. **Irreversibility asymmetry — the decisive argument.** Tuning on the *unfixed* base permanently forecloses
   T17: the converged vector adapts 5 knobs to a corrupted input, and no later run can separate them. Tuning
   on the *fixed* base is recoverable — if it disappoints, T8-on-unfixed can still be run later at the cost
   of another round. **Identical cost, one option reversible and one not.**
2. **The defect is proven, not suspected.** Slot 0 measured 0 after a full depth-8 search while slots 1/2/3
   held 653/419/271; 1068 sentinel references in the endgame position, **100% of them spuriously
   `improving=true`**. Both bugs are real and mechanism-proven.
3. **It is mechanism-level work, which is where this campaign's keeps come from.** Keeps replaced or added
   whole mechanisms (T5 aspiration, T4 corrhist, T1b/T1e); misses adjusted single sites or constants (T7,
   T10b, T9a, T13, T16, T3a, T1c, T18b, T20, T4d, T4e). *"Repair a proven defect and recalibrate its
   consumers"* sits in the first category; *"tune 14 constants on a known-corrupted input"* sits squarely in
   the second.
4. The defect's cost is **measured at ~6 Elo** (T17 standalone = -6.4), so this is not a hypothetical
   cleanup.

## The counter-argument, which TODO's analysis did NOT record — stating it because it is real

**The fixed base starts ~6.4 Elo in the hole.** The confirmation SPRT is *(fix + tuned vector) vs unmodified
HEAD*, so the tune must find **more than +6.4** merely to break even, whereas T8-on-unfixed starts from 0.
That is a genuine handicap and it was not mentioned in the queue's write-up of this decision.

**Why it does not flip the decision:** the -6.4 was measured with the consumers *not* retuned, which is the
entire premise of the reopen. The corrected `improving` signal is strictly more informative than the
corrupted one (the corrupted form computes "is the eval positive" at ply 2 and "always true" after checks),
so a well-tuned engine on the correct signal should dominate a well-tuned engine on the corrupted one *in
principle*. **The honest risk is that SPSA does not find that optimum** — 14 dims, a noisy objective, and no
guarantee of reaching a global optimum. That risk is recorded here rather than discovered later.

**Accepted, explicitly:** a negative outcome is **fix-or-tune ambiguous**. That ambiguity is inherent to any
interaction test and is sanctioned by CLAUDE.md when the experiment is explicitly about an interaction. The
alternative — cementing a proven, measured, 6-Elo-costly defect into the tuned vector forever — is worse.

## Prepared artifacts

- **Tuning base `ngn_t17fix.exe` sha256 `3e8016d7dd2bf6f76a5a405b21f69367d8727a4f6af5fb9922815267c538e83f`**,
  pushed to the box, **on-box hash VERIFIED equal**.
- `spsa.exe` on box re-verified at `b92d3c50…` (the M4-parallel tuner).
- Gates on the fixed base: **nodecheck -28.7% / +39.6% / -7.3%** (247063 / 208825 / 709959 vs the locked
  346662 / 149587 / 765656) — the fix moves the tree decisively, as it must by construction.
  `go test -short ./engine`, `-race ./engine`, `./...` all **green**, including the patch's own regression
  **`TestImprovingReferenceIsWritten` (PASS)**.
- Engine tree **reverted**; nodecheck re-verified at exactly 346662/149587/765656.

## Standing rules for the run itself — unchanged, restated so they are not lost

- **T8 IS A TUNER, NOT A VERDICT.** No keep/shelve may be read from `plus_pct` or from param movement — that
  is the exact failure the policy names, and it is how the 2026-06-28 joint-SPSA run was invalidated.
- Evidence requires **params verified live in code** and **>= 20k games**.
- The converged vector gets a **separate real-clock 10+0.1 c8 SPRT [-3,+3] vs unmodified HEAD**, and only
  that decides.
- **5+0.05 is NOT available** — M6-retest failed 2026-08-01 and the TC is closed, so this round runs at
  **10+0.1** and costs ~25h, not the ~12h an M6 pass would have bought.
- If the converged vector lands within rounding of the defaults, the finding is "already near a local
  optimum" and the lane closes without a second run.

```yaml
id: 2026-08-01-t8-base-decision
date: 2026-08-01
change_class: decision + preparation (no verdict)
decision: run T8 SPSA on the T17-FIXED base (T17b), not the unfixed base
premises_reverified: [t17 patch applies, 5 consumers are live SPSA dims, cmd/spsa has no dim-subset flag]
tuning_base_sha256: 3e8016d7dd2bf6f76a5a405b21f69367d8727a4f6af5fb9922815267c538e83f
box_cost: zero
next_action: >
  Launch only when the T-queue (T12 live, then T19, T15) is exhausted. ~25h at 10+0.1. Verdict is a separate
  SPRT on the converged vector vs unmodified HEAD, never plus_pct.
```

---

## DIM LIVENESS PRE-FLIGHT on the fixed base — all 14 LIVE, verified empirically 2026-08-01

**Why this was run rather than assumed:** the 2026-06-28 joint-SPSA round was invalidated precisely because
it tuned knobs that could not move (9/11 params pinned, sub-integer steps, `plus_pct` ~50% = zero gradient),
and the T17 fix changes how **five** of these dims' consumers read `improving`. Launching a **25-hour** run
on an unverified registry is the one pre-flight worth its cost. The T8 round-1 manifest verified 13 dims on
the *old* base; this re-verifies all 14 on the base that will actually be tuned.

Method: `setoption name <dim> value <min|max>` over UCI, then fixed-depth node count at kiwipete d12. Default
(T17-fixed base) = **247063** nodes.

| dim | default | at MIN | at MAX | status |
|---|---|---|---|---|
| NullMoveR | 247063 | 190988 | 302102 | LIVE |
| NullMoveMinDepth | 247063 | 442788 | 351834 | LIVE |
| LMRFullDepth | 247063 | **247063** | 455894 | LIVE (flat lower half — see note) |
| LMRMoveThreshold | 247063 | 436676 | 360584 | LIVE |
| FutilityMargin | 247063 | 265883 | 398545 | LIVE |
| FutilityMaxDepth | 247063 | 366722 | **247063** | LIVE (max == default — see note) |
| ReverseFutilityMargin | 247063 | 159005 | 450336 | LIVE |
| DeltaMargin | 247063 | 343380 | 404835 | LIVE |
| SingularDepth | 247063 | 394239 | 303979 | LIVE |
| SingularMargin | 247063 | 379137 | 187648 | LIVE |
| ExtensionBudget | 247063 | 320520 | **247063** | LIVE (rarely binds high — see note) |
| AspInit | 247063 | 570153 | 266354 | LIVE |
| AspMult | 247063 | **891106** | 298341 | LIVE |
| LMPBase | 247063 | 320575 | 372896 | LIVE |

**No dead dims. Every one of the 14 moves the tree on at least one side.**

**The three flat sides are all explained, and none is a defect:**

- **`LMRFullDepth` at min=1 reproduces the default exactly.** This is the *already-recorded* flat lower half
  (`1 == default 3` in effect) from the T8 round-1 manifest — reproduced here on the new base, so it is a
  property of the parameter's shape, not of the T17 fix.
- **`FutilityMaxDepth` at max=8 equals the default because the default IS 8** — the dim's range is
  `[1, 8]` and it already sits at its ceiling. Only the downward direction can move, and it does (366722).
- **`ExtensionBudget` at max=64 equals the default**, matching the recorded observation that it *rarely binds
  above 24*. The downward side moves (320520), so the dim is live.

**Also worth recording: `AspMult` is by far the most sensitive dim** — at min=110 the tree is **891106 nodes,
3.6x the default**. Aspiration behaviour dominates this parameter set, which is consistent with T5
(aspiration modernization) being the single biggest keep of the campaign at +20.2. **SPSA will feel this dim
hardest**, and a converged vector that moves `AspMult` far from 150 should be treated with suspicion until
the confirmation SPRT says otherwise — T5's window shape was games-validated and is not obviously improvable
by a noisy 14-dim search.

**Pre-flight verdict: the registry is sound on the fixed base and the 25-hour run is not at risk of tuning
dead knobs.** Tree reverted after the sweep; nodecheck re-verified at exactly 346662/149587/765656.
