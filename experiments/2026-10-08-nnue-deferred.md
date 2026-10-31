# Deferred NNUE accumulators — session 15

Session starts **16:46:53 EDT**, deadline **18:26:53 EDT**. Pure-speed
candidate against immediate accepted engine `5b882e2` (archive `ab1ee58`,
same runtime), accepted PGO `616b1ebe` and KB8 H512 net `be34532c`.
Candidate source lives on `night/lazy-accumulator`; it is not accepted.

Search currently updates both NNUE perspectives on every made move, even
when a draw, transposition hit or in-check path returns without evaluation.
Retain the exact move delta and compute the accumulator only when evaluation
or an explicit accumulator read needs it. Each perspective walks back to its
nearest computed frame or saved king-context refresh, then applies unchanged
integer row kernels. King crossings alone save the thirteen piece masks;
the existing worker-owned refresh cache can reconstruct those positions after
the board has moved on. Undo reuses the parent's frame; sibling pushes clear
the reused child. No evaluation, network, pruning or timing policy changes.

The new sequence test compares fresh scalar/board oracles with delayed
evaluation across NGNN1/2/3, extreme weights, king crossings, castling,
en passant, promotions, nulls, branch reuse, reset and stack growth beyond
256 plies. The push microbenchmark now includes materialization so that it
continues measuring the same work as before.

Prospective fixed rule, declared before launch:

- Narrow deferred/NNUE/row differential controls, engine short/race,
  full short/race and engine vet must pass on WSL, CPUs 13,15 / nice 19.
- **36 exact identities**: NGNN3 and NGNN2 each twelve iterative/fixed
  depth-14 records, HCE twelve depth-11 records. Compare exact nodes,
  depth, score, best move and PV before any timing.
- Warm both probes, then exactly **40 interleaved AB/BA blocks** of six
  positions × three seconds per side on **CPU 13 / nice 19 / GOMAXPROCS=1**.
  Geometric mean paired aggregate NPS ratio; two-sided 95% t interval on
  log ratios, df=39. Accept only lower ratio >1 and both order strata >1.
  Otherwise shelve without extension; incomplete samples have no verdict.

Both sources/probes use the same explicit accepted PGO, Go 1.25.5,
GOAMD64=v3, Ryzen 9800X3D / WSL2. This uses a spare hardware thread while
GPU feed continues on its sibling CPU 12. Training stays on 12,14; SPSA
stays on 0-11. The measured gain is throughput under this campaign workload;
it is not an Elo estimate. No tuner pause, GPU job, games, admission change
or replacement of a running binary is part of this experiment.

Frozen baseline archive, candidate patch and both scripts reside at
`/home/ehrli/ngn-data/night-20261007/session15/`. The detached
[controller](2026-10-08-nnue-deferred-controller.sh) builds and tests, runs
the existing fixed [identity/timing check](2026-10-08-nnue-deferred-check.py),
and appends terminal status to campaign `status.jsonl`. Its receipt requires
review before promotion. No automatic merge or next behavior change.

Launch once after freezing the inputs:

```bash
S=/home/ehrli/ngn-data/night-20261007/session15
setsid nohup bash "$S/controller.sh" < /dev/null > "$S/controller.log" 2>&1 &
echo $! > "$S/controller.pid"
```

Launched **16:53:37 EDT**, observed controller **598985**. First launcher
598958 produced no start event or work; its empty log/PID are preserved.
All new and existing narrow tests pass in **2.531 s**; engine short **12.366 s**,
engine short/race **151.597 s**, full short/race and vet pass. The fixed check
started **17:00:11 EDT**. Timing result remains pending; no mid-run verdict.

Completed **17:25:23 EDT**, **SHELVED** by the prospective rule. All 36
identities and the required tests pass, but the forty timing blocks give
ratio **0.912687 [0.899307, 0.926266]**: **−8.73% [−10.07%, −7.37%] NPS**.
AB/BA strata **0.911532 / 0.913844** are both negative. No extension, no
accepted-runtime merge. This is a speed failure under the declared concurrent
campaign workload; no Elo conclusion follows.

The four tested files are preserved as **1b403d6** on pushed topic branch
`night/lazy-accumulator`. Independent raw-record recomputation checks every
identity, six-case block, order, duration, NPS aggregate, df=39 interval and
stratum; local files equal the tested candidate. [Build](2026-10-08-session15-results/deferred-build.json),
[fixed sample](2026-10-08-session15-results/deferred-timing.json),
[review](2026-10-08-session15-results/deferred-review.json). Raw files remain
in box `session15/timing/`. Existing tuner/trainer/admission/CUDA processes
were never paused or replaced.

Reopen only with a newly declared implementation that reduces the traversal
and frame overhead while retaining the proved exact arithmetic; this sample
cannot be extended or used to justify the current candidate.
