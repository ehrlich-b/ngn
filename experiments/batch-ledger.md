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

## Batch 1 — CERTIFIED 2026-07-18 (pre-batch base: `ea69ad6`, opened 2026-07-01, certified HEAD: `c514a2b`)

**CERTIFIED 2026-07-18** — composite `c514a2b` vs `ea69ad6` SPRT [0,+6] = **+43.2 penta [+26,+61], pLLR +3.09
>= +2.94, H1 ACCEPTED, 745g / 372 pairs, 0/0 flag-outs, 46m21s, DONE_EXIT_0** (`2026-07-18-batch1-cert.md`).
The predeclared PASS rule fired: **iir3 graduated from provisional to permanent keep**, and the **pre-batch
base pointer advanced from `ea69ad6` to `c514a2b`**. Measured composite +43.2 exceeds the +34.3 claimed
self-play sum. No bisect (PASS, not FAIL).

| # | Change | Commit | Verdict (real-clock self-play, 10+0.1) | Run record |
|---|---|---|---|---|
| 1 | iir3 — phase-guarded cutNode-IIR (missing standard technique) | `7394668` | **CERTIFIED (graduated from provisional 2026-07-18)** — own run penta +2.0 [-3,+7], pLLR +1.71, 0 flag-outs, 6000g; confirmed inside the certified +43.2 composite | `2026-06-29-iir.md` |
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
- **Certification (2026-07-18) — FIRED, CERTIFIED:** manifest `2026-07-18-batch1-cert.md`; post-batch HEAD
  `c514a2b` (== `a195717` engine) vs pre-batch base `ea69ad6`, SPRT [0,+6] (pLLR bounds ±2.94), real-clock
  10+0.1 c8 adjudicated, on the LAN 9800X3D Windows box. Result **+43.2 penta [+26,+61], pLLR +3.09, H1
  ACCEPTED, 745g, 0/0 flag-outs, 46m21s, DONE_EXIT_0**. The PASS rule fired: whole batch certified, iir3
  graduated from provisional to permanent, pre-batch base pointer advanced to `c514a2b`. No bisect.

---

## Batch 2 — CERTIFIED 2026-07-25 (pre-batch base: `c514a2b`, opened 2026-07-18, certified HEAD: `172e9e1`)

**CERTIFIED 2026-07-25** — composite `172e9e1` (engine == `5b93966`) vs `c514a2b`, SPRT [0,+6] = **+35.1 penta
[+19,+51], pLLR +2.88 with the +2.94 bound crossed at G777 (max +2.98 @G781), H1 ACCEPTED, 785g / 392 pairs,
0/0 flag-outs, 48m36s, DONE_EXIT_0** (`2026-07-24-batch2-cert.md`). Measured **+35.1 exceeds the +26.8 claimed**
self-play sum — the second straight certification to overshoot its claim (Batch 1: +34.3 claimed → +43.2
measured). PASS, so **no bisect**; T1e and T5 both graduate to permanent certified keeps and the **certified
base pointer advances `c514a2b` → `172e9e1`**. Certified base binary staged on the box as `ngn_cert2.exe`
(`0a8f65b7…`, hash-verified after copy); `ngn_cert.exe` (Batch-1 base) deliberately left in place so an older
bisect remains reachable.

**Why this result mattered beyond the number:** it landed directly after three consecutive T-item shelves
(T10b, T7, T9a), when the live hypothesis was that self-play keeps were not transferring and that T5's +20.2 —
the campaign's biggest single keep — would be the first suspect in a bisect. The composite came in *above*
claim instead, which clears the instrument and the keep rule and localizes the recent nulls to candidate
quality. **Batch certification is now 2-for-2, both times overshooting**, mild evidence that isolated
per-change SPRTs systematically *understate* keeps (each is measured against a base already containing the
others' gains, and each stops at the first bound crossing rather than estimating precisely).

Certified base `c514a2b` is engine-identical through current HEAD `f5b7c34` (the `e7bfd46`/`f5b7c34` delta is
doc-only). Batch 2 accumulates the next capped-but-positive PROVISIONAL keeps against this base; full H1
accepts also land on main and are folded into this batch's composite for attribution, same as Batch 1.

| # | Change | Commit | Verdict (real-clock self-play, 10+0.1) | Run record |
|---|---|---|---|---|
| 1 | T1e — node-fraction (best-move node-effort) time scaling | `801d384` | FULL H1 accept — penta +6.6 [-1,+14], pLLR peak +2.96 crossed +2.94 at G4218, c8-drain to +2.85, 0/0 flag-outs, 4226g (2026-07-19) | `2026-07-18-t1e-node-effort.md` |
| 2 | T5 — aspiration modernization (scaled init + progressive widening + never-discard) | this commit | FULL H1 accept — penta +20.2 [+7,+33], pLLR peak +2.98 crossed +2.94 at G1202 (sustained G1299–1305), c8-drain to +2.88, 0/0 flag-outs, 1309g (2026-07-19) | `2026-07-19-t5-aspiration.md` |

- **Trigger fired:** the 2-week date trigger (2026-08-01), at 0 provisionals — certified 6 days early, 2026-07-25.
- **Certification run:** DONE — `172e9e1` vs `c514a2b`, real-clock `-tc 10+0.1` c8, SPRT [0,+6], standing M5 A/A basis, LAN 9800X3D box.
- **On fail:** n/a — PASSED.
- **Batch net (claimed self-play):** **+26.8** (T1e +6.6 + T5 +20.2, both FULL H1 accepts — landed on main, not provisionals; folded into the composite for attribution same as Batch 1's full accepts). 0 provisionals so far.
- **Note (2026-07-19) — T1e (FULL H1 accept, NOT a provisional):** node-fraction best-move effort scaling of the
  soft time budget (SF-standard third time signal, composed multiplicatively with T1b's stability factor under
  the T1b-proven [0.72,1.40] clamp) landed on main as an outright SPRT accept: penta **+6.6 [-1,+14]**, pLLR
  peak **+2.96 crossed the +2.94 bound at G4218** (first & only crossing, c8-drain to +2.85 at 4226g), 0/0
  flag-outs / 4226g, real-clock 10+0.1 c8 adjudicated (`2026-07-18-t1e-node-effort.md`) — under the STRICT
  per-change gate (H1-only, NO provisional-on-cap). Isolated SPRT [-3,+3] vs `c514a2b`. Fixed-depth
  node-identical (nodecheck 230036/181982/587557 unchanged), so no nodecheck re-lock; hash-back reproduced the
  exact playing binary `f65a3c16…`. The post-batch certification composite (vs `c514a2b`) will include this
  gain, isolable by this commit for the bisect-on-fail path.
- **Note (2026-07-19) — T5 (FULL H1 accept, NOT a provisional):** aspiration-window modernization (replace the
  fixed 50cp / jump-to-±INFINITY / discard-after-3 scheme with a Stash-shape scaled initial window
  `delta = 12 + |prevScore|/81`, progressive ×1.5 widening on fail with a +1 growth floor, never-discard
  iterations, `AspInit`/`AspMult` exposed as UCI/SPSA tunables) landed on main as an outright SPRT accept:
  penta **+20.2 [+7,+33]**, pLLR **peak +2.98 crossed the +2.94 bound at G1202** (first touch G1202–1203,
  sustained crossing G1299–1305, c8-drain to +2.88 at 1309g), 0/0 flag-outs / 1309g, real-clock 10+0.1 c8
  adjudicated (`2026-07-19-t5-aspiration.md`) — under the STRICT per-change gate (H1-only, NO
  provisional-on-cap). Isolated SPRT [-3,+3] vs the on-box T1e base `ngn_t1e.exe` `f65a3c16…` (== `801d384`).
  This is a search-behavior change, NOT node-identical: fixed-depth nodecheck MOVED (kiwipete d12
  230036→346662 +51%, mid d12 181982→149587 −18%, end d16 587557→765656 +30%), verified deterministic, so the
  baselines were re-locked to 346662/149587/765656 in the follow-up commit. Hash-back reproduced the exact
  playing binary `0a8f65b7…`. **Biggest single keep of the revamp campaign** (prior max T1b +13.1). The
  post-batch certification composite (vs `c514a2b`) will include this gain, isolable by this commit for the
  bisect-on-fail path. Batch-2 net claimed self-play now **+26.8** (T1e +6.6 + T5 +20.2), 0 provisionals.

---

## Batch 3 — OPEN (certified base: `172e9e1`, opened 2026-07-25)

Certified base `172e9e1` has engine byte-identical to the T5 keep `5b93966` (T7, T9a and T10b were all
shelved and reverted, so nothing since T5 has changed the engine). On-box certification binary:
`ngn_cert2.exe` sha256 `0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721`.

Batch 3 accumulates the next capped-but-positive PROVISIONAL keeps against this base; full H1 accepts also
land on main and are folded into this batch's composite for attribution, same as Batches 1 and 2.

| # | Change | Commit | Verdict (real-clock self-play, 10+0.1) | Run record |
|---|---|---|---|---|
| — | (none yet) | — | — | — |

- **Trigger:** certify when Batch 3 reaches 4 provisionals **OR** 2 weeks (**≤ 2026-08-08**), whichever first.
- **Certification run:** post-batch HEAD vs `ngn_cert2.exe` (`172e9e1`), real-clock `-tc 10+0.1` c8, SPRT [0,+6], LAN 9800X3D box.
- **On fail:** bisect Batch 3 commits to find the regressor; drop it, re-certify.
- **Batch net (claimed self-play):** +0.0 — empty.
