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
- **Active repair:** quiescence clocks/draws, mate at the fifty-move boundary,
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
- **DEPLOYED:** playing source `70eb527`, WSL and default Windows/Mac executables.
  WSL checkout includes subsequent docs/offline tuning through `e02be60`.
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
  checkpoint. These offline changes carry no Elo claim. The extension/scout issue is known J1, whose naive
  fix previously regressed; do not sell it as a new fix. No NNUE work is queued.
- **New highest-priority correctness defect:** actual loss GAME170 showed search
  PV `e1g1` while White was checked by Bh4; the UCI guard then substituted `e2g3`,
  immediately losing the knight. Reproduced from FEN/full history and independently
  verified by SF perft. Repaired in `70eb527`, including zero-legal-move root
  terminal scores. Short/race/vet pass; independent oracle covers 2,293 positions,
  48,926 divides and 128 internal choices. **`r0904castle` completed 400 games: 99/205/96,
  penta +2.6 [−20,+25], zero flags/errors.** Correctness gate accepted;
  strength result inconclusive. Deployed and smoke-verified on all three platforms. [Evidence and manifest](experiments/2026-09-04-castling-repair.md).
  No qsearch candidate has been installed as a default.

## Execution queue

1. **New proved qsearch interaction:** a real qDepth5 search stores a lower bound
   100 that warm qDepth0 accepts without searching; cold qDepth0 returns 51.
   Removing only the capture cap in a diagnostic overlay makes all three return
   51. Uncapped production qsearch is now implemented, regression/oracle/short/race
   checked and frozen. **Candidate `r0904qcap` running (400 fixed games):** the bounded A/A
   investigation passed: replication `r0904qaa2` completed 400 games at
   −4.34 [−25.98,+17.26]; all 600 games combined +7.53 [−10.22,+25.32],
   zero flags/errors. The first 200-game A/A failure remains recorded; the
   predeclared replication and combined intervals both contain zero. The
   fixed-node accounting diagnostic also passed all 12 pairs exactly.
   [Repair manifest and investigation](experiments/2026-09-04-qsearch-budget-repair.md).
   [Second review, corpus/profile evidence and lower-priority TT-age bug](experiments/2026-09-04-review-second-pass.md).
2. The [completed 24-game diagnostic study](experiments/2026-09-04-loss-study.md)
   and targeted search ablations exposed the castling defect. Four-times-node probes gave 19
   unchanged choices / five changes / no demonstrated rescue; several reference
   scores are unusable, so this does not establish an evaluation cause.
   Counter reports 1.39–1.77x NGN throughput in the local reference comparison;
   node definitions differ. Both search quality and runtime remain candidates.
   Tuning-contract repairs are complete; a future tune still needs a corpus
   acceptance audit and game-disjoint validation before expensive fitting.
   [Focused classical comparison](experiments/2026-09-04-classical-evaluation-gap.md)
   identifies frozen mobility/threat coefficients and missing safe-check inputs;
   an actual-function witness confirms the latter feature distinction. Neither
   establishes an Elo gain. Existing nonlinear danger/mobility are not missing.
   **Classical model v2 implemented offline:** all 138 appended mobility/threat
   coefficients have live activation and frozen-trace parity; all 936 stored
   entries survive source import and rebuild. Playing eval source/weights are
   unchanged. A replay-validated opening-grouped corpus pipeline and separate
   train/validation/final-test route replace anonymous rows for the next fit.
   [Implementation, evidence and fixed corpus/fit plan](experiments/2026-09-05-classical-model-v2.md).
3. Rebuild/predeclare T17 + T8 on the corrected base when warranted. Sync the new source/docs to WSL after native timed games; default playing
   artifacts are already deployed. Offline tuning repairs do not require
   replacing the frozen playing engine during a match.
4. Consider promotion-complete qsearch as an isolated game-tested candidate.
   Follow the [larger-gain pathway](experiments/2026-09-04-strength-pathway.md):
   a bounded study of fresh losses and controls selects one substantial search,
   runtime, or classical evaluation change. Avoid repeating the old feature queue.

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
