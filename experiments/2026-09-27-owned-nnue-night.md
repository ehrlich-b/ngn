# Owned NNUE night session — 2026-09-27

Live record for the overnight WSL session. Artifacts live under
`/home/ehrli/nnue-owned-k4-20260920/night-20260927/` on WSL; small receipts are
copied here with the `2026-09-27-` prefix. All matches use
[`scripts/quickmatch.py`](../scripts/quickmatch.py): hash-checked inputs, the
frozen fastchess (`e61220f6…`) argv shape (sequential frozen openings,
`-repeat` pairs, per-game affinity, `-strict`), the frozen independent chess
auditor (`4c4d7374…`) with Stockfish 18 (`6b087694…`), and a report paired by
round (percentile bootstrap over pair scores, 20,000 resamples). Engine
options are sent in the audited order Threads=1, OwnBook=false, EvalFile,
EvalBackend, Hash=128, Move Overhead=100; `GOMAXPROCS=1`. Host: 9800X3D WSL
(Go 1.25.5, GOAMD64=v3, CGO_ENABLED=0), shared with GPU training and the idle
hopper. Relative results only; no absolute rating.

## Recovered state

The Sep 24 A1 result (archive-labeled K4, **promoted**, +57.9 vs Rodent V1.1)
existed only as uncommitted files in a `/private/tmp` worktree that macOS was
already purging. Files were rescued, verified byte-identical to the sources
that built/trained A1 on WSL, and committed (`1835960`).

## M1 — A1-e10 vs Rodent V1.2 (old binary)

Same binary as the A1 gate (`1f250284…`, int32 K4 kernels), openings
4241–4340, seed 2026092301, 10+0.1, c4 on CPUs 8/10/12/14, 200 games.
Predeclared rule: owned net replaces V1.2 as working reference only if the 95%
lower bound is above zero; V1.2 stays if the upper bound is below zero;
otherwise parity. [Spec](2026-09-27-m1-a1-vs-v12-spec.json),
[report](2026-09-27-m1-a1-vs-v12-report.json).

- 59/103/38 (white/draw/black wins), penta `[2, 29, 50, 18, 1]`, 32,853 legal
  plies, audit PASS, zero book-signature plies.
- A1-e10 score 0.4675 [0.430, 0.505], **−22.6 Elo [−49.0, +3.5]**: parity.
- In-game NPS: K4 1.017M mean vs V1.2 1.193M (K4 at 85%) — the old int32 K4
  kernels, which S1 then replaced.

## S1 — exact int16 kernels (accepted, pure speed)

See [the S1 record](2026-09-27-s1-k4-int16-kernels.md): identical search on 88
fixtures, 1.162x faster. Research tags for K8/W1024 layouts:
[record](2026-09-27-k4-research-layout-tags.md).

## M1b — A1-e10 vs Rodent V1.2 on the S1 binary

Identical to M1 except both roles run the S1-era binary (`5d989a7f…`, built
from the tree committed as `d7501b6`, default tags); the V1.2 code path is
unchanged, so pairing with M1 by round isolates the K4 speedup.
[Spec](2026-09-27-m1b-a1-vs-v12-spec.json),
[paired report](2026-09-27-m1b-a1-vs-v12-paired-report.json).

- Audit PASS, 200 games; penta `[2, 21, 47, 26, 4]`.
- A1-e10 score 0.5225 [0.4825, 0.5625], **+15.6 Elo [−12.2, +43.7]**:
  parity under the M1 rule, point estimate now favoring the owned net.
- Paired with M1: **+0.055 score [+0.0025, +0.1075]** from the S1 speedup
  (supporting evidence only; S1 was accepted on identity plus timing).
- In-game NPS: K4 1.146M mean vs V1.2 1.126M (K4 now at 102%).

## K8 — eight king buckets, same corpus and schedule

Trainer built with `NGN_K8` from the `d7501b6` harness source (binary
`33ba487c…`), mode `archive-a1-e10` (1,638,400 updates, cosine 0.001→0.00001,
seed 26092001) on the unchanged A1 corpus (`270f65bd…`, 2,374,290,033 records);
~6.2M positions/s (A1: 8.8M), about 58 minutes. Selector (`ngnk4bridge`
built with `-tags ngnk8`) picked update **1,540,096**; integer validation MSE
on the frozen SF18 holdout ≈0.00478 (final 0.004783) versus A1's selected
0.004850. Converted model `765251377cb5…` (9,463,568 bytes).
[Convert](2026-09-27-k8-e10-convert.json),
[parity](2026-09-27-k8-e10-parity.json) (first 3 of 12 records kept): the
Bullet float probe and the Go reference agree to 6.7e-7 z on all 12 parity
FENs (so the Go K8 table matches Bullet's mirrored layout), quantization
delta mean 5.2 cp / max 11.7 cp, **PASS**.
