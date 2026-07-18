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
| 1 | iir3 — phase-guarded cutNode-IIR (missing standard technique) | `7394668` | **PROVISIONAL** — penta +2.0 [-3,+7], pLLR +1.71, 0 flag-outs, 6000g (capped-but-positive; fixed-nodes +3.6 concurring) | `2026-06-29-iir.md` |
| 2 | T4 — pawn correction-history re-plumb | `ffb24af` | FULL H1 accept — penta +9.1 [+0,+18], pLLR +2.80, 0/0 flag-outs, 2325g | `2026-07-02-t4-corrhist-replumb.md` |
| 3 | T4np — non-pawn correction-history add | `13560dc` | FULL H1 accept — penta +6.4 [-1,+14], pLLR crossed +2.94 at G3796, 0/0 flag-outs, 3804g | `2026-07-02-t4-nonpawn.md` |
| 4 | T1b — symmetric stability-scaling of soft time budget | `35c0c0a` | FULL H1 accept — penta +13.1 [+3,+23], pLLR crossed +2.94 at G2225, 0/0 flag-outs, 2233g | `2026-07-03-t1b-stability-scaling.md` |
| 5 | T4c — minor-piece correction-history add | `a195717` | FULL H1 accept — penta +3.7 [-2,+9], pLLR +2.98, 0/0 flag-outs, 7842g | `2026-07-03-t4c-minor-corrhist.md` |

- **Batch net (claimed self-play):** **+34.3** across 5 keeps = iir3 +2.0 (provisional) + T4 +9.1 + T4np +6.4 + T1b +13.1 + T4c +3.7 (4 full H1 accepts). Post-batch HEAD `c514a2b` engine == the T4c kept tree `a195717` (T13/T1c/T16 all reverted, `git diff a195717..c514a2b -- engine/` empty).
- **Trigger:** certify when this batch reaches 4 provisionals **OR** by **2026-07-15**, whichever first. **Date trigger reached; certification SPRT launching 2026-07-18** (3 days past the 2026-07-15 date trigger) — manifest `2026-07-18-batch1-cert.md`, post-batch HEAD `c514a2b` vs pre-batch base `ea69ad6`, SPRT [0,+6] on the LAN 9800X3D box.
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
- **Note (2026-07-09) — main also carries T4c (FULL H1 accept, NOT a provisional):** the minor-piece
  corrhist add (third corrector, on the fixed T4 plumbing) landed on main as an outright SPRT accept
  (+3.7 penta [-2,+9], pLLR +2.98 ≥ +2.94, 0/0 flag-outs / 7842g, real-clock 10+0.1 c8 adjudicated;
  `2026-07-03-t4c-minor-corrhist.md`) — the corrhist family is 3-for-3 on the fixed plumbing (pawn +9.1,
  non-pawn +6.4, minor +3.7, diminishing as predicted). Same attribution note as T4/T4np/T1b: the
  post-batch certification composite (vs `ea69ad6`) includes this gain too, isolable by commit `a195717`
  for the bisect-on-fail path. Cumulative claimed self-play since `ea69ad6` is now **+34.3** (iir3 +2.0
  provisional + T4 +9.1 + T4np +6.4 + T1b +13.1 + T4c +3.7). Post-T4c items T13/T1c/T16 were all shelved
  and reverted, so post-batch HEAD `c514a2b` engine is byte-identical to the T4c keep `a195717`.
- **Certification (2026-07-18):** launching now, 3 days past the 2026-07-15 date trigger. Manifest
  `2026-07-18-batch1-cert.md`; post-batch HEAD `c514a2b` (== `a195717` engine) vs pre-batch base
  `ea69ad6`, SPRT [0,+6] (pLLR bounds ±2.94), real-clock 10+0.1 c8 adjudicated, maxgames 8000, on the LAN
  9800X3D Windows box. PASS (pLLR ≥ +2.94) → whole batch certified, iir3 graduates from provisional,
  pre-batch base pointer advances to `c514a2b`. FAIL (pLLR ≤ −2.94) → bisect over
  {`7394668`, `ffb24af`, `13560dc`, `35c0c0a`, `a195717`}. Inconclusive at cap → escalate to Bryan (no
  auto-action; the cert is a safety instrument, it does not auto-pass on sign).
