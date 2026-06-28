# NGN Active Runbook

Reset date: 2026-06-28

This is the live project state and next-action map. It must be self-building: a capable agent should be able to start here, understand why the reset happened, pick the next task, know the acceptance gate, and update the file without needing a hidden conversation.

Historical plans, reviews, and postmortems are evidence, not instructions. When they conflict with this file, this file wins. Use `CLAUDE.md` for agent operating rules and `experiments/` for detailed run records.

## How This File Stays Useful

Keep this file compact but complete. It should answer:

- Where are we now?
- What is the active policy?
- What is the next ranked work?
- What must be true before an overnight run?
- What evidence is required to keep, shelve, or reject a change?
- What facts must not be forgotten?
- Where should detailed evidence be written?

Do not paste raw logs, long narratives, repeated reversals, or multi-page experiment diaries here. Write those to `experiments/` with a short link or summary here. If this file grows because a section is historical, move that section out or collapse it to one bullet.

## Current State

- Strength estimate: about 2655 CCRL blitz equivalent from the clean Jun-14 Blunder-family pin. The Jun-21 2716 read is useful evidence that NGN is above Blunder 8.0.0, but it is confounded and still single-family.
- Target: climb with classical HCE first. No NNUE before 2800; prefer waiting until 2900 unless an explicit later decision changes this.
- Main failure mode: speculative search/eval tuning was kept on weak self-play/proxy evidence while correctness and harness bugs remained open.
- Development priority: correctness and measurement integrity before more Elo hunting.
- Current local state: reset patch committed (engine P0s + regression tests + Makefile in `ba39088`, docs/evidence reset in the following commit). Baseline suite green at commit time. Active head is Immediate Queue item 2 (root repetition semantics).

## Why The Reset Happened

Claude's supplied diagnosis had to be treated as hypotheses, not truth. The code audit confirmed the load-bearing problems:

- `alphaBetaPV` and `quiescence` returned ordinary static evals on stop/time/node cutoffs, and parents could then update `bestScore`, alpha, TT, history, killers, capture history, and correction history from aborted child values.
- `quiescenceWithDepth` applied the six-ply cap before checking whether the side to move was in check.
- `UCIEngine.searching` had a confirmed race in `go test -race ./engine -run TestUCIStop`.
- `SearchFixed` diverged from production search by not setting `RootDepth`, using a different root make/unmake path, and carrying an `INFINITY -> 2000` workaround.
- `Hash` was advertised as a UCI option but was still a no-op in the active tree.
- `output/sprt_openings.txt` is gitignored and local-only. It has 5000 lines and SHA-256 `974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
- The uncommitted aux correction-history experiment was recorded as rejected and was removed from the active baseline.

The conclusion is not "everything is wrong." The core engine is worth repairing. The conclusion is that tuning must wait until stopped searches, UCI lifecycle, and experiment policy stop corrupting the verdict stream.

## Reset Patch Status

Fixed in the current reset patch:

- Interrupted child searches unwind without updating parent best scores, alpha, TT bounds, history, killer/counter/capture history, or correction history.
- Qsearch no longer returns static eval at the q-depth cap while the side to move is in check.
- Qsearch seldepth no longer double-counts qsearch ply.
- UCI `searching` is atomic and set before search launch; the confirmed `TestUCIStop` race is fixed.
- `Hash` UCI option resizes the TT, clamps to the advertised range, and persists across `ucinewgame`.
- `SearchFixed` sets `RootDepth` and uses root `GameMakeMove`/`GameUnMakeMove`.
- `make test` no longer swallows failing tests.
- Rejected aux correction-history code is not active.
- Added regressions for stopped-search TT pollution, qsearch-in-check at cap, UCI stop race, and Hash resizing.

Verified after the reset patch:

```bash
go test -short ./engine -count=1
go test -short -race ./engine -count=1
go test -short ./... -count=1
go test -short -race ./... -count=1
```

## Decision Policy

Optimize for Elo per confirmation hour, not Elo per coding hour. Default confirmation budget per change is 1-2 hours. Longer runs are deferred until the correctness queue and harness basics below are clean, unless there is a written reason that a short run cannot answer the question.

| Change class | Required evidence | If inconclusive |
|---|---|---|
| Provable correctness fix | Mechanism proof, regression test, short non-regression check if behavior changes | Keep only if the correctness claim is actually proved |
| Node-identical refactor | Exact node/score identity plus tests | Keep if identity is proved and maintainability or speed benefit is real |
| Pure speed change | Behavior identity plus repeatable wall-time/NPS gain | Shelve if speed gain is not repeatable |
| Search/eval heuristic | Completed predeclared paired game test against immediate base, with acceptance condition met | Shelve; do not keep on positive sign at cap |
| Tune | Holdout/proxy improvement plus paired games | Shelve or redesign |
| Lane closure | Multiple falsification attempts using the right instrument | Mark uncertain if evidence is proxy-only |

There is no "correctness-flavored" class. Either a change is a correctness fix with a proof and tests, or it is a heuristic.

## Operating Rules

- One behavior change at a time, candidate vs immediate base.
- Do not stack changes unless the run is explicitly an interaction experiment.
- No autonomous NNUE work.
- No long CPU burn for one verdict while correctness bugs below are open.
- Do not use `output/sprt_openings.txt` as unqualified canon. If a run uses it, record its SHA-256.
- Every game run must record base/candidate source, patch or commit hash, binary hash, command, TC, concurrency, machine, openings hash, game count, W/D/L or penta buckets, flags/errors, and verdict.
- Do not edit `scripts/cloudsprt.sh` while using it for a verdict. Harness edits require their own validation and A/A run.
- Do not commit speculative changes before verdict. Correctness/test/doc reset commits are allowed after tests pass.
- Update this file only when active state, policy, or ranked queue changes.

## Autonomous Work Loop

Use this loop for overnight work:

1. Re-read `TODO.md` and `CLAUDE.md`.
2. Check `git status --short` and preserve unrelated user/agent edits.
3. Pick the highest-ranked unblocked item from the Immediate Queue.
4. Prove the bug or missing behavior from code before editing.
5. Make the smallest scoped patch.
6. Add a regression test that would have failed before the patch when feasible.
7. Run the narrow test, then the required baseline tests.
8. If the change needs games, write a manifest before launching the run.
9. Do not start a second behavior change while a verdict for the first is pending.
10. Update this file only if active state changes; put detailed evidence in `experiments/`.

If uncertain, prefer a small correctness fix with a direct regression test over a speculative Elo idea.

## Immediate Queue

1. Done: reset patch reviewed, baseline-green (`go test -short [-race] ./...`), and committed. Codex P0 triage confirmed the patch covers the abort-time TT pollution, qsearch-in-check, atomic UCI, and Hash fixes; chess-accuracy P1s are items 2-6 below. Active head is now item 2.

2. Root repetition semantics:
   - Problem: root search mutates game-history counts via `GameMakeMove`, and child search can treat a second occurrence as a draw.
   - Target: game-history maps represent played positions only; search-path repetition remains separate and intentional.
   - Acceptance: tests for first, second, and third occurrence at root and below root.

3. En-passant hash/repetition normalization:
   - Problem: non-capturable EP squares can enter the hash and split positions with identical legal moves.
   - Target: hash EP only when an EP capture is actually legal or normalize EP state consistently.
   - Acceptance: tests for uncapturable EP, capturable EP, and pinned EP cases; full and incremental hash agree.

4. Promotion safety under forward pruning:
   - Problem: quiet pruning paths may treat underpromotions as ordinary quiet moves.
   - Target: all promotions are exempt from futility, LMP, history pruning, quiet SEE pruning, and LMR unless a promotion-specific proof says otherwise.
   - Acceptance: targeted underpromotion tests, plus node/score sanity on standard tactics.

5. UCI lifecycle hardening:
   - Problem: `quit` does not terminate the scanner loop; output and mutable engine state are still not owned by a process-level protocol model.
   - Target: process-level UCI tests for immediate `stop`, repeated `go`, `position` after stop, and `quit`.
   - Acceptance: race-clean process-level tests; no direct test races on `bytes.Buffer`.

6. `SearchFixed` authority:
   - Problem: it is still a second root implementation with special behavior and the `INFINITY -> 2000` workaround.
   - Target: either make it call the production one-iteration root path or mark it diagnostic-only and stop using it for conclusions.
   - Acceptance: fixed-depth diagnostics match production semantics for root repetition, extension budget, and stop handling.

7. Measurement cleanup:
   - Add a tracked manifest for the default opening set and any verdict corpus.
   - Add a lightweight result manifest template or JSONL ledger under `experiments/`.
   - Require A/A preflight for any real-clock concurrent self-play configuration.

8. Strength re-pin, only after items 2-6 are materially clean:
   - Multi-family, real-clock, with at least one opponent stronger than NGN.
   - Use Blunder for continuity and Counter/Fruit for transfer.
   - Do not extrapolate a single-family result as a narrow absolute rating.

9. Resume Elo levers after correctness/instrument cleanup:
   - Prefer correctness-adjacent search shape fixes with short gates.
   - Speculative search/eval heuristics must meet the decision table.
   - Speed work must prove identity and repeatable wall-time gain.

## Instrument Guide

- Unit/race baseline: `go test -short ./engine -count=1`, `go test -short -race ./engine -count=1`, `go test -short ./... -count=1`.
- Full repository race baseline: `go test -short -race ./... -count=1`.
- Absolute strength: real-clock `cmd/gauntlet` against multiple engine families.
- Self-play SPRT: useful for isolated behavior changes, but not a transfer guarantee.
- ACPL/Stockfish labels: useful eval filter, not a verdict for persistent in-game learning mechanisms.
- Nodecheck: required for node-identical claims and useful for detecting tree blowups.
- Game autopsy/lossxray: useful for choosing classes of work, not for keeping a patch.

## Run Record Minimum

Every game result needs a compact tracked record. Minimum fields:

```text
id:
date:
change_class:
hypothesis:
base_commit_or_patch:
candidate_commit_or_patch:
base_binary_sha256:
candidate_binary_sha256:
harness_commit:
command:
machine:
tc:
concurrency:
openings_path:
openings_sha256:
decision_rule:
games_or_pairs:
result:
flags_errors:
verdict:
next_action:
```

## Do Not Reopen Without New Evidence

- Aux correction-history tables: rejected in the recorded C-vs-B run. Reapply only as a fresh isolated experiment with reset tests and a predeclared gate.
- Large unregularized eval/PST tunes: prior evidence says overfit or no-op.
- Node-identical micro-speed lane: mostly tapped; new work needs a precise profile target and identity proof.
- Sign-at-cap keeps for speculative search/eval: banned by the decision policy above.
- NNUE: forbidden before 2800 and preferably before 2900.

## What Belongs Where

- `TODO.md`: live state, active policy, ranked queue, compact facts needed to continue.
- `CLAUDE.md`: agent behavior rules, commands, guardrails, how to update docs.
- `experiments/`: manifests, run logs, result summaries, negative results.
- `docs/`: stable architecture/spec notes, not live run instructions.
- Review files such as `codex_engine_review.md`: historical evidence and checklists, not policy.
