# NGN current state — September 19 SMP depth-stagger disposition

The original classical target is complete: accepted `53e4d1b` measured 2884
[2834,2934] on the fixed historical calibration, with 400 games, 55,349 legally
replayed plies and all 400 terminal outcomes audited. The active direction is the
strongest Go-based engine; NNUE and real multicore search are now in scope. All
implementation, builds, engines and matches run on the authorized WSL host. The
Mac is an editing and remote-control terminal only.

Development source through `a51233b` has integrated Lazy SMP, Counter 5.5
evaluation, evaluator startup selection and the accepted Counter AVX2 path.
Width 2/1, 4/1 and 8/1 HCE tests all accepted H1; Counter1/HCE1 and the longer
Counter8/HCE8 interaction also accepted H1. Fixed external cells completed for
Counter 5.5 at widths 1 and 8 and Rodent V1.1 at width 1. The exact Rodent V1.1
artifact cannot expose Threads at width 8, so that cell is unavailable without
substitution.

The current Counter candidate's one-thread rating orientation is **roughly
3000–3100**, not the former mixed-width 3130 headline. Accepted 100-game paired
cells give Counter1 relative Elo −323 [−402,−263], Rodent V1.1 relative Elo
−413 [−512,−338], and Counter8 relative Elo −186 [−230,−143]. Eight-thread
absolute rating is not established. All three used the same `a51233b` binary.
The September 13 output-kernel acceptance below is a subsequent pure-speed
improvement, not a new rating measurement. Historical 2884 remains separate.
See [the reproducible rating reassessment](experiments/2026-09-12-rating-reassessment.md).

**Optimized Rodent is now the stronger tested one-thread configuration:** on the
same accepted `7623779` binary, Rodent versus Counter completed 400 paired games
at 10+0.1, 187W/134D/79L, **+96.19 Elo [69.53,123.02]**. Fresh A/A and all legal,
operational and process audits pass. This is an accepted relative gain, not a new
absolute-rating estimate; do not simply add it to the older orientation above.
The existing HCE default and installed classical binary remain unchanged.

**Current user priority:** use existing networks, make the engine substantially
faster, and identify major gaps against direct competitors before pursuing small
heuristic changes. NGN-owned NNUE training, data generation and training-pipeline
work are explicitly deferred. A broad additional rating campaign remains
deferred: rating-list/settings uncertainty limits repeated anchor measurements.
After the newly accepted large speed gains, one bounded external Counter
checkpoint is admitted below to measure the changed engine. The historical external-release hold
below remains recorded, but completing every Rodent cell and porting the entire
external runner are not prerequisites to an isolated strength experiment.

The external queue is **not complete**. Rodent V1.2 width 1 stopped after 50
completed games when the exact release emitted the invalid promotion PV token
`g2g1`; strict trace policy correctly failed the cell. That partial score is not
accepted. Rodent V1.2 width 8 and both focused longer-clock V1.2 cells did not
run. The prepared `a51233b` Counter package is therefore still held. WSL
`build/ngn` remains the classical `53e4d1b` binary, while source has advanced;
the old coherent-install runbook
expects that checkout at `53e4d1b` and must not be run unchanged. No NGN match
or engine process was active during the September 12 read-only audit.

The direct quiet-promotion candidate completed its fresh 100-game A/A and fixed
400-game Counter-backed gate: 118W/178D/104L, paired **+12.17 [−11.30,+34.86]**.
All legal/terminal and operational audits passed. **Shelved** under the
predeclared positive-lower-bound rule; no extension or provisional keep.
That experiment made no accepted source change. See the
[completed run and archived patches](experiments/2026-09-12-qpromo-gate.md).

**Output AVX2 accepted and integrated in `66573f7`:** exact raw-bit and cold/warm
search identity, 2.0221x direct-kernel speedup and 27.85% less time on the fixed
one-thread cold-search fixtures. Every fixture and both order strata improve;
all 42 stages and the independent audit pass. Baseline tests, v1/v3 oracles and
Linux/Windows v3 packaging pass. This is not an Elo or eight-thread speed claim.
See the [completed experiment](experiments/2026-09-13-counter-output-kernel.md).

**Accumulator copy/update fusion shelved:** profiling the accepted output-kernel
base supported one exact-order candidate. Its fixed ten-block comparison
reduced time by 2.7407% on cold fixtures and 2.8703% on persistent-game rows,
missing the predeclared ≥3% threshold in both families. Every fixture and order
stratum improved, exact search identity and allocations matched, and all 42
stages completed cleanly. No production change, extension, deployment or Elo
claim. The patch is preserved, not lost or provisionally kept. See the
[completed speed experiment](experiments/2026-09-13-counter-update-fusion.md).

**Active goal: raise the rating to 3300.** This is not yet verified. Internal
relative gains and inactive compatibility milestones cannot close the goal;
eventual completion needs fresh, configuration-matched external evidence.

Next, in order:

1. Follow the [completed Astra xhigh competitor review](experiments/2026-09-13-competitor-review.md):
   the bounded correction diagnostic is complete with exact baseline search
   identity. Minor correction has material influence across five distinct sampled
   trajectories, without proof of harm. Root admitted one Counter-only minor
   application/learning-off candidate; source review and all six validation
   checks pass. WSL `ngn-counter-policy-gate-20260913/run-001` completed AA100
   and candidate400 (98W/207D/95L), but the frozen operational audit failed on
   benign interleaved startup-log lines. Independent diagnosis confirms clean
   process exits and no survivors; legal replay passes all 61,755 candidate plies.
   **Not adopted / shelved**, no accepted Elo, no replay or extension. Preserve
   the original failed audit. The future-run auditor now accepts strict startup
   record counts despite interleaving; all 27 harness unit tests pass. Future
   game gates still require their normal A/A preflight. See the
   [completed diagnostic and gate disposition](experiments/2026-09-13-counter-correction-interaction.md).
   Do not rerun the shelved fusion candidate or broaden to SPSA.
2. Existing Rodent V1.1 Anand **Slice A accepted as inactive compatibility**:
   strict external loader and portable full refresh match all 72 exact-release
   corpus records; package race and full short suite pass. No engine selection,
   bundled weights, or strength claim. See [Slice A](experiments/2026-09-13-rodent-anand-slice-a.md).
   **Slice B accepted as inactive compatibility:** exact incremental/full-refresh
   parity at 75 checkpoints, ten focused tests, 22 oracle sequences, package race
   and full short suite pass. The runner-only parent/subtest matcher failure is
   preserved; its narrow correction passed one bounded rerun with unchanged
   engine sources and oracle inputs. See the
   [completed Slice B verification](experiments/2026-09-13-rodent-anand-slice-b.md).
   **Slice C accepted as opt-in integration:** named engine/UCI selection,
   worker/search lifecycle parity, full short/race and configured exact-network
   integration race tests pass; existing defaults and search policy remain.
   See [Slice C](experiments/2026-09-13-rodent-anand-slice-c.md).
   The [bounded portable profile](experiments/2026-09-13-rodent-portable-profile.md)
   completed cleanly: output calculation was 46.11% of sampled CPU, incremental
   updates 33.69%. **Rodent output AVX2 accepted:** 10.08x direct-kernel speedup,
   45.55% less cold-search time, all six fixtures/order strata improve, exact
   cold/warm search identity and allocation equality; all 44 stages clean.
   See [the accepted speed gate](experiments/2026-09-13-rodent-output-kernel.md).
   The [optimized Rodent-vs-Counter strength gate](experiments/2026-09-13-rodent-counter-strength.md)
   **accepted Rodent**: 400 games / 200 pairs, 187W/134D/79L, +96.19 [69.53,123.02],
   all 60,198 candidate plies audited; AA100 and all runtime audits pass. The
   initial CPU rejection is preserved; after that unrelated job ended, one
   byte-identical-manifest launch completed without rerun or extension.
   Rodent is the new working reference at these tested one-thread settings.
   The [post-output profile](experiments/2026-09-13-rodent-postoutput-profile.md)
   completed cleanly: accumulator updates dominate at 58.21% flat CPU; output
   is now 2.69%. **Rodent update AVX2 accepted:** exact int16 arithmetic with
   copy/refresh/search unchanged, **58.84% less cold-search time** (~2.43x),
   all fixtures/order strata improving, exact cold/warm identity and allocation
   equality, all 44 stages clean. Full short/race, v1/v3 oracles and independent
   audits pass. See the [accepted update kernel](experiments/2026-09-13-rodent-update-kernel.md).
   This is another speed gain, not a measured Elo increment. Absolute 3300
   remains unverified. The [bounded external Counter checkpoint](experiments/2026-09-13-rodent-external-counter-checkpoint.md)
   is now **accepted with an explicit shared-host boundary**. `run-001` stopped
   at zero games on a startup-log layout error. Corrected `run-002` completed
   its games but was rejected before result audit by the then-frozen exclusive
   CPU rule; its candidate PGN remains unaccepted and unscored. On September 18
   the user explicitly directed that the Lean hopper remain running and the
   machine be shared. A result-unseen v4 amendment retained complete CPU
   telemetry but removed background load as a standalone rejection condition;
   engines, model, openings, 100+400 game cap, 10+0.1 clock, concurrency four,
   affinity, seeds, audits and no-extension rule stayed fixed. Fresh `run-003`
   completed A/A at 52/100, paired-bootstrap +13.90 Elo
   [−27.85,+56.07], then NGN Rodent scored 135W/163D/102L (216.5/400,
   54.125%) against exact Counter 5.5: **+28.73 paired Elo
   [+6.08,+51.62]**. All 67,082 candidate plies passed the legal audit; both
   supervisors and process witnesses were clean. The frozen external trace
   auditor initially failed because it keyed four concurrent `go` states only
   by shared display name. Its failed receipt is preserved; a separately hashed
   post-hoc correction changed only those state keys to `(trace thread, name)`
   and passed the unchanged trace. The score was already visible before that
   correction, and no game was rerun or extended. This is useful local evidence
   that current Rodent is stronger than this exact Counter artifact under these
   shared-host settings, not formal Elo or proof of 3300. Useful SMP work remains
   a later lane. The first such lane is now closed: the
   [helper initial-depth staggering experiment](experiments/2026-09-18-smp-helper-depth-stagger.md)
   passed WSL short/race tests and a fresh 20-game A/A, then scored
   66W/266D/68L in its fixed 400-game width-eight gate: **−1.74 paired Elo
   [−20.00,+16.52]**. All 59,130 candidate plies and runtime audits passed,
   the hopper stayed running, and shared-host CPU was disclosed. **Shelved** by
   the predeclared positive-lower-bound rule; the candidate source was removed
   and the run was not extended. Do not repeat this shallow staggering. The
   separate [completed-helper result-selection experiment](experiments/2026-09-19-smp-helper-result-selection.md)
   also passed its WSL suites, fresh A/A and all runtime audits. It scored
   83W/252D/65L in a separately frozen 400-game width-eight gate:
   **+15.65 paired Elo [−2.61,+33.98]** across 65,022 legal plies. This positive
   point estimate still missed the predeclared positive-lower-bound rule, so it
   too is **shelved**, unextended and absent from production source. Do not pool
   the two SMP experiments or use the near miss as a provisional keep. Owned
   training, new training data and another absolute-rating cycle remain deferred.
3. Before release installation, resolve or explicitly close the Rodent V1.2
   reporting boundary and amend the release contract. Rebuild the installation
   transition for the actual source state; its old `53e4d1b` precondition is stale.
4. Preserve the frozen external runner/results. Porting its approximately 2,000
   changed lines and resolving 27 inherited test failures is separate maintenance,
   not a reason to rerun accepted games or delay the next strength experiment.

See [the evidence and exact WSL artifact map](experiments/2026-09-12-release-state-audit.md).
The older September 5 runbook below is historical context. Do not restart its
workers or treat its queue as current.

---

# NGN active runbook

Updated 2026-09-05. **At least 2800 without NNUE: VERIFIED on the fixed historical calibration.** `CLAUDE.md` owns operating policy. Detailed evidence belongs
in `experiments/`. The former 132 KB runbook, including all old queue decisions,
is preserved in [the historical snapshot](experiments/2026-09-04-runbook-archive.md).

## Current state

- **USER RESTRICTION: never run NGN on this Mac; it is on battery.** All engine
  execution, matches, search tests, benchmarks, tuning and heavy chess validation
  must run on Windows/WSL. No background/E-core exception. At the user's request,
  all Mac match workers, controller19640/PID14811, collector22597/PID52989 and
  resumption helper49860/PID62206 were killed and verified absent. Do not restart
  them. `mac-gate/USER_STOPPED.txt` records the override. Mac mate-guard games
  remain interrupted with no candidate verdict; the separate Windows recovery
  protocol now owns further games.

- Current campaign source starts at `b378c432cb34`; certified search base is
  `50adc99` (T12 + T19). T17+T8 now passed its completed gate and is integrated;
  playing base is `5cf301a`. The accepted mate-stop correction is now deployed
  on Windows/WSL at `53e4d1b`; Mac defaults remain untouched. No T22 patch or
  fitted evaluation weights are applied.
- **Latest completed and validated report: 2884 [2834,2934]**, accepted mate-stop
  release,400 games,120+1/c8. The original primary lower95% bound exceeds2800;
  both families/all five anchors retained, scores bracket50%, zero errors/flags.
  Full legal replay400/55349plies and all400 terminal outcomes pass. **Goal met.**
  [Completed result, independent review and exact evidence](experiments/2026-09-05-2800-result.md).
  The prior2796 [2753,2838] report remains held for two cap-mate mislabels,
  unchanged and unpooled. [Prior audit](experiments/2026-09-05-accepted-t8-pin-result.md).
  The earlier200-game report was2726 [2668,2784], with its own later-discovered
  cap-mate mislabel. Local historical calibration, not an official rating.
  July's 2704 [2665,2743] used the old harness; the difference is not an
  engine-only gain measurement.
- **Terminal-result harness correction integrated:** mate/stalemate and rule
  draws take precedence over the cap and score adjudication. All159 Go/module
  files match the previously short/race/vet-tested archive; fresh native
  boundary tests and the corrected SPRT's200-game A/A pass. NGN playing source
  is unchanged. Future external games must use `gauntlet_20260905_terminal.exe`
  (499946e1...), not the old default. [Integration evidence](experiments/2026-09-05-draw-conversion-audit.md).
- The WSL checkout at `/home/ehrli/repos/ngn` was clean at `7587eb6` (early June),
  with equally old binaries. Authorized task: repair correctness, validate and
  deploy the current engine there, preserving rollback.
- **Earlier September repairs:** quiescence clocks/draws, mate at the fifty-move boundary,
  black en-passant validation, legal UCI parsing, and harness repetition counts.
  Removed an orphan experiment test that prevented compilation. Direct regression
  cases and an independent Stockfish move-generation audit prove the defects.
  [Scope, exact artifacts, commands and acceptance](experiments/2026-09-04-correctness-repair.md).
- **Optimized correctness repair ACCEPTED:** `r0904fast` completed 400 games,
  116/195/89 W/D/L, penta **+23.5 [+1,+46]**, zero errors/flags. Fresh A/A
  passed 200 games, penta −13.9 [−44,+16], zero errors/flags. The exact pawn-push
  witness keeps the corrected search identical while reducing legality-check cost.
  [Proof, timings, frozen binaries and completed results](experiments/2026-09-04-fast-correctness.md).
- The first repair lost −32.2 [−55,−10] and was not accepted. Its cost was
  profiled and reduced before the separate successful comparison. The original
  unmodified baseline comparison was invalidated by baseline-side illegal moves;
  the valid control adds only the required EP protocol fix. All raw runs remain
  separate; see the manifests for limitations and exact controls.
- **DEPLOYED:** accepted mate-stop correction at `53e4d1b`, inheriting T17+T8,
  qsearch, castling and prior correctness repairs. WSL source and exact frozen
  Windows/WSL binaries are updated; both corrected harness defaults are installed.
  Full WSL short/race/vet/oracle and installed UCI/default checks pass; rollback
  copies are retained. Mac defaults were not changed or executed.
  [Paths, hashes, rollback and verification](experiments/2026-09-04-deployment.md).
- **`r0904pin` complete, DONE_EXIT_0.** Original Stockfish legal replay passes
  all200 games /27546 plies, but a later all-final-board audit found game70's
  final-ply NGN checkmate mislabeled as a cap draw. Raw rating unchanged;
  [harness fix and carried-state search witnesses](experiments/2026-09-05-draw-conversion-audit.md)
  are isolated. The subsequent400-game pin also received that terminal
  audit before its held goal verdict. Zero NGN flags/errors. Blunder6.1 flagged once;
  the actual pre-existing rule retains real clock losses. Excluding that entire
  anchor gives a separate sensitivity of 2753 [2691,2815], not the primary pin.
- **T22 resolved:** completed August 4, recovered September 4: 8000 games,
  2108/3768/2124 W/D/L, penta -0.7 [-6,+4], pLLR -0.62, post-min envelope
  [-1.50,+1.58], zero errors/flags. **Shelved** under its original rule.
  [Original manifest](experiments/2026-08-03-t22-probcut-cutnode.md).
- **T17+T8 ACCEPTED:** the completed24000-game SPSA vector passed the fresh
  repaired-base confirmation `r0905t8qcap`:734 games,240/334/160,
  paired **+38.02 [+20.77,+55.45]**, H1, zero errors/flags. Exact source integrated;
  full short/race/vet, independent oracle and protocol checks pass.
  [Completed evidence and artifacts](experiments/2026-09-05-t8-qcap-confirmation.md).
  The old279-game interrupted castling-bug run was not pooled. Relative gain
  does not establish the absolute2800 goal.
- Batch 3 certified T12+T19 at +9.9 [+3,+17] vs Batch 2; Batch 4 had no keeps.
  [Batch ledger](experiments/batch-ledger.md).
- **Classical review:** [six-angle prompt](experiments/2026-09-04-classical-review-prompt.md)
  and [first findings with executable probes](experiments/2026-09-04-classical-review-findings.md).
  Reproduced a missing backward-pawn term in the gradient trace, clock-dependent
  quiet filtering, and immediate promotions admitted as quiet training data.
  **Tuning defects repaired:** [model/filter/checkpoint repair](experiments/2026-09-04-tuning-contract-repair.md),
  with frozen-trace parity under perturbed weights and conservative horizon handling.
  The optimizer also now considers the initial vector when selecting its best
  checkpoint. These offline changes carry no Elo claim. The extension/scout issue
  is known J1; its negative estimate came from 65 interrupted pre-reset games,
  not a completed rejection. Do not sell it as a new fix. No NNUE work is queued.
- **Accepted internal-castling correction:** actual loss GAME170 showed search
  PV `e1g1` while White was checked by Bh4; the UCI guard then substituted `e2g3`,
  immediately losing the knight. Reproduced from FEN/full history and independently
  verified by SF perft. Repaired in `70eb527`, including zero-legal-move root
  terminal scores. Short/race/vet pass; independent oracle covers 2,293 positions,
  48,926 divides and 128 internal choices. **`r0904castle` completed 400 games: 99/205/96,
  penta +2.6 [−20,+25], zero flags/errors.** Correctness gate accepted;
  strength result inconclusive. Deployed and smoke-verified on all three platforms. [Evidence and manifest](experiments/2026-09-04-castling-repair.md).
  The subsequent qsearch correction is also accepted and deployed: 400 games,
  106/184/110 W/D/L, paired −3.47 [−27.36,+20.38], zero flags/errors. The
  predeclared correctness gate passed; strength remains inconclusive.
  [Qsearch proof, full result and A/A investigation](experiments/2026-09-04-qsearch-budget-repair.md).

## Execution queue

1. **Classical model/data experiment complete offline:** producer `f556aef`
   finished3996 games (3989 complete;7 unresolved excluded),121818 retained rows.
   Full replay/corpus gates pass. The one joint fit improves integer validation
   MSE0.08457950→0.08406505 and final test0.07739660→0.07706922. The modest
   offline improvement is not an Elo claim. Fitted weights remain isolated;
   none are installed or committed. [Exact artifacts, checks and fixed corpus/fit
   plan](experiments/2026-09-05-classical-model-v2.md).
   The offline expansion exposes 138 existing mobility/threat coefficients;
   all have live activation/frozen-trace parity and all 936 stored values survive
   source import/rebuild. Initial playing eval source and weights are unchanged.
2. **Fitted candidate completed and shelved:** `ngn_20260905_texelv2.exe` (4feb83b0...), all936
   compiled entries equal the fitted model. Full short/race/vet, UCI smoke and
   independent SF oracle pass (intentional numeric fixture change documented).
   Fresh baseline A/A `r0905qbaseaa` passed200 games (paired+15.65 [−16.71,+48.28],
   zero errors/flags). **`r0905texelv2` completed and is shelved**:1600 games,
   480/654/466, paired+3.04 [−9.47,+15.56], pLLR−0.48, zero errors/flags.
   The fixed inconclusive-cap rule applies; no weights/defaults changed.
   [Completed audit and game manifest](experiments/2026-09-05-classical-confirmation.md).
3. **Fresh T8 baseline A/A passed as `r0905t8baseaa`.** Fixed200 games,
   49/108/43, paired+10.43 [−22.91,+43.95], zero errors/flags. Root independently
   verified the predeclared control. T8 remains the accepted installed base.
   [A/A and candidate manifest](experiments/2026-09-05-singular-t8-confirmation.md).
4. **Singular-only rejected:** `r0905singulart8` completed H0,794 games,
   paired−12.26 [−27.56,+2.99], zero errors/flags. The selected D interaction
   failed its game gate; it is not installed. The old J1−59 estimate still
   refers to65 interrupted games, not this completed test.
   **External400-game accepted-T8 pin `r0905acceptedpin` completed.**
   [Result2796 [2753,2838] and outcome-integrity hold](experiments/2026-09-05-accepted-t8-pin-result.md).
   Full legal replay400/56310plies passes. Terminal audit finds games211/216
   NGN checkmates mislabeled as cap draws. Raw results remain unchanged.
   WSL validation ended with HOLD; original gauntlet29692 and waiter22316 ended.
   All337 local inputs and all remote assets/raw snapshots passed final hashes.
   Original-run freeze is closed after the completed audit; preserve its artifacts.
5. **Cached-mate stop guard ACCEPTED.** NGN stopped on shallow cached mate
   scores and allowed immediate repetitions. The carried-state witnesses and
   WSL old-fail/new-pass regression prove the repair. The completed native
   `r0905mateguard` fixed400-game gate is91W/242D/67L, paired
   **+20.87 [+3.56,+38.29]**, zero flags/errors, DONE_EXIT_0. Root reviewed the
   complete parser/hash/drain evidence and accepts under the predeclared
   correctness rule; no sequential H1 or absolute rating claim is made.
   [Full gate and integration evidence](experiments/2026-09-05-mate-stop-windows-recovery.md).
   Main now includes the exact guard and both regression files alongside the
   accepted terminal harness fix. All162 staged source/data files match main;
   WSL full short/race/vet and independent oracle pass, all exits0/stderr empty.
   Deployed on Windows/WSL at `53e4d1b` with rollback copies and verified installed
   defaults/UCI behavior; corrected Windows harness defaults are installed.
   The Mac candidate remains user-interrupted, never resumed or pooled.
   The [corrected external protocol](experiments/2026-09-05-corrected-mate-pin.md)
   selects only this accepted candidate for `r0905matepin`, fixed400 games at
   120+1/c8 with unchanged labels/anchors/primary goal threshold. Exact artifacts
   and startup/launch evidence are preserved. **`r0905matepin` completed400,
   DONE_EXIT_0:2884 [2834,2934], goal verified.** Root and independent Sol review
   reconciled the original estimate, sample, pairings, labels and zero failures.
   WSL legal replay400/55349plies and all400 terminal outcomes pass; all input
   and pinned asset hashes remain unchanged. The original waiter hit a frozen
   script path-layout error after producing the report; one reviewed completion
   audit used byte-identical scripts in their original layout and passed.
   Original failed logs are preserved, with no games/relabel/rerating/restart.
   Gauntlet24264, waiter22592 and completion audit are terminal.
   [Root decision and complete evidence](experiments/2026-09-05-2800-result.md).
   The [quiet-promotion candidate](experiments/2026-09-05-quiet-promotion-candidate.md)
   stays separate and must rebase/rebuild onto the newly accepted guard before
   any further gate. No promotion patch is installed.

6. **Futility diagnostic completed; no policy patch yet.** On seven frozen middlegame
   disagreements, disabling futility at 400k nodes produces the same choices as
   the base at4M; five improve at least50cp by SF depth16. Selected sample, no
   Elo/correctness claim. The remaining16 controls include one83cp regression
   and three negative changes. Direct traces do not show futility skipping the
   watched SF replies; the local mechanism is still unproved.
   [Controls, traces and limits](experiments/2026-09-05-futility-followup.md).
   [Full ablation, audits and negative speed
   result](experiments/2026-09-05-search-ablation.md).
   Safe-check relevance was found in only one of49 tested continuations; more
   search also avoids it, so a king-danger transplant is not yet prioritized.
   [Fixed-sample study](experiments/2026-09-05-safe-check-study.md).
   Other candidates remain a coherent extension/search-policy review or richer
   classical king-danger inputs. [Classical comparison and actual
   safe-check witness](experiments/2026-09-04-classical-evaluation-gap.md);
   [loss/runtime and lower-priority TT findings](experiments/2026-09-04-review-second-pass.md).
   Existing nonlinear danger/mobility are not missing. The 24-game loss study's
   node probes did not establish evaluation as the cause. Follow the
   [larger-gain pathway](experiments/2026-09-04-strength-pathway.md); no NNUE.
   [September 5 review refresh](experiments/2026-09-05-classical-review-update.md):
   fresh 1,585-position base/fitted integer parity passes; rounding is small in
   this sample. Capture-promotion accounting is intact; quiet promotions remain
   absent. Exact block-picker experiment was about4% slower and is shelved.
   The selected singular-only interaction has since failed its completed gate;
   prioritize the proved carried-state mate failure above before further policy
   experiments. Any different interaction needs fresh mechanism evidence.

## Box access and artifacts

- SSH: `ehrli@192.168.4.108`, key `~/.ssh/id_ed25519`.
- WSL: `wsl -e sh -lc ...`; Ubuntu, user `ehrli`, Go at `/usr/local/go/bin/go`.
- Native Windows mill: `scripts/boxsprt.sh` uses `C:\Users\ehrli\ngn\sprt`.
  Both default `sprt.exe` copies now contain the corrected harness; old copies
  have the `.pre-repair-20260904` suffix. Versioned artifacts remain available.
- Poll: `scripts/boxsprt.sh tail <run>`. Fetch: `scripts/boxsprt.sh fetch <run> <dir>`.
  The helpers now include versioned harness names in process listing/cleanup.
  `BOXREPIN_BIN` selects the gauntlet binary; the default engine is `ngn.exe`.
- Run logs: `output/recovery-2026-09-04/`; SHA-256s and rules in the repair manifest.
- Opening set: `output/sprt_openings.txt`, 5000 lines, SHA-256
  `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
- No cloud workers were launched in this task. No speculative engine changes
  should be committed before their verdict. Correctness repairs require proof.

## Required validation

```sh
go test -short ./... -count=1
go test -short -race ./... -count=1
NGN_MOVEGEN_ORACLE=/path/to/stockfish go test ./engine -run TestMovegenOracle -v
```

The opt-in oracle compares depth-2 divides and incremental-state restoration;
its ordinary test skips when Stockfish is not configured. It found the black
EP defect that the prior standard perft coverage missed.

The at-least-2800 objective is complete on the fixed historical calibration.
Further research candidates above are unaccepted work, not a verified2900/3000 claim.
Search/eval heuristics need completed predeclared paired games against their
immediate base. Proxies do not establish Elo. A correctness repair with several
interacting entry points may use an explicitly combined integration test, but
must not claim per-fix Elo from the combined result.
