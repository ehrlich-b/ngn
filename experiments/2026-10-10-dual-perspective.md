# Pair quiet-move perspective updates — session41

Original pure-speed candidate on immediate accepted6c0fc26/be34532c,
isolated `night/dual-perspective`, original PGO616b1ebe. No foreign source
was consulted. The current-net profile assigns8.61% flat CPU to MoveRows.
This candidate fuses the **two perspective calls** for one-remove/one-add
moves with no context refresh, interleaves their independent arithmetic, and
shares one loop. It does not extend the shelved single-perspective loop-unroll
sample or the selection scan. Captures, castling and king refresh keep their
existing kernels. Source/destination aliasing and int32 wrapping stay exact.

Before timing, require guarded/misaligned/random/extreme/in-place and disjoint
controls for every16-wide increment through2048, plus purego and race fallbacks.
Run engine short/race, full short/race/vet and explicit-PGO builds on rig12,14,
MemoryMax2.5GiB/swap0/CPUQuota200%. Baseline probe86bdbc1c is already checked
on the current key source and reused. Preserve architecture13,15/provenance0–11,
forge conversion/SPSA, and the rig's native remote-coverage wait.

Require36 exact search identities: twelve iterative/fixed records each for
NGNN3/depth14, NGNN2/depth14 and HCE/depth11, matching nodes/score/best/depth/PV.
Warm each probe, then exactly20 alternatingAB/BA blocks, six positions/3s per
side, Hash16/CPU14/nice19/GOMAXPROCS1. Geometric paired NPS ratio with two-sided
95% Student-t interval on log ratios, df19/critical2.093024054408263.
Keep only lower95 ratio>1 and both order strata>1; otherwise shelve with no
extension. Incomplete samples have no verdict and speed provides no Elo.

Runtime remains uncommitted until the complete rule is reviewed. A passing
topic can be committed/pushed, while playing integration waits for the current
architecture verdict. If NGNN4 wins, rebase and verify using its actual trained
net before integration; NGNN3 speed evidence cannot establish NGNN4 speed.

Session41 closing06:08: full short/race/vet/explicit-PGO checks PASS, exhaustive
AVX2/purego controls PASS,36 exact identities PASS; fixed20-block timing is active.
No speed verdict yet. Checked source bytes are synchronized from rig formatting
and preserved with the receipt; runtime remains uncommitted.
[Build/check receipt](2026-10-10-session41-results/dual-perspective/build-receipt.json).

Metadata correction: the reused build-receipt template records a nominal
2.5GiB memory cap; the exact launch uses2500MiB (2,621,440,000bytes), a tighter
cap. Affinity, binaries, source and fixed timing contract are unchanged.
