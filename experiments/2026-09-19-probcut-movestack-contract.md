# ProbCut live-MoveStack contract: prepared F2 run

Status: **PREPARED, NOT EXECUTED**

The gaming PC became unavailable before F2 validation began. No test, build,
engine or match from this ticket has run. The goal was paused without starting
or terminating any WSL process. This record is the exact resume boundary; it is
not an accepted fix or correctness verdict.

## Candidate scope

Immediate accepted base: `f1f6b7e` (`Fix singular verification ProbCut contract`)

Prepared WSL worktree:
`/home/ehrli/repos/ngn-sol-probcut-f2-20260919`

Intended production diff: after a legal ProbCut capture is made and its
evaluator frame is pushed, assign that actual move to `info.MoveStack[ply]`
before entering quiescence or the reduced main search. This mirrors the normal
real-move lifecycle. It deliberately adds no slot restoration, history-policy
change, pruning change, or evaluator change.

Prepared local file hashes:

```text
d26651a27ad0c2f355c03817881c613e42a892c0e36d49f18d8023f3d42b63eb  engine/search.go
8386b2be9c004a77128a0265fc3584182c9f8782789a0068728d480c579cc1a6  engine/search_probcut_contract_test.go
```

The remote worktree currently contains an earlier test draft copied before the
PC pause. Do not run it as the final test. First overwrite only the test file
with the current local version and retain the accepted-base `search.go` for the
failing-before proof. Copy the candidate `search.go` only after that proof, then
verify both hashes match.

## Prepared regression contract

The three new tests are designed so clearing the slot, retaining the stale move,
or updating it only after child search all fail:

1. `TestProbcutMoveStackFeedsGrandchild` seeds `MoveStack[0]` with `e1d1`,
   proves the root ProbCut capture is `a1a7`, reaches a reduced-search
   grandchild, and requires its live two-ply ancestor context to be `a1a7`.
   The observer requests stop independently of the value read, then checks the
   stopped sentinel and complete board/history/frame/evaluator unwind.
2. `TestProbcutMoveStackIsLiveBeforeQuiescence` requests cancellation as the
   evaluator publishes the root capture. The assignment occurs at the ordinary
   lifecycle boundary immediately afterward; quiescence must observe the stop,
   unwind, and leave evidence that the traversed capture—not the stale seed—was
   the active slot.
3. `TestProbcutMoveStackTracksCompletedSiblings` gives the root two legal,
   non-losing captures. Because the evaluator observer fires just before each
   assignment, the next ProbCut sibling must see the prior sibling in the slot;
   the first normal root move must see the final ProbCut sibling. The search then
   completes normally and must restore the board, predecessor history and frame
   depths.

The local review explicitly accounted for `PushMove` invoking the transition
observer before `alphaBetaPV` assigns `LastMovePlayed` and `MoveStack`. Do not
move the production assignment before `mustPushMadeMove`; that would diverge
from the normal real-move transaction and could publish a move whose evaluator
push failed.

## Exact next-run order

Keep the hopper running and use the shared-host limits from the Sol plan. The
commands below are a runbook, not receipts.

1. Confirm the WSL worktree is still detached at `f1f6b7e` and has no changes
   except the expected test draft. Do not reset or clean it.
2. Copy the current local test file only, verify its SHA-256, and run the three
   `TestProbcutMoveStack` tests against the unmodified accepted source. Preserve
   stdout/stderr and require at least one intended stale-slot failure. A compile
   failure, fixture failure, timeout, or failure unrelated to the slot contract
   is not a failing-before proof.
3. Copy the current local `engine/search.go`, verify both hashes above, and rerun
   the focused group. All three tests must pass.
4. Run the complete `Probcut` regression group, then the baseline ladder.

Focused failing-before and passing-after command:

```sh
timeout --signal=TERM 120s taskset -c 6 nice -n 10 \
  env CGO_ENABLED=0 GOAMD64=v3 GOMAXPROCS=1 GOFLAGS=-mod=readonly \
  /usr/local/go/bin/go -C /home/ehrli/repos/ngn-sol-probcut-f2-20260919 \
  test -p 1 -vet=off -run TestProbcutMoveStack -count=1 -timeout=60s -v ./engine
```

Post-patch focused contract command:

```sh
timeout --signal=TERM 120s taskset -c 6 nice -n 10 \
  env CGO_ENABLED=0 GOAMD64=v3 GOMAXPROCS=1 GOFLAGS=-mod=readonly \
  /usr/local/go/bin/go -C /home/ehrli/repos/ngn-sol-probcut-f2-20260919 \
  test -p 1 -vet=off -run Probcut -count=1 -timeout=60s -v ./engine
```

Baseline ladder:

```sh
timeout --signal=TERM 600s taskset -c 6,7 nice -n 10 \
  env GOAMD64=v3 GOMAXPROCS=2 GOFLAGS=-mod=readonly \
  /usr/local/go/bin/go -C /home/ehrli/repos/ngn-sol-probcut-f2-20260919 \
  test -short -p 2 -count=1 -timeout=8m ./engine

timeout --signal=TERM 900s taskset -c 6,7 nice -n 10 \
  env GOAMD64=v3 GOMAXPROCS=2 GOFLAGS=-mod=readonly \
  /usr/local/go/bin/go -C /home/ehrli/repos/ngn-sol-probcut-f2-20260919 \
  test -short -race -p 2 -count=1 -timeout=12m ./engine

timeout --signal=TERM 900s taskset -c 6,7 nice -n 10 \
  env GOAMD64=v3 GOMAXPROCS=2 GOFLAGS=-mod=readonly \
  /usr/local/go/bin/go -C /home/ehrli/repos/ngn-sol-probcut-f2-20260919 \
  test -short -p 2 -count=1 -timeout=12m ./...
```

If all gates pass, replace this prepared status with the actual receipts and
validated WSL hashes, commit F2 separately, push the private `main` checkpoint,
and advance to the paired C0/V0 diagnostics. Do not claim Elo, run a match,
install a binary, or begin C0/V0 while F2 remains unaccepted.
