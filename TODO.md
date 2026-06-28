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
- Current local state: reset patch committed (engine P0s + regression tests + Makefile in `ba39088`, docs/evidence reset in the following commit). Baseline suite green at commit time. Items 2-8 are all DONE. Item 8 (strength re-pin) RESULT: **NGN 2666 CCRL, 95% CI [2627, 2706]** — multi-family (Blunder ladder 610/720/740/800 + Counter 3.8/2994) real-clock `-tc 120+1`, 400 games on the LAN 9800X3D (`experiments/2026-06-28-strength-repin.md`). FLAT vs the Jun-14 single-family pin (2655 [2606,2704]) ⇒ the correctness reset + items 2-6 PRESERVED strength while the instrument is now trustworthy (multi-family, ±40 CI, real clock, cross-family non-transitivity now visible). Active head is **item 9 (resume Elo levers)** — the first real "make elo higher" work, now on a clean engine + trustworthy instrument. NGN+gauntlet+anchors are cross-compiled to windows/amd64 v3 and staged at `~\ngn\repin\` on the box (the re-pin pipeline is now wired); `repin.pgn` there has all 400 games for lossxray when picking the first lever.

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

2. Done (`4899ae6`): both root drivers (`searchIterativeDeepeningUnsafe`, `searchFixedUnsafe`) now explore with `MakeMove`/`UnMakeMove`, so `pos.Positions` counts only played positions and the ply-1 child no longer reads an inflated count (the 2nd-occurrence-as-draw bug). Regression `TestRootExplorationDoesNotInflateRepetitionCount` (2nd occurrence not a draw, 3rd is); full short+race baseline green. Deferred non-regressing refinement: the root position is still not pushed onto the search RepStack, so a line that forces a return to the exact root is only caught via the game-history count, not as a 2-fold-in-search draw — track as its own isolated change if pursued.

3. Done (`792f4a3`): `p.EnPassant` is gated at both set-sites (makeMoveHelper double push + ParseFEN) via the existing `canCaptureEnPassant`, so the field invariantly holds only capturable EP squares; a phantom EP no longer splits the hash/repetition signature, and full==incremental stay consistent with no hash.go change. Adjacency gating (Stockfish/Polyglot-consistent): a pinned-but-adjacent EP is still hashed — the test asserts full==incremental there rather than same-as-no-EP. Regression `TestEnPassantHashCapturabilityGate`; perft byte-identical; stale `TestUCIFEN` (asserted the post-1.e4 phantom e3) updated to the gated NoSquare. The internal zobrist is separate from `PolyglotHash`, so opening-book lookups are unaffected.

4. Done (`3e52a54`): all five quiet-move prune/reduce gates that lacked it now carry `move.PromoType() == NoType` — futility (was queen-only `!= Queen`), LMP, history pruning, main LMR, and root LMR; quiet-SEE and quiet-tracking already had it. Move ordering scores underpromotions 70000 (after captures), so they were the exposed case. `TestUnderpromotionKnightForkFound` is a behavioral guard (engine finds the winning knight fork); perft byte-identical (move-gen unchanged), full short+race green. The deep-node exemptions are mechanism-proven (a promotion is forcing, not quiet) rather than red→green: the root searches all moves and a constructible fork is too shallow to be missed by a reduction, so deterministic isolation isn't practical.

5. Done (`3218b4d`): `quit` now breaks the scanner loop — `handleCommand` returns a quit flag the `Run` loop checks (red→green `TestUCIRunQuitTerminatesLoop`). `handleStop` joins the search goroutine by polling the atomic `searching` until the searcher clears it (in its defer, after its last write/state update), making stop / position-after-stop / quit synchronous and race-clean; `Run` also stop+joins on loop exit so the search never outlives `Run` (and `go infinite` + EOF now exits cleanly instead of hanging). A `syncWriter` serializes all output writes (search goroutine vs main loop). Four process-level `Run` tests; full short+race ./... clean; real binary verified (clean UCI session, clean quit, clean EOF-mid-search). Polling join chosen over a WaitGroup so go/stop on different goroutines (as TestUCIStop does) stay safe with no existing-test churn.

6. Done (`748bb4b`): chose the diagnostic-only path (SearchFixed is used only by perf/profiling tests + cmd/trace_move, never production). Documented: it shares alphaBetaPV with production, so repetition / extension-budget / stop handling are identical by construction; it diverges only in the root loop (full-width, no PVS/LMR) → larger node counts (not a strength/EBF measure) and the EXACT full-width minimax score that production's PVS/LMR only approximate (observed 4cp gap). Removed the dead `INFINITY->2000` workaround (suite green confirms it never fired; production runs the same full-window search without it). `TestSearchFixedMatchesProductionScore` asserts move parity + score within a 25cp tolerance. Out-of-scope leftover: the unused `alphaBeta` wrapper in search.go is pre-existing dead code.

7. Done (`918bed5`): `experiments/corpus_manifest.md` pins the gitignored opening set (5000 lines, sha256 `974e4b5...`) plus the tracked acpl/ratings corpora to checksums; `experiments/RUN_RECORD_TEMPLATE.md` is the structured (YAML) per-run record with a required `aa_preflight` field; CLAUDE.md Run Records now binds both and the A/A-preflight requirement for real-clock concurrent verdicts.

8. Done (2026-06-28, `experiments/2026-06-28-strength-repin.md`): multi-family real-clock re-pin on the LAN box — **NGN 2666 CCRL [2627, 2706]** over 400 games vs Blunder 610/720/740/800 + Counter 3.8 (the stronger-than-NGN cross-family anchor at 2994). Flat vs the Jun-14 2655; correctness reset preserved strength. Cross-family non-transitivity now visible (Blunder ladder rates NGN 2493-2731 with the 2155 rung saturating low; Counter rates 2766; weighted pool 2666). Not extrapolated single-family. `repin.pgn` retained for lossxray.

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
