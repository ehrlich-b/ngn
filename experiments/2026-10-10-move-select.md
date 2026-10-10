# Four-lane move selection — session40

Pure speed candidate on immediate accepted `6c0fc26/be34532c`, original
PGO `616b1ebe`, isolated uncommitted topic `night/move-select`.
No third-party source was consulted. A current accepted-net, six-position
30-second profile on rig CPU14 shows `selectNextMove` at 10.77% flat CPU.
The candidate maintains four independent maxima for suffixes of at least
eight moves, then merges by score and earliest original index. Short scans
keep the scalar loop. Move scores, capture SEE, history, ties and swap order
are unchanged. This is a new selection mechanism, separate from the failed
TT scan and quiet accumulator unroll; no completed sample is repeated.

Before timing, compare every selection step to the scalar reference across
lengths1–256/eight patterns, including cross-lane ties, negative sentinels
and all remainder lengths. Run engine short/race, full short/race/vet on
the rig, then explicit-PGO UCI/probe builds. Reuse the previously checked
current-key baseline probe; no baseline source checks or builds are repeated.

Require36 exact search identities: twelve fixed/iterative records each for
NGNN3/depth14, NGNN2/depth14 and HCE/depth11, with nodes/score/best/depth/PV
identical. Warm each side, then exactly20 alternating AB/BA blocks, six
positions/three seconds per side, Hash16/CPU14/nice19/GOMAXPROCS1. Use the
geometric paired aggregate NPS ratio and two-sided95% Student-t interval
on log ratios, df19/critical2.093024054408263. Keep only if lower95 ratio>1
and both order strata>1; otherwise shelve without extension. An incomplete
run has no verdict. Speed evidence supplies no Elo estimate.

Rig directory `night-20261007/session40-move-select/`, tests/builds12,14,
MemoryMax2.5GiB/swap0/CPUQuota200%. Existing provenance3GiB/0–11 and
architecture2GiB/13,15 remain untouched; record both in the receipt. GPU
training is complete. Keep C:>=25GiB. Controller writes append-only campaign
status and terminal files; review results once before any topic commit or
integration. Estimated runtime20–30min, detached throughout. Preserve the
playing package while architecture games use their frozen immediate base;
a passing speed topic can be committed/pushed separately pending that verdict.
