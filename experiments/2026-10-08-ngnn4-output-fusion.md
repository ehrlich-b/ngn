# NGNN4 fused output — prospective session 17 speed candidate

Topic `night/ngnn4-output-fusion`, immediate architecture base **9b6a3d6**.
The accepted playing runtime remains **5b882e2 / be34532c**. No real net,
search policy, layer dimensions, loader, trainer, or serialized format changes.

Fuse the post-affine signed division, int16 bias, [0,255] clamp, square and
wide-weight dot into one original AVX2 kernel. Signed division by64 is
`(x + ((x >> 31) & 63)) >> 6`, matching Go truncation toward zero for all
int32 values. Each weighted square fits int32, including weight−32768;
widen before adding products. The sum for L<=64 fits int64. The portable
kernel uses the same arithmetic contract. Existing affine/input kernels stay
unchanged; this is separate from the shelved sixteen-output affine tiling.

Predeclared rule: guarded AVX2/Go/independent scalar output comparisons for
L16/32/48/64, signed-division boundaries, int32 endpoints, all int16 bias/
weight bounds and sums exceeding int32; all existing NGNN4 controls and320
H512/L16 parity cases; engine/full short/race and vet. Then48 exact search
identities (12 each NGNN4 H512/L16 depth12, accepted NGNN3 and NGNN2 depth14,
HCE depth11). Same frozen owned random net `67043325` and explicit accepted
PGO on both sides. Microbenchmarks at H256/512 × L16/32 describe cost only.

Warm both probes then exactly20 fixed AB/BA timing blocks, six positions ×
three seconds per side, CPU13 / nice19 / GOMAXPROCS1. Geometric candidate/base
aggregate NPS ratio, two-sided95% t interval on log ratios with df19. Keep on
topic only if lower ratio>1 and both order strata>1; otherwise shelve without
extension. This provides no strength verdict or accepted-line change.

Results `/home/ehrli/ngn-data/night-20261007/session17f/`, status stage
`ngnn4-output-fusion`. CPU13,15 tests wait natively for corrected batched CPU
controller641098, preserving its tests. GPU factorizer and CPU0–11 SPSA stay
active. Freeze source archive, five-file patch, binaries, PGO and net before
identity/timing. No automatic merge, net training, or games follow.

## Completed verdict — session 18

Controller641437 completed **21:48:15 EDT**, exit0. **KEEP ON TOPIC**:
all48 search identities and exactly20 fixed timing blocks pass. Independent
raw recomputation gives NPS ratio **1.108407 [1.077645,1.140047]**,
AB/BA **1.164587 /1.054937**. Thus **+10.84% [7.76%,14.00%]** against
the immediate NGNN4 architecture base on its owned random H512/L16 net.
Guarded AVX2/portable/scalar and320-case integer parity, engine/full
short/race and vet pass. All five local source files equal tested source.
Exact topic commit **14a203a** is pushed on `night/ngnn4-output-fusion`.
No main merge, trained net, accepted-runtime change or Elo interpretation.
[Retained receipts and review](2026-10-08-session18-results/completed-topics.json).
