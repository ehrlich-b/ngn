# Rodent V1.2 L0 demand trace

Status: **held before execution**. This package asks whether demand-driven
V1.2 accumulators have enough measured headroom to justify an L1
implementation. It does not change the accepted evaluator or engine behavior.

The source is `9c8ecc7ba97fa8d5eb8bd6143bb08f35c907301b`, after V1.2 passed its
400-game Anand gate. The observer is a disposable patch applied only in an
isolated WSL worktree. Eager accumulator copies, updates, refreshes, and output
evaluation all remain in place.

## What is measured

Each pushed frame carries a two-bit demand mask, one bit per accumulator
perspective. An evaluation marks both current perspectives. On pop, demand is
propagated toward the root, but a king-perspective refresh terminates only that
perspective's ancestor dependency. This distinguishes direct consumers,
descendant-only consumers, fully abandoned frames, partially avoidable copies,
update calls, and refreshes. It deliberately does **not** infer waste from
`1 - evaluation_calls / pushes`.

The observer must preserve exact one-thread fixed-node trajectories: nodes,
completed depth, seldepth, best move, score, and PV. Every accounting family
must partition exactly, every search must unwind to evaluator depth zero, and
unbalanced resets are forbidden. The focused synthetic test proves that a
later White king refresh cuts the White dependency chain while retaining the
Black chain.

## Frozen workloads

- Cold: the established six V0 roots, 400,000 nodes each.
- Warm: the first three games selected without result or position filtering in
  the immutable PGN with SHA-256
  `35bdd00b28f585224e6c04392bdbf79cd4472c9ea7549b84acfc4379b07847d2`.
  Each game is replayed from the start; the same searcher, TT, history,
  evaluator, and repetition-bearing position are retained from the ply-23
  search through the ply-48 search. `NewGame` separates games. Each search is
  200,000 nodes.
- Both: `GOMAXPROCS=1`, engine `Threads=1`, `Hash=128`, Go 1.25.5,
  `GOAMD64=v3`, exact V1.2 model SHA-256
  `c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053`.

## Frozen decision rule

The post-V3 profile contributes whole-search shares: move accumulator copy
2.01%, update-2 2.87%, update-3 2.29%, refresh kernel 1.58%, and an unassigned
3.72% remainder inside `applyUpdates`' 10.46% cumulative share.

The directly attributed opportunity is:

```text
2.01 * abandoned_move_copy_fraction
+ 2.87 * abandoned_update2_fraction
+ 2.29 * abandoned_update3_fraction
+ 1.58 * abandoned_refresh_fraction
```

The optimistic ceiling adds `3.72 * max(abandoned fractions for update-2,
update-3, update-4, refresh)`, assigning all residual cost to the most
favorable measured class. Update-4's unisolated flat cost and null-copy cost are
otherwise uncredited; neither may be invented from aggregate `duffcopy` data.

If even that optimistic whole-search ceiling is below 3%, L1 is shelved with a
reopen condition. At or above 3% admits exactly one lazy prototype to a later,
separately frozen parity and timing gate; it is not itself evidence of a speed
gain.

The complete machine-readable contract is in `manifest-held-v1.json`.

