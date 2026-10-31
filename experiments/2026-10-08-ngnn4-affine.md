# NGNN4 affine tiling — prospective session 16 speed candidate

Topic `night/ngnn4-affine16`, baseline **9b6a3d6** on the CPU/CUDA-tested
hidden-layer topic. The accepted runtime remains **5b882e2 / be34532c**;
this experiment never promotes NGNN4 or changes a real net.

The original NGNN4 affine kernel rereads inputs for each eight-output
block. Tile sixteen outputs and eight inputs per iteration, share each
activation broadcast across two output vectors, and keep independent
four-input partial sums before exact recombination. No packed weight
layout, loader, trainer, integer rounding or evaluation changes. Every
partial sum is bounded by the existing complete-dot bound, 66,846,720;
unsigned/signed int16 adjacent products remain within 32,640.
All assembly is written from the NGN integer specification.

Predeclared rule before launch: guarded AVX2/Go/scalar differentials across
all supported H/L widths, new cancelling/reinforcing input-phase controls,
existing 320 H512/L16 integer cases, engine/full short/race and vet.
Then **48 exact search identities**: twelve each NGNN4 H512/L16 depth12,
accepted NGNN3 and embedded NGNN2 depth14, HCE depth11. Freeze source patch,
binaries, PGO and the existing owned random H512/L16 net before searches.
No new trainer or fixture generation is needed.

Warm both probes, then **20 fixed AB/BA blocks**, six positions × three
seconds per side, CPU13 / nice19 / GOMAXPROCS=1, same explicit PGO.
Use paired aggregate NPS, geometric mean and two-sided 95% t interval on
log ratios, df19. Keep on topic only if lower ratio >1 and both order
strata >1; otherwise restore without extension. Microbenchmarks at
H256/512 × L16/32 describe inference cost and supply no verdict or Elo.

All builds/tests/searches run on WSL, CPU13,15; the factorizer GPU trainer
and SPSA CPU0–11 stay active. No live source/binary/net changes. Results
under `/home/ehrli/ngn-data/night-20261007/session16a/`, campaign status
stage `ngnn4-affine`; [controller](2026-10-08-ngnn4-affine-controller.sh),
[fixed check](2026-10-08-ngnn4-affine-check.py). Frozen common base archive
and two-file candidate patch are separate from all live jobs. A positive
verdict retains only an architecture-topic optimization; real training
and its fresh A/A40/gate400 remain required before strength acceptance.

## Completed verdict — session 17

Controller630484 completed **21:07:48 EDT**, exit0. **SHELVED**: all48
identities and20 fixed blocks pass completeness, geometric NPS ratio
**1.03902 [1.00863,1.07033]**, but AB/BA strata **0.98929 /1.09125** fail
the required both-positive rule. No extension or Elo interpretation. The
independent raw-record recomputation agrees; original box sample remains
in `session16a/`. The two-file local patch is preserved in the frozen box
source/patch and local scratch, then restored on the topic. No topic commit
or accepted-runtime change. [Reviewed result](2026-10-08-session17-results/affine-review.json).
