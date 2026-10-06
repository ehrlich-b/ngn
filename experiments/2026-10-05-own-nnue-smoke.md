# Own NNUE end-to-end pilot smoke — 2026-10-05

Bounded WSL run of NGN's own datagen → PyTorch trainer → NGNN1 inference →
NGN SPRT harness. Source: `indep/main`, engine merge `7a61203e`, tooling
`7b7645a1`; isolated checkout `/home/ehrli/wt/int`. All engine execution and
training ran on WSL; the Mac only edited and committed files. No external
engine, chess library, trainer, data, or weights were used. The existing venv
provided its interpreter, NumPy, and PyTorch only.

## Build and checks

Go 1.25.5, `GOAMD64=v3`, `GOMAXPROCS=4`, build/test parallelism 2. Passed
`go test -p 2 -short ./...`, `go test -p 2 -short -race ./engine/...`, then
the changed SPRT/UCI packages, UCI race tests, and all 16 trainer tests.
The initial full checks and engine build took 54 seconds.

Engine SHA256: `2945df9476378b71746c3a2fff2f2b542cddef997c2e97fa9a0115bf678f1b94`.
SPRT SHA256: `73624b14bb240db045c1d25d02b175853a3c4c3d4de312cb855c2783cf0339de`.
The engine binary remained byte-identical after adding tooling.

## Training and export

Immutable source: `/home/ehrli/ngn-data/pilot/run-20261005T224036Z-woQn6X/*.ngnp`
(14 shards, **1,328,267 records**). Each shard's final `floor(N/20)` records
were copied to validation; preceding records were copied to training. This
gives **1,261,860 train / 66,407 validation**, 4.99952% held out. Default flag
filter `0x0f` discarded no records. Source SHA256s were unchanged afterward.
Record ranges, source hashes, and copies are recorded in `split.json` and
`split/{train,val}/` under the output directory below. A range boundary can
divide a self-play game; this is a record-range holdout, not a game holdout.

H=256, batch 16,384, AdamW LR 0.001, cosine decay to 0.1×, weight decay 0.01,
SCReLU, sigmoid-MSE with K=400 and score/result blend 0.75/0.25; seed 20261005.
RTX 5080, PyTorch 2.7.0+cu128. CUDA training used BF16 matrix operations and FP32 parameters/loss;
validation used FP32. Four CPU threads. All 40 full epochs completed in
**12 seconds**, within the 880-second foreground timeout. Training loss is
the mean over updates; validation is evaluated after each epoch.

| Epoch | Train loss | Val loss | Epoch | Train loss | Val loss |
|---:|---:|---:|---:|---:|---:|
| 1 | .03398944 | .01597730 | 21 | .00678817 | .00956299 |
| 2 | .01104934 | .01066504 | 22 | .00668677 | .00971227 |
| 3 | .00954720 | .00994978 | 23 | .00661616 | .00966031 |
| 4 | .00932675 | .00991816 | 24 | .00655236 | .00962758 |
| 5 | .00907782 | .00997583 | 25 | .00646459 | .00970032 |
| 6 | .00893634 | .00974889 | 26 | .00640687 | .00963486 |
| 7 | .00870555 | .00968380 | 27 | .00634764 | .00955783 |
| 8 | .00854511 | .00958551 | 28 | .00629664 | .00962396 |
| 9 | .00833391 | .00948610 | 29 | .00625559 | .00960962 |
| 10 | .00817673 | .00952952 | 30 | .00620694 | .00969867 |
| 11 | .00803885 | .00989254 | 31 | .00616471 | .00966646 |
| 12 | .00792510 | .00950735 | 32 | .00611999 | .00965314 |
| 13 | .00773386 | .00950674 | 33 | .00608803 | .00967905 |
| 14 | .00754878 | .00979756 | 34 | .00606192 | .00968037 |
| 15 | .00745659 | .00960297 | 35 | .00602641 | .00970026 |
| **16** | **.00731638** | **.00945503** | 36 | .00601011 | .00963676 |
| 17 | .00718907 | .00963942 | 37 | .00598785 | .00968124 |
| 18 | .00708682 | .00966456 | 38 | .00596269 | .00968445 |
| 19 | .00698296 | .00966842 | 39 | .00595435 | .00968740 |
| 20 | .00687864 | .00964905 | 40 | .00593974 | .00966868 |

Selection rule declared before training: lowest validation loss. **Epoch 16**
won with **0.009455028324091869**; later training reduced train loss while
validation plateaued. Exported `pilot-h256.nnue` SHA256:
`3c6973541ea70ccbbdc82e1eb282e7f967c300be99f7e9ed513a49da286d77e5`.

`trainer.refeval.Network` and the engine's UCI `eval` diagnostic agreed
exactly on **20/20** held-out positions, chosen at evenly spaced accepted
validation indices from 0 through 66,406. FENs, reference/engine cp, and
indices are in `diagnostics.json`; raw UCI output is in `eval-uci.log`.

Scores below are side-to-move cp, before correction history. Full FENs are
included in `diagnostics.json`.

| Position | NNUE | HCE |
|---|---:|---:|
| Startpos | -3 | +10 |
| White queen up, `7k/8/8/8/4Q3/8/8/K7 w - - 0 1` | +871 | +1016 |
| Same board, Black to move | -846 | -996 |
| Kings only, a1/h8, White to move | -22 | +10 |
| King/pawn ending, White Ke4/Pe5, Black Ke7 | +189 | +178 |
| Open game after 1.e4 e5 2.Nf3 Nc6 | -24 | +10 |
| Starting board without White's queen | -994 | -973 |

## Match

Same binary on both sides; candidate `EvalFile=pilot-h256.nnue,UseNNUE=true`,
baseline `UseNNUE=false`. Both: `OwnBook=false,Threads=1,Hash=32`, default
100 ms Move Overhead, wrapper `GOMAXPROCS=1`. Identical foreground `tee`
wrappers save each engine's UCI stdout for NPS/integrity checks.

NGN's `cmd/sprt` at **10+0.1**, concurrency **4**, maximum **200 games** or
**35 minutes**; `mingames=201` disables SPRT stopping for this fixed smoke.
No score adjudication; the harness's default 200-ply cap remains. Per-game
watchdog 120 seconds; outer foreground timeout 2100 seconds.

The suggested existing opening file was absent. NGN generated 100 distinct
eight-ply uniform-random legal opening lines with seed **20261005**, using
`sprt -genbook 100 -genplies 8 -balancedepth 1 -balancecp 30000` (permissive
balance ceiling). Each line is played with reversed colors. Opening SHA256:
`cc2e8f06c9a8a305941bfcefa64372e16f7c51e009e79f52870896984b887708`.

Completed **200 games / 100 pairs** in **25m33s** (foreground exit 0):
**85 W / 41 D / 74 L**, score **52.75%**. Pilot minus HCE: **+19.1 Elo**;
paired 95% CI **[-16, +55]**, Wilson game-score 95% CI **[-29, +67]**.
Pentanomial `[LL, LD, {LW/DD}, WD, WW] = [7, 16, 47, 19, 11]`.
The result is inconclusive about a strength gain; both intervals include zero.

Integrity: **0 observed crashes, 0 illegal moves, 0 watchdog voids, 0 UCI
configuration errors**. Time losses: **NNUE 0 / HCE 1**, retained in the score.
End reasons: 158 checkmates, 20 draw-rule, 21 200-ply-cap draws, 1 time-forfeit.
All eight engine subprocess transcripts were present; no run engine process
remained afterward.

Match transcript NPS, excluding the first warmup search in each subprocess
and searches with zero reported time/NPS; one sample per move's last emitted
depth-info line:

| Side | Timed searches | Median NPS | Sum(nodes)/sum(time) NPS |
|---|---:|---:|---:|
| NNUE | 12,852 | 828,712 | 812,823 |
| HCE | 12,748 | 1,226,682 | 1,203,099 |

NNUE's median throughput was about 67.6% of HCE's. These are reported search
rates from the concurrent timed match, not isolated evaluation benchmarks.

Outputs and receipts: `/home/ehrli/ngn-data/nets/pilot-h256/`, **198 MiB**.
Large data/checkpoints/binaries stay on WSL. Active check/training/match work
was about 27 minutes, below the 60-minute bound.
`match.log` SHA256: `732dbd835a258b0f8363023cb061e3cfc248b6b817a2049a07461d8460ddfcee`.
`match-stats.json` SHA256: `b3d6e04f454299e698344649c68fe175bbdbec5faf1a70437f2788634d96b9be`.

## Merged AVX2 validation and repeat

Tested source `b1c404a3` on `indep/main`, including `own/avx2` and
`own/clean`. Mac edits only; all builds, checks, searches and games used WSL
`/home/ehrli/wt/int`, Go 1.25.5 and `GOAMD64=v3`. No external sources,
tools, openings, data or weights were introduced. Compact raw evidence:
[verification receipt](2026-10-05-int-avx2-verification.json).

Datagen now accepts `-net PATH` through `SelectNGNN1Evaluator`, the same
search-worker path used by UCI `UseNNUE`. It hashes the exact validated input
stream and writes `net_sha256` in each shard's JSON sidecar; settings also
record the path. Missing, truncated, corrupt or trailing-data networks fail
before creating a shard, with no HCE fallback. Empty `-net` keeps HCE and
omits the network digest. The process launcher forwards `-net`; search remains
one thread per process. Tests cover engine-selection/search equivalence for
both colors, sidecar provenance, invalid inputs and HCE provenance.

WSL checks, with `GOMAXPROCS=2` and Go package parallelism at most two:
`go vet -p 2 ./...`, `go test -p 2 -short -count=1 ./...`, and
`go test -p 2 -short -race -count=1 ./engine/... ./internal/...` all passed.
The six-position perft suite, exhaustive magic checks, all 128 NGNN1 kernel
widths and 64-FEN trainer parity fixture passed in the ordinary AVX2 build;
NGNN1 tests also passed with `-tags purego`. All **16 trainer tests** passed.
No merge repair was needed. HCE depth-four dumps match `origin/own/clean`
(`d33f6639`) byte-for-byte, including nodes **1184 / 4620 / 274 / 553 / 387 /
332** on the verification document's six positions.

Datagen CLI smoke: one 12-ply game, 512 nodes/move, no random opening, Hash=1;
**10 records**, closed complete, correct pilot network digest. Invalid-net
CLI smoke rejected before reserving either output file. Datagen was built
with `-ldflags "-X main.engineCommit=$(git rev-parse HEAD)"` to supply the
required provenance; this worktree's ordinary Go build omitted VCS metadata.

Pinned throughput: CPU **14**, `GOMAXPROCS=1`, Threads=1, Hash=16 MiB,
OwnBook=false, Move Overhead=0, fresh process/TT for each 10-second search.
FENs are `scripts/netinfer-probe.py --middlegames`, also stored in the receipt.
Backend order alternated between positions. Wall NPS uses final debug nodes,
including the unfinished iteration, divided by measured search time
(10.0003–10.0004 s).

| Position | HCE wall NPS | Pilot NNUE AVX2 wall NPS | NNUE/HCE |
|---|---:|---:|---:|
| Kiwipete | 1,255,426 | 1,615,753 | 1.29x |
| Italian | 1,167,206 | 1,832,495 | 1.57x |
| Queen's Gambit | 1,183,967 | 1,770,373 | 1.50x |
| Arithmetic mean | 1,202,200 | 1,739,540 | 1.45x |

These are single samples on different evaluator search trees on a shared
host; affinity does not reserve the CPU. Mean NNUE throughput was 44.7%
higher than HCE in this panel.

Repeated match: same AVX2 binary on both sides, pilot NNUE versus HCE, exact
original opening file/hash above, **10+0.1**, concurrency **4**, Threads=1,
Hash=32, OwnBook=false, default 100 ms Move Overhead. Identical foreground
`tee` wrappers set `GOMAXPROCS=1`. Caps: 200 games / **30 minutes**;
`mingames=201`, per-game watchdog 120 seconds, no score adjudication and the
default 200-ply cap. The receipt records the full command.

Completed **200 games / 100 pairs** in **25m33s**, exit 0:
**104 W / 47 D / 49 L**, score **63.75%**, **+98.1 Elo** versus HCE;
paired 95% CI **[+63, +135]**, Wilson game-score CI **[+48, +148]**.
Pentanomial `[LL, LD, {LW/DD}, WD, WW] = [2, 9, 41, 28, 20]`.
Both intervals exclude zero in this comparison; this does not establish an
absolute rating. SPRT stopping was disabled, so its generic inconclusive
verdict is not a fixed-match confidence assessment. This tree also includes
`own/clean`, so the difference from the earlier scalar smoke cannot isolate
AVX2's effect on Elo.

Integrity: **0 observed crashes, illegal-move reports, watchdog voids, UCI
configuration errors or time losses**. End reasons: 153 checkmates, 22
draw-rule, 25 ply-cap draws. All eight engine transcripts were present; no
run engine/wrapper/harness process remained. Excluding the first search in
each process and zero-time/NPS samples, transcript median NPS was
**1,780,588 NNUE / 1,468,925 HCE**; aggregate nodes/time was
**1,757,408 / 1,451,166** across **12,603 / 12,563** timed searches.

Engine SHA256: `27a7ef1bbfcf67143e8962af95ac27c91879fcee522a5b04d66142b0e3b15768`.
Match log SHA256: `8bee21af4b0a31e74da68d01174d300cececf25d8a17e488acecb1d6a06a40e5`.
Network SHA256 remained unchanged. Logs, binaries and the tiny CLI shard stay
under `/home/ehrli/ngn-data/int-avx2-20261005/`, about **58 MiB** total.
Checks, throughput and match occupied about **28 minutes** of foreground box
work; no detached jobs or unrelated process kills.
