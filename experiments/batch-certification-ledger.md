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
- **On fail:** bisect batch-1 commits (`7394668`, …) to find the regressor; drop it, re-certify.
