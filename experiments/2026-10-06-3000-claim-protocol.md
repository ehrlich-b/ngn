# Predeclared NGN >3000 claim protocol

This protocol fixes the analysis before new claim games. It is a local
historical-label calibration, conditional on the anchors; it does not establish
an official CCRL rating. The [accepted September 5 calibration](2026-09-05-2800-result.md)
was 2884.1429896655936 [2834.197941381757, 2934.08803794943] at 120+1,
concurrency 8. Its games, older NNUE experiments and October 5 orientation are
excluded from this sample. No claim run or multi-hour job is authorized here.

## Candidate and frozen inputs

Use the independently produced NGN engine on `indep/main`, built on WSL with
Go 1.25.5, GOAMD64=v3. All engine, trainer and match-runner implementation and
candidate weights must be NGN-produced; general purpose stdlib, NumPy and PyTorch
are allowed. Third-party-evaluated positions are allowed as training data
(user, 2026-10-06). Opponent executables serve only
as rating anchors. Do not read, port or derive NGN implementation from them.

Before the screening games, save a dated immutable receipt giving the exact
source commit and clean tree, Go version/build argv, engine and runner SHA-256,
net format/H/bucket count and SHA-256 (or explicitly HCE), generation manifest
digest and all data/training/export/parity provenance. Use that **same candidate**
for the expensive run. A changed binary/net requires a new prospective protocol
and independent sample. The tiny pipeline validation is a tooling check, not
the nominated >3000 candidate. Identity fields cannot remain unresolved when
games launch; the receipt must be recorded before results exist.

NGN settings: `Threads=1`, `Hash=64`, `OwnBook=false`, `Ponder=false` if
advertised; load `EvalFile=<frozen NGN net>` then `UseNNUE=true`, or explicit
`UseNNUE=false` for HCE. Freeze Move Overhead and every additional UCI option.
Save handshakes, readiness and static evaluator probes, effective thread count
and exact launch paths; the default version suffix alone does not identify the
active evaluator. Keep the same settings at both time controls. Opponents use
one search thread and Hash 64 where supported, with their book/ponder disabled;
record supported options and receipt of selected settings rather than sending
unadvertised options. Check host load and a bounded A/A clock control beforehand.

## Fixed anchor pool and openings

Labels are frozen historical **CCRL Blitz 1CPU** values, not current estimates.
Do not replace them with 40/15 or multicore labels.

| Anchor | Fixed label | Located binary and evidence |
|---|---:|---|
| Counter 3.8 | 2994 | `C:\Users\ehrli\ngn\repin\anchors\counter_38.exe`; [ratings record](../opponents/ratings.json) and [October 5 inventory/probe](2026-10-05-anchor-orientation.md); UCI says Counter dev, so the historical binary digest is essential |
| Maelstrom 3.3.0 | 3317 | `/home/ehrli/nnue-owned-maelstrom-20260930-v2/maelstrom-3.3.0`; [1CPU catalog](2026-09-30-owned-top3-go-goal.json), [version receipt](2026-09-30-owned-maelstrom-result.json), [located inventory](2026-10-05-anchor-orientation.md) |
| Counter 5.5 | 3359 | `/home/ehrli/repos/ngn-next/output/go-opponent-preflight-20260906/counter-5.5-linux-amd64`; [September label reassessment](2026-09-12-rating-reassessment.md) and [inventory](2026-10-05-anchor-orientation.md); retain 3359 rather than later 3360 |

The recorded broad [box inventory](2026-10-05-anchor-orientation.md) found
**no additional executable with a verified 2900–3200 Blitz label**. Counter 3.8
is the sole available anchor in that interval. Counter 4.1's 3157 reference is
40/15 and its executable was absent; Blunder 8.5.5 and GoChess lack verified
labels. Do not silently add one. A later discovery needs a prospectively revised
pool and equal sample size before any claim games, with version/label records.

Verify hashes/version identity again before play. Recorded Counter 3.8 digest:
`d260174182ea10c5b902a8d25e217166f5b5b54dc4fa64c220b498acce6dec11`;
Counter 5.5:
`6c48fb52934d49d3774633e32f0b4fb4796b1e2c63925f24167ef0f0c0d761c8`.
Pin Maelstrom's executable digest and any adjacent network dependency in the
pre-run receipt. Save before/after hashes for every engine, net and runner.

Use the NGN opening file preserved at
`/home/ehrli/repos/ngn/output/corrected-pin-20260905/results-r0905matepin/inputs/sprt_openings.txt`,
SHA-256 `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
The [inventory](2026-10-05-anchor-orientation.md) verifies this owned copy and
notes the shorter `output/sprt_openings.txt` path is absent. Reading these NGN
opening bytes is permitted; old third-party sources/toolchains are not inputs.
Freeze the full file and a first-80-usable-lines slice with its own digest.
Check each opening legally before launch. Use indices 0–79 once in each color
against each anchor, identically across anchors, with **160 games per anchor**
(480 total, 240 complete pairs). Do not drop or replace opening indices after
results are observed. Prior use does not permit importing any prior game.

## Expensive confirmation and decision

Play real tournament clocks **120+1**, concurrency **8 games total** on the
shared 9800X3D Windows/WSL box, with the same candidate and options throughout.
Record host/OS, scheduling and CPU load; the earlier accepted calibration was
native Windows, so WSL/OS and anchor-label transfer remain unmodeled limitations.
Obtain separate user authorization for this multi-hour workload. Do not run it
while unrelated workloads make the clock control fail.

Use the NGN-owned match tooling with score stopping disabled (`mingames >
maxgames`, when using cmd/sprt), no score resign/draw adjudication, and a frozen
400-ply cap. Natural legal terminals/rule draws have priority. Save all moves,
opening histories, clocks, game results/reasons and pair identifiers. The current
genloop gate is 10+0.1 and records summary logs; it cannot supply this claim's
full replay evidence by itself. Admit a runner only after it demonstrably saves
the required complete histories and terminal receipts.

For anchor i, N_i = W_i + D_i + L_i = 160. Copy the original primary formula
from [September 5](2026-09-05-2800-result.md) without substituting a new estimator:

```text
p_i = (W_i + 0.5*D_i + 0.5)/(N_i + 1)
R_i = label_i + 400*log10(p_i/(1-p_i))
v_i = (400/ln(10))^2/(N_i*p_i*(1-p_i))
weight_i = 1/v_i
R_pool = sum(weight_i*R_i)/sum(weight_i)
SE_pool = 1/sqrt(sum(weight_i))
CI95 = [R_pool - 1.96*SE_pool, R_pool + 1.96*SE_pool]
```

Claim **local historical-label strength >3000** only if all three anchors have
exactly 160 admitted games and 80 intact color pairs, every integrity audit
passes, the **primary pooled lower 95% bound is strictly >3000**, and raw
per-anchor scores `(W_i+D_i/2)/N_i` lie **strictly above and below 50%** across
the pool. Report every per-anchor W/D/L, score, performance, variance, weight,
pooled estimate/interval and all exclusions (the permitted number is zero).
Equality at 3000 or unbracketed scores fails the claim. No sample extension,
optional stopping, post-hoc anchor replacement, games pooled from earlier runs,
or selective replay of failed games. An operationally invalid sample is invalid;
any new attempt needs its own prospectively declared complete sample.

Report a supplemental whole-opening cluster bootstrap: resample the 80 indices
jointly across all anchors, retaining both colors, 10,000 draws with seed 3000,
and recompute the same pool. It does not replace the primary rule. Both intervals
are conditional; label uncertainty, OS/TC transfer, shared openings, correlated
Counter versions and non-transitivity prevent an official-rating interpretation.

## Legal and terminal admission

Replay every opening and played ply with NGN-produced audit code, retaining the
full prior history for repetition. Verify side to move, legal move membership,
clock arithmetic and every checkmate/stalemate, repetition, fifty-move and
insufficient-material terminal. Verify the 400-ply cap is applied only after
testing true terminals; disclose cap draws. Reconcile game/pair coverage, color,
results, reasons, raw logs and W/D/L independently of the collector. Zero illegal
moves/openings, missing moves, no-move forfeits, watchdog voids, clock losses,
duplicate games, unexpected engine errors or mismatched terminal labels are
required. Before admission, owned fixtures must show that the audit detects a
deliberately mislabeled checkmate-as-draw and handles repetition/cap boundaries.
Freeze all audit code and receipts; do not reuse the earlier third-party oracle
implementation or toolchain. Hashes must remain unchanged after the games.

## Cheaper screen

First play **10+0.1**, concurrency **8 total**, **80 games per anchor** (240
total), opening indices 0–39 with both colors once each. Freeze identities and
the slice before launch; use the same options, terminal rules, integrity checks
and primary formula with N_i=80. No SPRT stop, extensions, or older-game pooling.
This screen needs separate authorization beyond the tiny development test.

The expensive run is worthwhile only if the screen completes cleanly, scores
bracket 50%, and its nominal pooled lower 95% bound exceeds 3000. A failed screen
returns to development; it is never combined with confirmation. Passing the
screen admits only the already-frozen candidate, subject to user authorization,
and supplies no >3000 claim on its own. The [October 5 H256 pilot orientation](2026-10-05-anchor-orientation.md)
was 2557 [2509,2605] at 10+0.1 and supplies no reason to run this expensive
protocol for that pilot identity.
