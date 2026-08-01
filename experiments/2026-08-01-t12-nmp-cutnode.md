# 2026-08-01 T12 — NMP restricted to true cut nodes — BUILT, GATED, STAGED

**STATUS: built, gated, staged on the box with the on-box hash verified. NOT LAUNCHED.** Runs after T4e.
Engine tree is REVERTED and clean; nodecheck re-verified at exactly 346662/149587/765656 from a fresh build.
Change lives only in `output/t12-nmp-cutnode.patch` (sha256
`26b3d9c5d05848e6e054b610e9c94f9944101baf83e7d09d76c7b8496f75db9f`).

Supersedes the 2026-07-25 pre-verification in `experiments/2026-07-25-t12-nmp-cutnode.md`, whose mechanism
and confound findings are carried forward below and still stand.

## Premise RE-VERIFIED against HEAD (not trusted from the queue)

TODO's pointer to a shelved `output/t12` patch was **stale — no such file exists**, so the change was
re-derived from source. Both halves of the premise re-checked at HEAD:

- The NMP gate (`search.go:1482`) reads `... && canNull && !isPV && staticEval >= beta && ...` — **no
  cut-node condition**.
- `cutNode` is **already plumbed**: parameter of `alphaBetaPV` (`search.go:1177`), propagated as `!cutNode`
  at 1501/1568/2002 and `!isPV && !cutNode` at 1915, and already consumed by IIR at 1362.

So the candidate is literally `&& cutNode` inserted — a genuine one-line behavior change.

**A detail worth recording: the comment directly above that gate already says "Null-move pruning at cut nodes
only."** It has been inaccurate since the C3 fix — `!isPV` admits **both** cut nodes and all nodes. T12 makes
the code match the claim the comment has been making all along.

## Why the mechanism is sound

In PVS, a non-PV node is either a cut node (expected to fail high) or an all node (expected to fail low). NMP
assumes *"this node is probably already >= beta, so let the opponent move twice and check"*. **At an all node
that premise is contradicted by the node's own expectation**, so the null search is speculative work against
the search's own prediction. Restricting to `cutNode` removes the class of NMP attempts whose justifying
assumption does not hold.

## Gates

### Nodecheck — PASSES, and reproduces the 2026-07-25 measurement exactly

| position | base | T12 | delta |
|---|---|---|---|
| kiwipete d12 | 346662 | **451141** | **+30.1%** |
| mid d12 | 149587 | **136100** | **-9.0%** |
| end d16 | 765656 | **796906** | **+4.1%** |

**Byte-for-byte identical to the numbers recorded on 2026-07-25** (451141/136100/796906), which independently
confirms the re-derived one-liner is the same change that was pre-verified then. Measured from a fresh build
both sides. The **mixed direction is encouraging** — a pure slowdown would grow every position.

### The T18b soundness question — PASSES, in the strongest possible form

*Does the change persist speculative information?* **It does the opposite: it REMOVES a speculative prune**
from the node class where the prune's premise is known to be wrong. Nothing is cached, nothing is stored, no
heuristic guess gains durability. This is the structural inverse of T18b.

### Depth at fixed nodes — -2 net plies, NOT a rejection

| position | base d | T12 d | delta |
|---|---|---|---|
| quiet-mg-QGD | 14 | 14 | 0 |
| quiet-mg-closed | 12 | 12 | 0 |
| lateMg-ph6-11 | 13 | 12 | **-1** |
| najdorf-mg | 14 | 14 | 0 |
| rook-eg-R4P | 17 | 16 | **-1** |
| pawn-eg | 32 | 32 | 0 |

Under the **2026-08-01 narrowing** of the ebfprobe rule, a depth loss rejects only **depth-BUYING**
candidates — those that prune or reduce MORE and therefore claim depth (T6a, T20). **T12 prunes LESS by
construction**, so it claims soundness and pays in depth; the loss is its expected signature, exactly as for
corrhist (the kept T4c measures -1) and ttPv. Recorded as the size of the bill, not as a verdict: **-2 is
mild**, a third of T4e's -6, and 4 of 6 positions are flat.

### Suite

`go test -short ./engine`, `-race ./engine`, `./...` — all green with the patch applied. No new test: the
change is a one-line gate tightening with no new state, and the existing NMP tests plus nodecheck cover it.

## Mechanism evidence carried forward from 2026-07-25 (still valid — HEAD is unchanged in this region)

- **Proven to fire via a temporary direct probe: 473 NMP attempts blocked solely by the new gate at kiwipete
  d12.** Narrow (~3.1% of eligible NMP sites are all-nodes) but high-leverage (~220 nodes per blocked
  cutoff).
- **METHODOLOGY NOTE, keep this: `nmtry` is a CONFOUNDED witness for this change** — it went **UP**
  10850 -> 14796 under a strictly tighter restriction, because blocked all-nodes expand fully and their
  children are cut nodes, so the tree both grows and gets enriched in the node type where NMP still fires.
  The per-node rate rose too (3.13% -> 3.28%). **Absolute or rate counters cannot witness a restriction that
  changes tree composition — use a blocked-attempt probe.** Same trap as T20's `seeq`.

## PRE-REGISTERED EXPECTATION AND FAILURE HYPOTHESIS — written before launch, outcome unknown

**Honest expectation: +2 to +6, materially BELOW the +9.7 [S #230] prior, and a null is a live outcome.** Two
independent reasons to discount the prior, both registered here:

1. **The mechanism touches only ~3.1% of NMP sites.** A +9.7 mined from another engine's changelog was
   measured in that engine's calibration, not against NGN's already-guarded NMP (which carries the W8
   fail-low guard and the non-pawn-material zugzwang guard).
2. **kiwipete +30.1% at fixed real clock buys soundness with depth, which is exactly how T9a lost.** The
   -9.0% middlegame delta is the encouraging half; the +30.1% is the headwind.

**The base-rate warning belongs on the record too: this is a node-level pruning change, and that family is
1 keep / 11 attempts for +2.0 Elo total.** T12 is the best-motivated member left (a soundness argument rather
than a constant tweak, and it makes the code match its own comment), but the family prior is poor and this
candidate does not escape it.

**Leading failure hypothesis if T12 rejects:** the blocked NMP attempts at all nodes were *cheap and mostly
correct anyway* — an all-node that fails low costs little whether or not NMP was tried, so removing the
attempt buys back less than the +30.1% tree growth costs at real clock. **That would be evidence about NMP
node-type restriction generally, not about this constant**, since there is no constant here to blame — which
makes a null unusually informative for this lane.

**What a rejection would NOT license:** it would not reopen `DoubleExtMargin`-style constant hunts, and it
would not by itself close NMP polish (the eval-margin-scaled R variant is a separate, untested sub-item).

```yaml
id: 2026-08-01-t12-nmp-cutnode
date: 2026-08-01
change_class: search heuristic (one-line pruning-gate restriction)
hypothesis: >
  Restricting NMP to true cut nodes removes null searches whose justifying premise is contradicted by the
  node's own expectation. Expect +2 to +6, below the +9.7 prior.
base_commit_or_patch: HEAD (clean tree, nodecheck 346662/149587/765656 re-verified from a fresh build)
candidate_commit_or_patch: output/t12-nmp-cutnode.patch sha256 26b3d9c5d05848e6e054b610e9c94f9944101baf83e7d09d76c7b8496f75db9f
base_binary_sha256: 0a8f65b75e59c376052f83b14787b0af2eba65552cedfee6f70ce576891ca721   # on-box ngn_t5.exe
candidate_binary_sha256: 2081f5ac11095672c85a569bb24af407b8481076a1222e6d9716fe5fa10db4c4  # on-box ngn_t12.exe, hash verified after push
harness: sprt.exe sha256 30c33e0512725b7f552d8a1cf72ba6f1e0deb4dcb122c6b6ba8f6433c686a762 (M5-validated mill, untouched)
command: sprt.exe -new .\ngn_t12.exe -base .\ngn_t5.exe -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 8000 -mingames 300 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
machine: AMD Ryzen 7 9800X3D 8c/16t, native Windows (LAN box 192.168.4.108)
tc: 10+0.1 (seconds)
concurrency: 8
openings_path: sprt_openings.txt
openings_sha256: 974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222
adjudication: STANDARD-ON (M5-validated flags)
decision_rule: >
  H1 accept (pLLR >= +2.94) = KEEP. H0 accept (pLLR <= -2.94) = SHELVE. At the 8000g cap a PROVISIONAL keep
  requires point estimate >= +1; below +1 shelves regardless of pLLR sign. Verdict from the harness H0/H1
  line plus a full post-mingames envelope scan, NEVER the printed final pLLR alone. HALT if flag-outs are
  non-zero on either side.
next_action: launch after T4e resolves; base stays on-box ngn_t5.exe (do NOT pool with T4e -- one behavior change at a time)
```

## Base note

T12's base is on-box `ngn_t5.exe` `0a8f65b7…`, the same base T4e uses. **If T4e KEEPS, T12 must be rebuilt on
the T4e tree before launching — do NOT pool.** If T4e shelves (the more likely outcome per its own
pre-registration), this base holds unchanged.
