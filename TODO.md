# NGN active runbook

Updated 2026-09-05. Active target: **at least 2800 without NNUE**. `CLAUDE.md` owns operating policy. Detailed evidence belongs
in `experiments/`. The former 132 KB runbook, including all old queue decisions,
is preserved in [the historical snapshot](experiments/2026-09-04-runbook-archive.md).

## Current state

- Current campaign source starts at `b378c432cb34`; certified search base is
  `50adc99` (T12 + T19). T17+T8 now passed its completed gate and is integrated;
  deployed on Mac/WSL/Windows at `5cf301a`. No T22 patch or fitted evaluation
  weights are applied.
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
- **DEPLOYED:** accepted T17+T8 at `5cf301a`, inheriting the prior qsearch,
  castling and other correctness repairs. WSL source fast-forwarded to that
  commit; Mac/Windows retain the exact game-tested T8 binaries. Native WSL
  short/race and all installed UCI/default checks pass, with rollback retained.
  [Paths, hashes, rollback and verification](experiments/2026-09-04-deployment.md).
- **`r0904pin` complete, DONE_EXIT_0.** Original Stockfish legal replay passes
  all200 games /27546 plies, but a later all-final-board audit found game70's
  final-ply NGN checkmate mislabeled as a cap draw. Raw rating unchanged;
  [harness fix and carried-state search witnesses](experiments/2026-09-05-draw-conversion-audit.md)
  are isolated. The live400-game pin also requires that supplemental terminal
  audit before any goal verdict. Zero NGN flags/errors. Blunder6.1 flagged once;
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
   **External400-game accepted-T8 pin `r0905acceptedpin` is running.**
   [Frozen protocol and additional terminal-audit requirement](experiments/2026-09-05-2800-pin-plan.md).
   Collector22597 only collects that existing run; do not restart the old failed
   preflight controller or launch competing Windows/WSL engine work.
5. **Next correctness priority: cached-mate stop guard.** Carried-state replay
   reproduces two depth1 mate claims that actually allow immediate threefold.
   Counter's depth/distance guard repairs both, and root's continued games
   reach checkmate. Isolated final builds, full short/race/vet and UCI checks
   pass. [Frozen candidate and prospective game gate](experiments/2026-09-05-mate-stop-confirmation.md).
   Exact candidate/harness/native-regression binaries and startup verifier are
   staged/hash-verified in native `sprt`; no startup or match launch yet.
   At the actual default Hash128, one immediate-draw failure reproduces;
   both guarded continuations finish in verified mate.
   Prospectively moved before launch to the available Mac: corrected terminal
   harness, fixed200-game A/A then fixed400-game comparison,10+0.1/c4 with
   taskpolicy background scheduling and score adjudication disabled. A/A passed:
   200 games,44/116/40, paired+6.95 [−21.12,+35.11], zero flags/errors.
   The400-game candidate now runs under controller19640/PID14811, started
   11:46:15UTC. The unlaunched Windows plan is superseded; no machine fallback
   or pooling. Main integration waits for the live pin's freeze.
   The [quiet-promotion candidate](experiments/2026-09-05-quiet-promotion-candidate.md)
   is also locally ready, but remains separate and must rebase/rebuild if the
   immediate accepted playing base changes. Neither candidate is installed.
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
