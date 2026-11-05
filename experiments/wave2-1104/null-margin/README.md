# Bounded null-move surplus reduction

`W2NullMargin` is a UCI spin option, default 0, range 0..1. It adds
`clamp((correctedStaticEval-beta)/200,0,2)` to the existing depth-based
null reduction before the child depth is floored at zero. It requires
`W2NullVerify=1`; unknown and mate-range evaluations or windows add nothing.
The input is the corrected static evaluation captured before TT refinement.
All eligibility gates and the parent's verification depth, suppression and
state restoration remain unchanged.

`candidate.json` predeclares the sole candidate: `W2NullVerify=1` and
`W2NullMargin=1`, 200 cp per extra ply, capped at two. No parameter sweep,
speed measurement or strength claim is part of this experiment. Adoption is
**DEFERRED** pending a completed rig strength gate; both defaults stay zero.
No quantitative rig match protocol was provided.

From the repository root:

```sh
nice -n 10 python3 experiments/wave2-1104/null-margin/verify.py fragment
```

The executable command runs only `GOMAXPROCS=2 go test -p 1 -count=1 ./engine
-run '^TestWave2Null(Margin|Verify)' -timeout 15s -v`, using the existing
native `time_v.py` accounting and shared Go caches. Proof subprocesses strip
credential/token/secret/password variables, restrict CPU/tensor threads to
two and hide GPUs. Logs are outside the checkout. `result.json` is generated
from actual test names and witnesses; reproduction compares deterministic
results, excluding elapsed time and resource measurements.

`TestWave2NullMarginReduction` intercepts the real `alphaBetaPV` null child
and asserts its depth and search flags for all four option combinations.
A shallow TT exact hit deliberately refines the working evaluation to 900;
reductions must still follow the pre-TT corrected value, including real
pawn-correction contributions crossing the 200 cp threshold. Cases include
negative, zero, 199, 200, 399, 400 and capped surplus, both mate boundaries,
unknown evaluation and depth flooring. Verification entry and path-state
restoration are checked. A rejected verification with both options enabled
must reach ordinary move search. The parent's six verification tests also
run, covering eligibility, actual recursive suppression, completion after
failure, cancellation, HCE/H768 sparse positions and default-off identity.

`parent-trace.json` was captured before product edits from the passed parent
`fdd3227fa50b50dea7cdbbeaab7b7c4a4f6f071c`, with verification enabled and
margin absent. It uses the parent's four FENs, depth 4, Hash=1 MiB, one
worker and embedded H768 evaluation. `parent-probe.go.txt` is the exact
capture program. To reproduce in a checkout of that parent, copy it to an
external `parent-probe.go`, build with `GOMAXPROCS=2 go build -p 1 -trimpath
-o /external/parent-probe /external/parent-probe.go`, then run the binary
with `experiments/wave2-1104/null-verify/fens.json` as its argument.
The original capture was bounded to ten CPU seconds for the entire process.
The margin-off test compares all fields: FEN, depth, nodes, qnodes, score,
move, PV, null attempts, null cutoffs, stop status and network digest.
The parent's pre-verification baseline is also reproduced by its own test.

`verify.py metadata` checks parent ancestry, author/committer identity,
clean committed HEAD, fixed candidate settings and unchanged defaults.
