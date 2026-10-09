# Session24 — temporal correction candidate

Session starts **11:12:52 EDT October9**, maximum deadline **12:52:52 EDT**.
Merged October9 plan item7 / Fable L5d / Astra item6 supply this lane.
All higher-priority code candidates are already prepared; C: **8.99GiB**
prevents new box builds/tests/games/training. SPSA continues unchanged.

## One isolated behavior change

Original NGN implementation on `night/continuation-correction`, worktree
`output/campaign-continuation-correction/`, immediate source **39eb128**.
[Source patch](2026-10-09-continuation-correction.patch). No external source
or net was opened or imported. No qsearch-TT/time/SEE/key/pawn overlay.

Add one worker-local temporal residual to the corrected static estimate.
The ordered pair of preceding real moves is keyed by resulting piece and
destination; promotions use their resulting piece, source/capture flags are
ignored, and side-to-move selects an independent table. Fold the pair into
16,384 slots per side, int16 storage, **64KiB per worker**. Require alternating
colors, a live two-ply path and both moves nonempty. Root/first-ply nodes have
no played-game context; a null move disables the next two nodes' context.
Main and qsearch legal moves record live ancestors before recursive search.

Applied weight3/64 and bounded1024 entry give at most **48cp**. Reuse NGN's
depth-weighted gravity update, learning the backed-up score minus the *total*
corrected static before TT refinement. Learn only at main search's completed
quiet fail-high or consistent upper/exact result. Skip checks, captures,
promotions, mates, invalid contexts, stopped and singular-excluded results.
Qsearch consumes the term after check/draw/stop handling and never learns it.
The existing board tables and all other policies retain their own plumbing.

The initial key/weight are placeholders for a one-change experiment, not
claimed optima. Correlated correction learning, key aliasing, context memory
traffic and altered pruning are risks for the games to decide.

## Predeclared checks and diagnostic

[Controller](2026-10-09-continuation-correction-controller.sh) natively waits
for session23-qstatic completion, then C:>=32GiB. Narrow/full short/full race,
vet and PGO builds use12,14, nice19. The baseline witness copies only the new
helper/table/test scaffold over39eb128, retaining pristine search.go: a seeded
quiet horizon must ignore the residual and fail the named assertion. This is
a missing-consumer witness, not a compile-error witness or game baseline.
All other baseline builds use pristine39eb128.

Candidate tests cover both colors/signs and addition to existing correction,
root/null/promotion context, signed clamps, bound direction, check/stop/mate/
capture/promotion/exclusion learning guards, reset/worker isolation, main-static
consumption, stopped-parent root restoration, qsearch evasion/capture ancestry,
terminal/check precedence and learning during a real search.

After all code checks, [resource diagnostic](2026-10-09-continuation-correction-resource.py)
runs exactly12 AB/BA blocks over the unchanged six1s fixtures on14 alone,
GOMAXPROCS1, accepted be34532c net, original PGO. Two fixed warmups precede it.
Pin actual binaries/source/net/argv/schedule before sampling. NPS ratio and
both order strata describe cost only; no depth/NPS threshold accepts or rejects
this eval candidate, and the sample is never extended.

Results: `session24-continuation/{controller.exit,controller.done,baseline-witness.log,
candidate-narrow.log,full-short.log,full-race.log,vet.log,build-receipt.json,
resource-manifest.json,resource.json}`. Expected runtime5–10min after admission.
No automatic games, source adoption or runtime-topic commit.

## Prospective playing verdict

After provenance/factorizer/data priorities and actual accepted-base replay,
write a fresh game manifest with exact candidate/base source, binaries, net,
instrument/openings and argv. A/A40 at the same setting, then paired10+0.1
SPRT[0,+6], alpha/beta0.05, minimum400/cap4000, concurrency8 on0–11,
GOMAXPROCS1/Threads1/Hash64/OwnBook=false/nice19, opening974e4b5a.
An exclusive CPU window requires SPSA completion or its durable checkpoint
handoff; do not stop the live tuner from this controller.

**Keep only completed H1 plus penta lower95>0 and zero flags/crashes/illegal/
no-move results. H0 or inconclusive cap shelves without extension.** No use of
the provenance upper95 exception, fixed-node depth or validation loss.
Any accepted-base change requires a new manifest/A/A; evidence does not transfer
automatically. No claim or absolute rating follows from this candidate.

## Local state

Source parses/formats with gofmt; patch applies cleanly to39eb128; diff and
shell/Python syntax checks pass. All chess code tests and resource measurements
remain pending on WSL. No Go build/test, engine or trainer ran on the Mac.
Speculative runtime source remains uncommitted; this tracked patch and frozen
box snapshot preserve it until the actual verdict.

Detached controller **675472** launched **11:25:47 EDT**, waiting on qsearch's
native completion. At launch C: **8.987GiB**, GPU0%, SPSA673954/673958 still
running with latest1025/12,300 checkpoint at11:00; all seven queued controllers
and hopper1168 are alive. No new heavy work started.
[Launch receipt](2026-10-09-session24-results/launch.json),
[end state](2026-10-09-session24-results/end-state.json).

Private origin `ehrlich-b/ngn` verified using the stored GitHub account;
the environment-token lookup could not resolve it. Only the tracked experiment
patch/controller/diagnostic/records and live handoff are committed/pushed on
night/night. Runtime topic source remains uncommitted pending its verdict.
No publication, external message, training, new game verdict or3600 claim.
