# Capture pairing on the accepted quiet-speed base — session44

Original capture-only sample is complete: 36 exact identities and 40 fixed
blocks, NPS ratio 1.0316125 [1.0252475, 1.0380170], both order strata > 1.
Raw aggregates reconcile. Retained as c66fbc1 on night/capture-perspective;
this result on 6c0fc26 does not establish the marginal benefit on 56b35ef.

Apply only that capture kernel to the immediate accepted 56b35ef/be34532c.
The existing quiet dispatch and new capture dispatch select disjoint delta
shapes. Numerical arithmetic, feature rows, refresh rules, search and net
remain unchanged. Exact combined patch is tracked; runtime is uncommitted
on night/capture-quiet-current until this distinct combination's verdict.

New source checks/builds run on forge, restricted to CPUs 8–11, nice 10,
2 GiB/swap0, alongside the unchanged 2.5 GiB SPSA and 128 MiB reader scopes.
Require the capture and quiet arithmetic controls on AVX2/purego, full short
and race suites, vet, and explicit accepted-PGO builds of both probes/engines.
No strength games run on forge and no games from different hosts are pooled.

Only after C: is restored above 25.5 GiB, run on rig CPU15/nice19/GOMAXPROCS1:
36 exact identities (NGNN3/NGNN2 depth14, HCE depth11), warmup, then exactly
40 balanced AB/BA blocks, six positions and 3 seconds per side, Hash16.
Use the original geometric log-ratio Student-t interval (df39,
critical2.0226909200367606). Keep iff lower95 ratio > 1 and both order
strata > 1; otherwise shelve this combination without extension.
An incomplete sample has no verdict. No speed result implies an Elo gain.

The old time-admission controller stopped at capacity before tests or games.
That leaves a clean game boundary: integrate capture only after its own
current-base result passes, then nominate time-admission on that actual
accepted base. If capture shelves, retain 56b35ef. Do not start a time gate
while this playing-source decision remains pending.
