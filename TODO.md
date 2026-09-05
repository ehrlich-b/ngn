# NGN active runbook

Updated 2026-09-05. Active target: **at least 2800 without NNUE**. `CLAUDE.md` owns operating policy. Detailed evidence belongs
in `experiments/`. The former 132 KB runbook, including all old queue decisions,
is preserved in [the historical snapshot](experiments/2026-09-04-runbook-archive.md).

## Current state

- Current campaign source starts at `b378c432cb34`; certified search base is
  `50adc99` (T12 + T19). No T8 tuned vector or T22 patch is applied.
- **Fresh strength pin: 2726, reported 95% CI [2668,2784]**, September 4,
  200 games, 120+1/c8, five anchors. Original 2600 gate passes; **2800 remains
  unconfirmed**. Local historical-CCRL calibration, not an official rating.
  [Result, clock-loss sensitivity and audit](experiments/2026-09-04-strength-check.md).
  July's 2704 [2665,2743] used the old harness; the difference is not an
  engine-only gain measurement.
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
- **DEPLOYED:** qsearch correction committed at `ec9d264`, exact frozen WSL /
  Windows / Mac binaries. WSL source includes offline model/corpus work through
  `bf5c6b9`; no tuned weights are applied.
  Native WSL short/race suites and installed UCI smoke checks passed.
  [Paths, hashes, rollback and verification](experiments/2026-09-04-deployment.md).
- **`r0904pin` complete, DONE_EXIT_0.** Independent Stockfish audit passes
  all 200 games / 27546 plies. Zero NGN flags/errors. Blunder 6.1 flagged once;
  the actual pre-existing rule retains real clock losses. Excluding that entire
  anchor gives a separate sensitivity of 2753 [2691,2815], not the primary pin.
- **T22 resolved:** completed August 4, recovered September 4: 8000 games,
  2108/3768/2124 W/D/L, penta -0.7 [-6,+4], pLLR -0.62, post-min envelope
  [-1.50,+1.58], zero errors/flags. **Shelved** under its original rule.
  [Original manifest](experiments/2026-08-03-t22-probcut-cutnode.md).
- **T8 remains unconfirmed:** 24000-game SPSA completed; nine parameters moved.
  The vector is not in the engine. [Vector](experiments/2026-08-04-t8-spsa-result.md).
  T17 + the vector is now rebuilt in an isolated checkout on the repaired base;
  short/race/vet and UCI checks pass. [Confirmation manifest](experiments/2026-09-04-t8-confirmation.md).
  **`r0904t8confirm` cancelled without a verdict** at 279 completed games after
  an independently proved root-castling defect. Preserve the vector and rebuild
  after that correction. No interrupted games may be pooled into a new run.
  Main/default engines retain the accepted release. The old patch was manually
  rebased around the new draw helper rather than applied blindly.
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
2. **Frozen candidate ready:** `ngn_20260905_texelv2.exe` (4feb83b0...), all936
   compiled entries equal the fitted model. Full short/race/vet, UCI smoke and
   independent SF oracle pass (intentional numeric fixture change documented).
   Fresh baseline A/A `r0905qbaseaa` passed200 games (paired+15.65 [−16.71,+48.28],
   zero errors/flags). **`r0905texelv2` is running**, SPRT[0,+10], max1600 games,
   against the accepted qsearch baseline. Its bounded controller collects
   results but never deploys. Root must review the completed paired verdict;
   H0 or capped inconclusive shelves this fitted vector. [Game manifest](experiments/2026-09-05-classical-confirmation.md).
3. **T17 + the completed24000-game T8 vector rebuilt and staged on qcap**:
   `ngn_20260905_t8qcap.exe` (43b2a2c9...), isolated source, full short/race/vet
   and protocol checks pass. Not launched or installed. After the fitted-model
   verdict, use it only if qcap remains the accepted base; otherwise rebase and
   rebuild. [Exact artifacts and bounded original confirmation rule](experiments/2026-09-05-t8-qcap-confirmation.md).
   Do not reuse the interrupted279-game castling-bug run.
4. **Singular-only search candidate built and validated, not launched.** The fixed
   four-arm [depth-policy diagnostic](experiments/2026-09-05-depth-policy.md)
   supports testing removal of positive check/passed-pawn extensions, retaining
   singular extensions and consistent scout depth. Six changed choices on23
   frozen cases all have nonnegative SF depth16 differences; selected diagnostic,
   not Elo. Full plain/overlay identity, short/race/vet and independent oracle
   pass. [Frozen binaries, diff and game gate](experiments/2026-09-05-singular-only-confirmation.md).
   It must follow the fit/T8 decisions and rebuild if the accepted base changes.
   The old J1−59 estimate was65 interrupted games, not a completed rejection.
   [External2800 confirmation protocol](experiments/2026-09-05-2800-pin-plan.md)
   is prepared for an accepted gain, not launched.
5. **Futility diagnostic completed; no policy patch yet.** On seven frozen middlegame
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
   The next substantial search hypothesis is a coherent extension/reduction
   policy, with explicit interaction evidence before a playing candidate.

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
