# NGN next sprint — authoritative September5 update

The previous2800 goal is complete: accepted53e4d1b measured2884 [2834,2934] on the original fixed historical calibration; full legal400/55349plies and all400 terminal audits pass. Raw evidence remains in the main WSL checkout under output/corrected-pin-20260905.

New direction: strongest Go-based engine, with NNUE now permitted. All work and computation must run on WSL. This isolated worktree starts at the accepted playing source; no new engine behavior is installed. Next milestone: validated NNUE, actual multicore search and measured contributions, then profile-guided Go optimization. Read [the ranked roadmap and completion gates](experiments/2026-09-05-next-stage-roadmap.md). Public reference metadata is pinned under /home/ehrli/repos/ngn/output/next-stage-20260905.

Research and staged designs: [NNUE and multicore plan](experiments/2026-09-05-nnue-smp/report-source.md). Sol owns major implementation; root coordinates and reviews. Next: WSL baseline/runner controls, then exact one-thread ownership gates and the minimal Bullet pipeline. No production feature changes were made during research. [The WSL accepted-source build and required short/race baseline](experiments/2026-09-05-wsl-baseline.md) now pass; runner controls/A-A and feature integration remain pending. [NNUE N0 foundation](experiments/2026-09-05-nnue-n0-foundation.md) now passes integrated scalar/format/race checks; the default HCE binary is byte-identical to B0. [SMP M0 control ownership](experiments/2026-09-05-smp-m0-control.md) now passes integrated full short/race/vet and exact accepted cold/warm search parity. [NNUE N1a incremental context](experiments/2026-09-05-nnue-n1a-context.md) also passes integrated full-lane reference, allocation, full short/race/vet/build checks. [SMP M1 worker-history ownership](experiments/2026-09-05-smp-m1-history.md) passes integrated full short/race/vet, real receiver-isolation tests and exact accepted cold/warm parity. Next: Bullet export/parity bridge and evaluator/cache ownership; production NNUE and concurrent search remain gated on remaining integration/ownership.

The older runbook below is historical context. Its live-process claims, old no-NNUE objective and execution queue are superseded by this update and the roadmap. Do not restart historical workers.

---

# NGN active runbook

Updated 2026-09-05. Active target: **at least 2800 without NNUE**. `CLAUDE.md` owns operating policy. Detailed evidence belongs
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
  deployed on Mac/WSL/Windows at `5cf301a`. No T22 patch or fitted evaluation
  weights are applied.
- **Latest completed raw report: 2796 [2753,2838]**, accepted T8,400 games,
  120+1/c8. **Outcome audit HELD:** two NGN checkmates were recorded as cap
  draws by the old harness. No relabel/rerating;2800 remains unverified.
  [Completed result and audit](experiments/2026-09-05-accepted-t8-pin-result.md).
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
- **DEPLOYED:** accepted T17+T8 at `5cf301a`, inheriting the prior qsearch,
  castling and other correctness repairs. WSL source fast-forwarded to that
  commit; Mac/Windows retain the exact game-tested T8 binaries. Native WSL
  short/race and all installed UCI/default checks pass, with rollback retained.
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
   Deployment with rollback is next. The current installed engine is still T8.
   The Mac candidate remains user-interrupted, never resumed or pooled.
   The [corrected external protocol](experiments/2026-09-05-corrected-mate-pin.md)
   selects only this accepted candidate for `r0905matepin`, fixed400 games at
   120+1/c8 with unchanged labels/anchors/primary goal threshold. Exact artifacts
   and WSL-only validation package are staged/hash-verified; startup script AST
   passes. Require deployment, startup and idle box before the single launch.
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

Keep the active objective: at least 2800 without NNUE.
Search/eval heuristics need completed predeclared paired games against their
immediate base. Proxies do not establish Elo. A correctness repair with several
interacting entry points may use an explicitly combined integration test, but
must not claim per-fix Elo from the combined result.
