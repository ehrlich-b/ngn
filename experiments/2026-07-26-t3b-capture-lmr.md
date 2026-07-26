# 2026-07-26 T3b — capture LMR (non-winning captures only) — PRE-REGISTERED, not yet launched

**STATUS: QUEUED, NOT LAUNCHED.** The box is running T18b until it resolves. This manifest is written in
advance per the CLAUDE.md rule (manifest before launch) and the anti-stall rule, so T3b fires the moment the
box frees.

Origin: fell out of the 2026-07-26 plateau diagnosis (`experiments/2026-07-26-plateau-diagnosis.md`), not from
the technique-gap prior list. Change size: **one condition + one reduction branch** (70-line patch, mostly
re-indentation of the existing quiet-only adjustments).

## Premise VERIFIED from code

`search.go:1921` gates LMR on `!move.IsCapture()`. **NGN reduces ZERO captures, ever** — captures are always
searched at full depth once they pass the earlier prunes. That is a genuine divergence from the class, which
reduces losing/neutral captures.

T3b changes the gate to `(!lmrIsCapture || !seeNonLosingByMVV(move))` and gives captures that do enter LMR
**one ply less** reduction than quiets, bypassing the quiet-history and killer adjustments (both keyed on
quiet tables, meaningless for captures).

## Explicitly NOT inheriting T3a's gate

T3a (-7.9) reduced **quiets** harder at ~84% FMC and over-reduced mis-ordered moves. Captures are a
separately and better-ordered population (MVV/LVA + SEE + capture history). The measured re-search churn
below is the direct check on whether that reasoning holds.

## Behavioral-delta gate (>= ~1%, earned from T7) — PASSES DECISIVELY

Nodecheck vs the HEAD lock (346662 / 149587 / 765656): **-29.6% / +35.4% / -1.4%**. Mixed direction, large
magnitude — nothing like T7's 0.001% near-inert null. Absolute values get re-locked from a fresh nodecheck
run at keep time, not copied from here.

Depth-at-fixed-nodes (the metric this lane is steered by, per the b_all correction):

| Probe | kiwipete depth @400K | QGD depth @400K |
|---|---|---|
| base | 12 | 13 |
| T3b | **13** | 13 |

Ordering: kiwipete FMC 88.5 -> **89.9%**. LMR re-search churn **unchanged at 0.2%**.

## HONEST DEFECT IN THIS PATCH — recorded before the run, not after

`seeNonLosingByMVV` (search.go:212) is a **cheap MVV-only sufficient condition**, not SEE:

```go
return !move.IsEnPassant() && move.PromoType() == NoType &&
    move.CapturedPiece().seeWeight() >= move.MovingPiece().seeWeight()
```

It proves non-losing *from material values alone*. So `!seeNonLosingByMVV(move)` is **strictly broader than
`SEE < 0`** and the reduced set therefore includes:

1. genuinely losing captures — the intended target;
2. **captures of a lower-valued but UNDEFENDED piece** — e.g. RxN on a hanging knight. This is one of the most
   common ways to win material outright, and T3b reduces it. This is the patch's real risk, and the honest
   mechanism by which it could lose Elo;
3. en passant (PxP, materially neutral) — vanishingly rare, not worth churning the patch for.

Capture-promotions are NOT affected: the LMR condition's separate `move.PromoType() == NoType` clause already
excludes them.

**The measurement that bounds concern (2):** if T3b were badly over-reducing winning captures, the LMR
re-search rate would spike, because those reductions would fail high and get re-searched. It did **not** —
churn is flat at 0.2%. That is evidence against the failure mode, not proof, so:

**REOPEN CONDITION (predeclared).** If T3b lands negative, the first re-derived variant is **T3c: gate the
capture-LMR entry on exact `staticExchangeEvaluation(pos, move) < 0`** instead of the MVV proxy, accepting the
full SEE swap cost at late captures. A negative T3b does NOT close the capture-LMR lane — it would only
falsify the cheap-proxy form of it.

```yaml
id: 2026-07-26-t3b-capture-lmr
date: 2026-07-26
change_class: search heuristic (LMR eligibility + reduction shape). Requires a completed predeclared paired game test.
hypothesis: >
  NGN searches every capture at full depth while the class reduces non-winning ones. Reducing captures that
  are not provably non-losing, at one ply less than quiets, buys depth at equal nodes (+1 ply kiwipete) without
  costing tactical accuracy, because captures are well ordered and any over-reduction is caught by the
  fail-high re-search (churn measured flat at 0.2%).
patch: output/t3b.patch (70 lines, engine/search.go only)
base: ngn_t5.exe sha256 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721
  CONDITIONAL — this is correct only if T18b SHELVES. If T18b is KEPT, T3b must be rebuilt and gated against
  the T18b tree instead; do NOT run against a superseded base and pool.
harness: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-validated, untouched)
command: sprt.exe -new .\ngn_t3b.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
tc: 10+0.1 (seconds)
concurrency: 8
openings: canonical sprt_openings (5000 lines) sha256 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: STANDARD-ON (M5-validated flags)
maxgames: 8000
expected_duration: ~8h30m at the measured 934-944 g/hr if it drains to cap; decisive runs on this rig have stopped at 4000-4500
```

## Predeclared decision rule (standard per-change T-item gate)

- pLLR **>= +2.94** -> **KEEP** as a Batch-3 keep; re-lock nodecheck to the T3b numbers; re-run ebfprobe +
  the fmc/ttlist table per the re-measure cadence (structural keep).
- pLLR **<= -2.94** -> **SHELVE** and revert; open T3c (exact-SEE gate) per the reopen condition above.
- Capped with point est **>= +1 and pLLR > 0** -> **PROVISIONAL** keep into Batch 3.
- Capped **nonpositive** -> SHELVE.

Verdict is taken from the harness's own `H0/H1 ACCEPTED` line plus a full-envelope scan of the printed
samples — **never the printed final pLLR alone** (c8-drain artifact, observed 3x, symmetric in both
directions).

## Honest expectation

**+2 to +6 class.** The depth gain is real and measured, and the lane is virgin (zero captures reduced today).
Discounts: the MVV-proxy defect above is a live risk; and the diagnosis put T3b in the *ordering/efficiency*
family only loosely — it raises FMC, but it does not raise TT-move availability (`ttlist`), which is the
specific deficit Finding 2 identified. T18b, T11a and T19 attack `ttlist` directly; T3b does not.
