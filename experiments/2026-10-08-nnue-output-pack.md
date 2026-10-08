# Clamp packed NNUE output lanes — campaign session 4

Session began **12:34:34 EDT**, deadline **14:14:34 EDT**. Pure speed against
immediate accepted source **aa082a9**, PGO **616b1ebe**, accepted NGNN3 KB8 H512
net **be34532c**. Only `ngnn1OutputSmall` in
`engine/ngnn1_kernels_amd64.s` changes. No search, net, data layout, trainer or
match-runner change. No third-party source or net is used.

The accepted-net profile attributes **4.56%** of search time to this kernel.
Each perspective currently clamps two eight-lane int32 vectors before packing.
Pack with signed saturation first, then clamp sixteen int16 lanes together.
For every int32 value x,
`min(255,max(0,saturate_int16(x))) = min(255,max(0,x))`, since both saturation
endpoints lie outside [0,255]. Lane packing/permutation, multiplication,
accumulation and int64 reduction stay the same. This removes four vector clamp
instructions per loop over both perspectives. The 255 bound is broadcast as
int16. Loader-verified output weights remain in [-128,128]; full-range int16
weights retain the existing wide kernel and portable/race paths remain intact.

Prospective fixed rule, declared before builds and measurement:

- Existing output differential covers every width 16–2048 in steps of 16,
  guarded/misaligned views, full int32 accumulator extremes, full int16 wide
  weights, packed ±128 bounds, mixed signs, both perspectives and score caps.
  Require it, loader-dispatch boundary controls, and NGNN2/3 tests to pass.
  Run engine short/race, full short and engine vet on WSL.
- Require **36 exact search identities**: NGNN3 and embedded NGNN2 each twelve
  fixed/iterative depth-14 cases; HCE twelve depth-11 cases. Nodes, depth, score,
  best move and PV must match before timing.
- Warm both probes once, then exactly **40 interleaved AB/BA blocks**, six
  positions × three seconds per side, CPU 4 / nice 19 / GOMAXPROCS=1.
  Geometric mean of paired aggregate NPS ratios; two-sided 95% t interval on
  log ratios, df=39. Accept iff lower ratio >1 and both order strata >1.
  Otherwise shelve without extension. Incomplete samples supply no verdict.

The [controller](2026-10-08-nnue-output-controller.sh) uses isolated sources,
explicit identical PGO builds, tests on CPU 13,15 / nice 19, and drains/restores
only the current campaign tuner during identity/timing. Training continues on
12,14. The [fixed check](2026-10-08-nnue-output-check.py) saves all argv and raw
records. Root `/home/ehrli/ngn-data/night-20261007/session5/` contains baseline
archive/commit, exact candidate patch, pre-run manifest and frozen input hashes.
Go 1.25.5, amd64/v3, WSL2 Linux 6.6.87.2, Ryzen 9800X3D. The frozen release,
long-net gate and v3 development-control binaries are never replaced.

**Launched 12:37 EDT**, detached controller **559907**. Frozen baseline archive
SHA-256 `7664ff356efe1c2ae484b5b29f6336af2ab42b49abf80c96763b6a5c1f65960f`,
candidate patch `42249222ca9bc909457f5eb4f17bd9c494a998b515858a3aa945237f5fc44b8e`,
controller `68d0d7ee9f7b7bc63fa4a3cb1df0e18cdabcee66952ab6db9f9d730b44295f06`.
These are immutable box copies; this launch annotation is separate from its
pre-run manifest. Results expected about **13:08 EDT** at
`session5/timing/receipt.json`. Completion, not the ETA, controls review.

**ACCEPTED**, reviewed **13:08 EDT** after complete receipt/formula/source
reconciliation. All **36/36** search identities match. The forty blocks give
geometric NPS ratio **1.012059 [1.007220, 1.016922]**, AB/BA strata
**1.011232 / 1.012888**. Both predeclared conditions pass. This is **+1.21%
search throughput**, not a measured Elo increment or an absolute strength claim.

Output/NGNN2/3 controls pass (**1.158 s**); engine short **13.511 s**, engine
short/race **143.542 s**, full short and engine vet pass. All **227** compiled,
test, embedded-net and profile inputs match the tested candidate; only the
assembly file differs from the accepted base. Existing extreme-width tests
exercise the unchanged overflow/reduction guarantees. No extra runtime change
was added after the fixed sample.

Controller **559907** completed **13:06:30 EDT**, preserved both input hash
checks and restored SPSA as wrapper **566718**, tuner **566722**. Raw records,
both source trees, exact patch and immutable manifest remain in `session5/`.
Exact [build receipt](2026-10-08-session4-results/output-build-receipt.json),
[source/tests](2026-10-08-session4-results/output-source-tests-receipt.json),
and [full identity/timing receipt](2026-10-08-session4-results/output-timing-receipt.json).
Independent reconciliation recomputed every ratio, fixed interval and order
stratum, checked all forty indices, and compared every compilation input to the
local source. Existing long-net gate and v3 artifacts retain their original pins.

Operational observation: SPSA still has only iteration 300 / **3,600 retained
games** at review. The prior speed-controller pauses discard uncheckpointed
iterations; the driver writes every 25 iterations. Let it advance to a fresh
durable checkpoint before another optional timing interruption. No mid-run
parameter vector or discarded games are counted as tuning strength evidence.
