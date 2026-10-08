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

The timing verdict is **PENDING**. The checkpoint event determines launch;
after it, identities/warmup/fixed timing take about 25 minutes. If no iteration
>=325 arrives within 90 minutes of the wait, the controller fails without
pausing or restarting the tuner. Do not infer a speed gain from successful
builds/tests. No source edit or running binary is installed.

Next session: read `session6/{controller.log,controller.done,build-receipt.json,
checkpoint-ready.json,timing/receipt.json}` when present and terminal events.
Require the complete 36 identities and forty-block rule, unchanged frozen input
hashes, all required test exits, and exact candidate source inputs before deciding.
On pass, apply only `session6/candidate.patch` after checking the local engine
still equals baseline **df52dfc**, reconcile all compiled/test/profile inputs,
commit/push and update the accepted line. On failure, retain the baseline and
record the negative result without extending the sample. Detailed source/test
receipt can be assembled at review from the frozen trees and existing test logs.
