# NGN active runbook

Updated 2026-09-04. `CLAUDE.md` owns operating policy. Detailed evidence belongs
in `experiments/`. The former 132 KB runbook, including all old queue decisions,
is preserved in [the historical snapshot](experiments/2026-09-04-runbook-archive.md).

## Current state

- Current campaign source starts at `b378c432cb34`; certified search base is
  `50adc99` (T12 + T19). No T8 tuned vector or T22 patch is applied.
- Historical strength pin: **2704, 95% CI [2665,2743]**, July 19, 400 games,
  real-clock 120+1, five anchors. This is a local CCRL-calibrated estimate,
  not an official rating. [Run record](experiments/2026-07-19-gauntlet-repin.md).
  The September repair has no new absolute rating; its new harness repairs
  also mean future comparisons must identify which instrument was used.
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
- **Now:** commit and deploy the accepted source/binaries, preserving rollback;
  then run the 200-game [fresh strength check](experiments/2026-09-04-strength-check.md),
  poll name `r0904pin`, five anchors at 120+1/c8. The 2600 target is not yet
  re-measured. Keep WSL builds/tests outside the timed gauntlet.
- **T22 resolved:** completed August 4, recovered September 4: 8000 games,
  2108/3768/2124 W/D/L, penta -0.7 [-6,+4], pLLR -0.62, post-min envelope
  [-1.50,+1.58], zero errors/flags. **Shelved** under its original rule.
  [Original manifest](experiments/2026-08-03-t22-probcut-cutnode.md).
- **T8 remains unconfirmed:** 24000-game SPSA completed; nine parameters moved.
  The vector is not in the engine. [Vector](experiments/2026-08-04-t8-spsa-result.md).
  Any future candidate must be rebuilt on the repaired current base, and its
  consumers rechecked. Do not blindly apply the old `output/t17.patch` after the
  search refactor. It is deferred behind the correctness repair and deployment.
- Batch 3 certified T12+T19 at +9.9 [+3,+17] vs Batch 2; Batch 4 had no keeps.
  [Batch ledger](experiments/batch-ledger.md).

## Execution queue

1. Commit the accepted repair, test it natively on WSL, preserve rollback, and
   deploy/verify the exact frozen binaries in the default Linux/Windows paths.
2. Finish `r0904pin`; require lower 95% CI >2600, two engine families, bracketing
   scores, and no NGN errors/flags. Report exclusions; do not extend on sign.
3. Update this file with rating and deployment pointers. Commit and sync final docs.
4. Resume strength work from this repaired base only after the current task.
   T8 needs a fresh candidate and explicit game manifest; no vector is applied.

## Box access and artifacts

- SSH: `ehrli@192.168.4.108`, key `~/.ssh/id_ed25519`.
- WSL: `wsl -e sh -lc ...`; Ubuntu, user `ehrli`, Go at `/usr/local/go/bin/go`.
- Native Windows mill: `scripts/boxsprt.sh` uses `C:\Users\ehrli\ngn\sprt`.
  `BOXSPRT_BIN=sprt_20260904.exe` selects the repaired harness. The old `sprt.exe`
  remains preserved and must not be used for new authoritative verdicts.
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

Keep the classical engine policy: no NNUE before 2800; prefer waiting to 2900.
Search/eval heuristics need completed predeclared paired games against their
immediate base. Proxies do not establish Elo. A correctness repair with several
interacting entry points may use an explicitly combined integration test, but
must not claim per-fix Elo from the combined result.
