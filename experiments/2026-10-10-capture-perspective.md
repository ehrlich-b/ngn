# Pair capture perspective updates — session42

Original pure-speed candidate on immediate accepted6c0fc26/be34532c,
isolated `night/capture-perspective`. The accepted-net profile assigns2.06%
flat CPU to CaptureRows. Fuse the two independent perspective calls for
two-remove/one-add moves with no king-context refresh, sharing one loop and
interleaving their int32 arithmetic. Quiet moves, king refresh and other
updates retain their original kernels. This is a separate candidate without
the passing quiet-pair patch; no foreign source was consulted.

Before timing, require guarded/misaligned/random/extreme/in-place and disjoint
controls at every16-wide increment through2048, AVX2/purego/race, engine and
full short/race suites, vet and explicit accepted-PGO builds. Preserve rig
provenance0–11 and data coverage/training12,14. Native pidfd wait for the
existing architecture controller4007 releases13,15 before any checks begin.
Scope13,15/nice19/2500MiB/swap0/CPUQuota200%; baseline probe86bdbc1c on6c0fc26
is already checked and reused. Runtime remains uncommitted until verdict.

Require36 exact search identities:12 iterative/fixed cases each for
NGNN3/depth14, NGNN2/depth14 and HCE/depth11, matching nodes/score/best/depth/PV.
Warm both probes, then exactly40 alternatingAB/BA blocks, six positions/3s
per side, Hash16/CPU15/nice19/GOMAXPROCS1. Geometric paired NPS ratio and
two-sided95% Student-t interval on log ratios, df39/critical2.0226909200367606.
Keep only lower95 ratio>1 and both order strata>1; otherwise shelve without
extension. Incomplete samples have no verdict; speed evidence gives no Elo.

Playing integration remains held for architecture disposition and the quiet
candidate. A passing isolated candidate requires compatibility and speed
evidence against the then-accepted runtime/net before integration; do not
infer its marginal gain on a combined or NGNN4 runtime from this sample.
