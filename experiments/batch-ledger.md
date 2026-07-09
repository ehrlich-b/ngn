# Batch-Certification Ledger

The sanctioned route (CLAUDE.md / TODO item 10, 2026-07-01) for capped-but-positive results.
A result with **point estimate ≥ +1, LLR > 0, and no regression signal** may enter main as a
PROVISIONAL keep instead of being shelved-at-cap. After **≤4 provisionals OR 2 weeks**, ONE
powered real-clock SPRT **[0, +6]** of the post-batch HEAD vs the pre-batch base certifies the
whole batch at once (or, on failure, a bisect over the batch finds the regressor). This recovers
the +2–8 self-play class the classical climb is made of (Stash ≈ 100 keeps at median +4.5) without
keeping speculative changes on sign-at-cap noise.

Rules:
- A provisional's own verdict must be a COMPLETED real-clock self-play run (never a killed/mid-run SPRT).
- Certification run needs its own A/A preflight at the exact TC/concurrency/machine.
- Fail → bisect the batch (each provisional isolable via its commit) and drop the regressor(s).
- Certified provisionals graduate to permanent keeps; the pre-batch base advances to the certified HEAD.

---

## Batch 1 — OPEN (pre-batch base: `ea69ad6`, opened 2026-07-01)

| # | Change | Commit | Verdict (real-clock self-play, 10+0.1) | Run record |
|---|---|---|---|---|
| 1 | iir3 — phase-guarded cutNode-IIR (missing standard technique) | `7394668` | penta +2.0 [-3,+7], pLLR +1.71, 0 flag-outs, 6000g (capped-but-positive; fixed-nodes +3.6 concurring) | `2026-06-29-iir.md` |

- **Trigger:** certify when this batch reaches 4 provisionals **OR** by **2026-07-15**, whichever first.
- **Certification run:** post-batch HEAD vs `ea69ad6`, real-clock `-tc 10+0.1`, SPRT **[0, +6]**, A/A preflight required, on the LAN 9800X3D box.
- **What the [0,+6] gate certifies (coordinator, 2026-07-08):** the HEAD-vs-`ea69ad6` composite SPRT is a
  WHOLE-BATCH gate on the NET post-`ea69ad6` transfer (iir3 + T4 + T4-nonpawn + T1b together), consistent
  with the 2689 [2649,2729] gauntlet point-signal (+23 vs 2666, inside the ±40 delta floor). It is NOT an
  iir3-isolation test: iir3's per-change evidence is its OWN completed real-clock c4 run (+2.0 [-3,+7],
  6000g, 0 flag-outs; `2026-06-29-iir.md`) and **no further iir3-specific SPRT is planned.** On PASS the
  whole batch stands; on FAIL the existing bisect-on-fail path isolates the regressor by commit.
- **On fail:** bisect batch-1 commits (`7394668`, …) to find the regressor; drop it, re-certify.
- **Note (2026-07-02) — main also carries T4 (FULL H1 accept, NOT a provisional):** the pawn corrhist
  re-plumb landed on main as an outright SPRT accept (+9.1 penta [+0,+18], pLLR +2.80, 0/0 flag-outs /
  2325g, real-clock 10+0.1 c8; `2026-07-02-t4-corrhist-replumb.md`). Because the certification run measures
  post-batch HEAD vs `ea69ad6`, the certified composite will INCLUDE T4's gain alongside the iir3
  provisional's; the bisect-on-fail rule already covers attribution (T4 is isolable by its own commit).
- **Note (2026-07-02) — main also carries T4-nonpawn (FULL H1 accept, NOT a provisional):** the non-pawn
  corrhist add on the fixed T4 plumbing landed on main as an outright SPRT accept (+6.4 penta [-1,+14],
  pLLR crossed +2.94 at G3796, 0/0 flag-outs / 3804g, real-clock 10+0.1 c8; `2026-07-02-t4-nonpawn.md`) —
  under a STRICT gate (H1-only, NO provisional-on-cap). Same attribution note as T4: the post-batch
  certification composite (vs `ea69ad6`) will include this gain too, and it is isolable by its own commit
  for the bisect-on-fail path.
- **Note (2026-07-03) — main also carries T1b (FULL H1 accept, NOT a provisional):** symmetric
  stability-scaling of the soft time budget (wire the dead `stableIters` read) landed on main as an
  outright SPRT accept (+13.1 penta [+3,+23], pLLR crossed +2.94 at G2225, 0/0 flag-outs / 2233g,
  real-clock 10+0.1 c8 adjudicated; `2026-07-03-t1b-stability-scaling.md`). Same attribution note as
  T4/T4-nonpawn: the post-batch certification composite (vs `ea69ad6`) will include this gain too, and it
  is isolable by its own commit for the bisect-on-fail path. Batch 1 still holds ONE provisional (iir3);
  T4 / T4-nonpawn / T1b are all full H1 accepts. Cumulative claimed self-play since `ea69ad6` is now
  ~+28-30 (T4 +9.1, T4-nonpawn +6.4, T1b +13.1) — past the +15-20 milestone-gauntlet cadence trigger.
