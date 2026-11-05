# Deep null cutoff verification

`W2NullVerify` is a UCI spin option, default 0, range 0..1. At effective
search depth 8 or greater, an enabled null fail-high is checked from the
restored position at `max(1, depth-NULL_MOVE_R)`, the same ply, and
`[beta-1,beta]`. A worker-local flag suppresses null pruning throughout that
search. The flag and path context are restored on both completion and stop.
Excluded-move searches and mate windows bypass this experimental null path.
Shallower and option-off cutoffs retain the original behavior. The existing
null reduction is unchanged. The new diagnostics also merge across workers.

From the repository root, run:

```sh
nice -n 10 python3 experiments/wave2-1104/null-verify/verify.py fragment
```

This executes only `GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run
'^TestWave2NullVerify' -timeout 15s -v`, with shared Go caches, two tensor
threads, disabled GPU visibility, a clean proof environment and native
`time_v.py` wait4 accounting. Raw logs go outside the checkout.
`result.json` is generated from actual passing test names and log witnesses;
reproduction compares those exact deterministic results, excluding duration.

`fens.json` contains the four depth4 fixtures; `baseline.json` is the actual
output of `probe.go.txt`, built against starting head
`3517a353a99bbe627a505ccc5463aadce047431b` before product edits. It uses
the embedded H768 network, Hash=1 MiB, one engine worker, a new engine for
each FEN, and compares depth, nodes, qnodes, score, move, PV, null attempts,
null cutoffs and stop status. `manifest.json` pins inputs, net digest, base
revision and baseline binary digest. To reproduce the original capture in a
checkout of that exact base, copy the probe text to an external `probe.go`,
then run `GOMAXPROCS=2 go build -p 1 -trimpath -o /external/probe
/external/probe.go` and `/external/probe
experiments/wave2-1104/null-verify/fens.json` (supply the committed fixtures
by absolute path if the base checkout does not contain them).

The injected search seam proves null fail-high / verification fail-low,
continued and completed ordinary search, verified cutoff, depth flooring,
shallow and off cutoffs, cancellation at null/verification entry and inside
an H768 verification descendant. Real recursive searches prove nested
suppression and sibling recovery. Position snapshots include board, hash,
clock, EP, side, tags and repetition map; checks also cover predecessor,
move/static-eval context, evaluator depth and active H768 accumulator.
Actual sparse K+R searches run with both real HCE and H768 evaluation and
production pruning, bounded by 2,000 nodes. A pawn-only guard is separate.

Rig strength/adoption status: **DEFERRED**. No games, speed samples or Elo
claims are part of this proof. Default play remains unchanged. A completed
rig gate is required before enabling the option by default; this task does
not specify a quantitative rig match protocol.
