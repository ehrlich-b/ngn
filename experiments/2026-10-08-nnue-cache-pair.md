# Pair changed rows during NGNN3 cache refresh — session 3

Pure speed against immediate accepted source `f473351` (same playing engine
as `8d9289c`), PGO `616b1ebe`, accepted KB8 H512 net `be34532c`. The
[preceding assembly experiment](2026-10-08-nnue-move-unroll.md) is shelved and
its edit is restored before this candidate. This experiment changes only
`ngnn1State.cachedRefresh` in `engine/ngnn3.go`; no assembly, net, search,
trainer or game-runner change. Long-net controller 522940 remains untouched.

The accepted-net profile puts refresh-cache work at **7.36% cumulative**. For
each piece type the existing algorithm independently subtracts every removed
row, then adds every new row. Pair one removal with one addition using the
existing fused move kernel, aliasing its source and destination exactly, then
handle leftovers normally. This replaces two complete accumulator traversals
with one for each pair. The feature multisets and signed int32 arithmetic are
identical; the cache can describe any earlier branch. Full-range int16 input
rows are supported and no network file or quantization rule changes.

Prospective rule, fixed before measurement:

- Existing NGNN3 independent-oracle, random game, mirror/bucket crossing,
  special move, undo/revisit, worker and reload tests must pass. The exhaustive
  row differential tests check exact aliasing, guarded/misaligned buffers,
  extreme arithmetic and every width 16–2048. Run required engine short/race,
  full short and engine vet on WSL.
- **36 exact identities**: accepted NGNN3 and embedded NGNN2, each twelve
  iterative/fixed depth-14 records; HCE, twelve depth-11 records. Require exact
  nodes/depth/score/best move/PV equality before timing.
- Warm both probes once; exactly **40** interleaved AB/BA timing blocks,
  each six positions × three seconds per side, CPU 4 / nice 19,
  GOMAXPROCS=1. Geometric mean of block aggregate NPS ratios; two-sided 95%
  t interval on log ratios, df=39. Accept only if lower ratio >1 and both
  order strata >1. Otherwise shelve, no extension. Interrupted/incomplete
  samples give no verdict. This is a fresh mechanism/sample, not an extension
  of the failed unroll experiment.

Separate WSL sources, binaries and private cache in `night-20261007/session4/`.
Same explicit accepted PGO profile for both probe builds, Go 1.25.5, amd64/v3,
Ryzen 9800X3D / WSL2. Builds/tests use CPU 13,15 / nice 19. The
[controller](2026-10-08-nnue-cache-controller.sh) drains only the current
campaign tuner during identities/timing and resumes from its last retained
checkpoint on success/failure. GPU training continues on 12,14. The fixed
[check](2026-10-08-nnue-cache-check.py) saves every raw record and exact command.
No speculative engine code is committed before verdict.

**ACCEPTED**, reviewed **12:27 EDT**, after complete fixed sample and independent
receipt/formula/source reconciliation. **36/36** search identities match; forty
blocks give geometric NPS ratio **1.018143 [1.012627, 1.023689]**, AB/BA strata
**1.015801 / 1.020491**. Both predeclared conditions pass. This is **+1.81%
search throughput**, not an Elo increment. No sample extension was needed.

Existing independent-oracle/cache tests and the guarded kernel differential
pass. Required engine short **13.259 s**, engine short/race **143.114 s**,
full short and engine vet all pass on WSL. All **227** compilation/profile
inputs match the tested source; only `engine/ngnn3.go` differs from the base.
The committed source is the exact candidate, with no added engine edits.

Controller **552951** started **11:55:06 EDT**, completed **12:23:46 EDT**.
It preserved both frozen input hash checks, resumed SPSA as wrapper **559707** /
tuner **559711**, and left training and gate binaries untouched. Raw records,
the rejected/accepted source snapshots and exact patches remain at
`night-20261007/session4/`. Source archive SHA-256
`5dea4f9332118b234138b3931374b75d8432764d0e85c7ba068602e14e43294f`,
candidate patch `06ce5633e9a4d4d7ef9d5126130d1dc7b667a16f37c7b80cf4126adb5932e2a6`.

Exact [build receipt](2026-10-08-session3-results/cache-build-receipt.json),
[source/tests](2026-10-08-session3-results/cache-source-tests-receipt.json),
[full identity/timing receipt](2026-10-08-session3-results/cache-timing-receipt.json).
Both probes use the same explicit profile; build argv in the controller resolves
OUTPUT to `session4/{baseline,candidate}-probe` in the corresponding source cwd.
All run argv are recorded in the timing receipt. The new implementation retains
portable/race Go kernels and the original AVX2 kernel; no third-party source,
net or data was added. Frozen long-net games and v3 development-control artifacts
retain their old source/binary pins; future clean builds inherit this change.
