# 2026-07-26 T20 — SEE-prune depth range 4 -> 8 — PRE-REGISTERED, launches behind T3b

**STATUS: QUEUED, NOT LAUNCHED.** T3b is on the box. This manifest is written in advance per the CLAUDE.md
manifest-before-launch rule and the anti-stall rule.

Origin: the June technique census flagged that NGN's SEE-prune dies at `depth <= 4` while the class runs to
~8. Futility was extended to 8 in W1; **SEE-prune never was.** Change size: **two tokens** (`4` -> `8` on the
capture and quiet SEE-prune conditions, search.go:1715 and search.go:1729).

## A SPLIT VARIANT WAS TESTED AND FALSIFIED — run the FULL patch

Reading T20 against the lesson T18b had just delivered (a faithful-looking transplant that imports a
technique without its surrounding calibration) surfaced a real structural asymmetry:

- **capture path prunes at a FLAT `SEE < -100`** — identical threshold at depth 1 and depth 8;
- **quiet path prunes at a SCALED `SEE < -80*depth`**, and the code's own comment states the principle:
  *"Threshold scales with depth so deeper nodes prune only larger losses."*

So T20 extends a **flat** margin into deep nodes on the capture path, violating the convention the engine
states two lines below for quiets. The hypothesis was that the sound half is the quiet path, and a split
variant **T20q** (`output/t20q-quietonly.patch`, quiet path only) would capture the gain at lower risk.

**Measured, and the hypothesis is dead.** Depth at 400K fixed nodes — the metric this lane is steered by:

| position | base | T20 full | T20q | full Δ | T20q Δ |
|---|---|---|---|---|---|
| quiet-mg-QGD | 14 | **15** | **15** | +1 | +1 |
| quiet-mg-closed | 12 | **13** | **13** | +1 | +1 |
| lateMg-ph6-11 | 13 | 13 | 13 | 0 | 0 |
| najdorf-mg | 14 | **15** | **12** | **+1** | **-2** |
| rook-eg-R4P | 17 | 17 | 17 | 0 | 0 |
| pawn-eg | 32 | 32 | 32 | 0 | 0 |

**The capture-path extension is what carries najdorf; removing it costs 2 plies there** — a 3-ply swing
against the split. T20q is strictly worse and is NOT the candidate.

**The endgame-cost argument for splitting also failed on its own evidence.** Fixed-depth nodecheck showed
T20 full at **-27.8% / -18.2% / +78.4%**, and that +78.4% endgame figure was the original reason to suspect
the capture path. But it comes from **one** position, and both ebfprobe endgames (rook-eg-R4P, pawn-eg) show
**flat depth and flat nodes**. This is precisely the artifact class `project_endgame_tree_diagnostic` records
("the 2.5-5x eg tree was a mislabeled-position artifact; single-position node counts unreliable for pruning
knobs"). One position is not evidence of an endgame regression.

**What survives from the analysis:** the flat-vs-scaled capture-margin asymmetry is real and is exactly the
premise of **T15** (capture margin `-100` -> `-100*depth`, mirroring the quiet convention), already queued.
T20 and T15 are independent code paths and stay separate changes — but if T20 keeps and T15 later keeps, the
capture margin will have been both extended and re-shaped, and **that composition is worth an explicit
interaction check** rather than assuming additivity.

## Behavioral-delta gate (>= ~1%, earned from T7) — PASSES DECISIVELY

Fixed-depth nodecheck vs the HEAD lock (346662 / 149587 / 765656): **-27.8% / -18.2% / +78.4%**
(250165 / 122356 / 1365868). Baseline re-verified this session from a **fresh HEAD build** — note
`scripts/nodecheck.sh` does **not** rebuild, it measures whatever sits at `build/ngn`, so a stale binary will
silently report "no change" for any candidate.

**Second gate question, earned from T18b: what KIND of information moved the tree?** SEE pruning skips moves
on `staticExchangeEvaluation`, an **exact material-swap computation** — proven information, not a cached
guess, and nothing is persisted to the TT. T20 passes the soundness question that T18b failed. The residual
speculation is the *threshold*, not the computation: SEE cannot see promotion, king activity, or attacking
compensation, so a deep prune can discard a real sacrifice.

Mechanism fires (kiwipete d12 `seeq`): base 10647 -> **T20q 19569 (+84%)**.

> **Counter caveat — `seeq` is CONFOUNDED for the full patch, same trap as T12's `nmtry`.** T20 full reads
> `seeq 10067`, *below* base, despite strictly widening the quiet prune's depth range — because the
> capture-path extension shrinks the tree ~28%, leaving fewer quiet nodes to prune at all. An absolute counter
> cannot witness a change that alters tree composition. The +84% on the isolated quiet variant is the honest
> witness that the mechanism fires.

```yaml
id: 2026-07-26-t20-see-prune-range
date: 2026-07-26
change_class: search heuristic (pruning depth range). Requires a completed predeclared paired game test.
hypothesis: >
  NGN's SEE-prune dies at depth<=4 while the class runs to ~8, and W1 already extended futility to 8 without
  extending SEE. Widening both SEE-prune paths to depth 8 buys depth at equal nodes (+1 ply on 3 of 4
  middlegames) on exact material information, with no persisted speculation.
patch: output/t20-seerange.patch (22 lines, engine/search.go only; two tokens of real change)
rejected_variant: output/t20q-quietonly.patch (quiet path only) — FALSIFIED, -2 plies najdorf
base: ngn_t5.exe sha256 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721
  CONDITIONAL — correct only if T3b SHELVES. If T3b is KEPT, T20 must be rebuilt and re-gated against the
  T3b tree; do NOT run against a superseded base and pool.
harness: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-validated, untouched)
command: sprt.exe -new .\ngn_t20.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
tc: 10+0.1 (seconds)
concurrency: 8
openings: canonical sprt_openings (5000 lines) sha256 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: STANDARD-ON (M5-validated flags)
maxgames: 8000
expected_duration: ~8h30m at ~948 g/hr if it drains to cap; decisive runs on this rig have stopped at 1300-5300
```

## Predeclared decision rule (standard per-change T-item gate)

- pLLR **>= +2.94** -> **KEEP** as a Batch-3 keep; re-lock nodecheck to the T20 numbers; re-run ebfprobe +
  the fmc/ttlist table per the re-measure cadence.
- pLLR **<= -2.94** -> **SHELVE** and revert.
- Capped with point est **>= +1 and pLLR > 0** -> **PROVISIONAL** keep into Batch 3.
- Capped **nonpositive** -> SHELVE.

**REOPEN CONDITION (predeclared).** A negative T20 does not close the SEE-prune-range lane by itself, because
depth 8 is a single guess at the cap. The re-derived variant is **depth <= 6** (half the extension), and it
should be tried only if the loss is small; a large negative implicates the flat capture margin and should be
routed to **T15 first** (fix the margin shape at the current depth cap), after which the range extension can
be re-tried on top of the corrected shape.

Verdict is taken from the harness's own `H0/H1 ACCEPTED` line plus a full-envelope scan of the printed
samples — **never the printed final pLLR alone** (c8-drain artifact, observed 3x, symmetric).

## Honest expectation

**+2 to +6 class.** In its favour: +1 ply on 3 of 4 middlegames is the strongest depth result of any candidate
gated this campaign, it runs on exact material information, it persists nothing, and it completes a transplant
the engine already half-adopted (futility to 8 in W1). Discounts: **history pruning is still capped at
`depth <= 3`** — the same unfinished extension, deliberately kept as a follow-up rather than bundled; SEE's
blindness to promotion and attacking compensation is a real cost at depth 8; and the flat capture margin means
this extension lands in a regime the engine's own stated convention says should scale.
