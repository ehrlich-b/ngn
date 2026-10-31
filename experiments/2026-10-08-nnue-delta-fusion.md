# Direct NGNN3 move deltas — campaign session 5

Session begins **13:11:53 EDT**, deadline **14:51:53 EDT**. Pure speed against
immediate accepted source **df52dfc**, accepted PGO **616b1ebe**, and NGNN3 KB8
H512 net **be34532c**. No net, search policy, accumulator arithmetic or runner
changes. All execution is on WSL; the Mac only edits and reads receipts.

The current PrepareMove builds ordinary feature indices, returns the complete
delta, then passes it by value to kingMoveDelta and transforms each row in a
second walk. Build each row directly in its old king context and record any
crossing king's refresh context once. The order and counts of removals/additions,
castling rook squares, en-passant capture square, promotion piece, uint16 indices
and refresh masks are identical. Only NGNN3 selects the new builder; NGNN1/2
retain their existing path. The old two-pass transform moves to a test oracle;
special-move and deterministic random-game transitions compare complete deltas
in addition to the existing independent accumulator oracle.

Prospective fixed rule, before any build or measurement:

- Require the full NGNN3/NGNN2 controls and row differential to pass, including
  both colors, king buckets/mirrors, castling, en passant, promotions, undo,
  revisit, worker isolation, reload and full-range int16 arithmetic. Run engine
  short/race, full short/race and engine vet on WSL CPUs 13,15 / nice 19.
- Require **36 exact identities**: NGNN3 and NGNN2 each twelve fixed/iterative
  depth-14 records, HCE twelve depth-11 records. Exact nodes, depth, score,
  best move and PV before timing.
- Warm both probes, then exactly **40 interleaved AB/BA blocks**, six positions
  times three seconds per side, CPU 4 / nice 19 / GOMAXPROCS=1. Geometric mean
  of paired aggregate NPS ratios and a two-sided 95% t interval on log ratios,
  df=39. Accept only if lower ratio >1 and both order strata >1. Otherwise
  shelve without extension; incomplete samples give no verdict.

Isolated sources/private cache under `night-20261007/session6/`; exact accepted
PGO is explicitly identical for both builds. Go 1.25.5, GOAMD64=v3, Ryzen 9800X3D,
WSL2 Linux 6.6.87.2. The baseline archive, candidate patch, prospective manifest,
controller, check and checkpoint waiter are hash-frozen before launch. No
speculative runtime source is committed before the fixed verdict.

The [controller](2026-10-08-nnue-delta-controller.sh) builds/tests first, then
uses [native inotify and pidfd](2026-10-08-spsa-checkpoint-wait.py) to wait for
SPSA's next durable checkpoint (**iteration >=325**, 3,900 retained games),
with a 90-minute deadline. It does not pause the tuner before that event. It
then drains only the identity-checked current tuner group, performs the fixed
[check](2026-10-08-nnue-delta-check.py), and restores that tuner from its last
checkpoint on success/failure. File replacement, tuner exit, changed ownership,
timeout, missing trainer-controller or active admission games halt the probe.
No retries, sample extensions, duplicate monitor or periodic status commentary.

Longer-net controller **522940**, trainer **541007**, v3 admission **552631**,
frozen binaries and nets remain untouched; GPU training continues on CPUs 12,14.
The minimum checkpoint is a progress requirement, not tuning strength evidence.
Neither partial SPSA theta nor uncheckpointed games supply a strength verdict.

**Queued 13:19:41 EDT**, detached controller **566961**. Prospective manifest,
controller, check and native checkpoint waiter were committed/pushed as **34d0a5e**
before launch. Startup and its start event were observed; no duplicate launcher.

Frozen baseline archive SHA-256
`114470ecc0e907bae00011800d8639a3b9404d3fd11c0235f0d379449062c1e4`,
candidate patch `a3bce8c7664c6e57f07ed96cdce5f41de0532b1f08e01d74a305d42932b4eeb0`,
controller `605dab332b97d0238396cc322cd846dc08ea44e103264c819d385f57c083c9fe`.
The frozen-input list hashes to
`64c5b0701322cfe870ba2e1993da7731214efcd86b29ed2fd17f26f279c78465`.

At **13:22:51 EDT**, both builds and the delta/NGNN2/3/row controls pass
(**1.743 s**), engine short **13.167 s** and engine short/race **142.792 s**
pass; full short/race and vet remain under the controller. Actual Linux
notification, already-ready, timeout, file-replacement and wrapper-exit fixtures
all pass (`session6/checkpoint-controls.json`). SPSA has not been interrupted.

## Completed fixed verdict — session 7

**ACCEPTED**, after controller **566961** completes **14:01:00 EDT**. All
**36** identities match and all **40** fixed blocks are present. Geometric
NPS ratio **1.007972 [1.002689, 1.013282]**, order strata **1.006455 / 1.009491**:
both predeclared conditions pass. This is **+0.80% [0.27%, 1.33%] search NPS**,
not a measured Elo increment or an absolute-strength result. No extension.

Independent review recomputed every aggregate NPS ratio from all raw six-case
records, checked empty stderr, identities, block indices/orders/durations,
the fixed df=39 interval and both order strata. Pre/post binary/net/input hashes
are intact. Delta/NGNN2/3/row controls pass (**1.743 s**), engine short
**13.167 s**, engine short/race **142.792 s**, full short/race and vet pass.
All **230** Go/assembly/test/fixture/profile/embedded-net inputs match local
baseline bytes before applying the patch and tested candidate bytes afterward.
Only the three declared engine/test files differ. The optional trainer work
from sessions 6/7 is outside this compilation closure and is preserved.

Exact [build receipt](2026-10-08-session7-results/delta-build.json),
[source and tests](2026-10-08-session7-results/delta-source-tests.json),
[complete timing](2026-10-08-session7-results/delta-timing.json),
[independent verification](2026-10-08-session7-results/delta-verification.json).
Raw frozen source/patch/manifest and all records remain in box `session6/`;
review script `review-session7.py` reproduces the verification without engines.
No running campaign, gate, screen or admission binary was replaced.

The controller restores SPSA from durable iteration **325 / 3,900 games**,
wrapper **577981**, tuner **577984**, at **14:01:00 EDT**. Do not count discarded
uncheckpointed iterations or the current parameter movement as strength evidence.
