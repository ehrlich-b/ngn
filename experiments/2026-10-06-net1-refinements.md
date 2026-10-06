# Net1 refinements — 2026-10-06

**NEEDS_DISK; training, parity and gates have not started.** Initial Windows
C: free space was 0.421077728 GiB. The subsequent WSL setup invocation failed
with `Wsl/Service/E_UNEXPECTED` before producing setup output. A read-only
Windows check then reported **0 GiB free** and Ubuntu **Stopped**. The required
0.25 GiB cutoff prevents launching new work. No service restart, cleanup of
other workloads, data copy or datagen was attempted.

The [prepared NGN runner](2026-10-06-net1-refinements.py) is committed but
**has not been executed or validated on WSL**. It snapshots all closed
`gen0/data/*/*.ngnp` and `gen0b/data/*/*.ngnp` shards, excludes the same two
holdout paths from `trials-20261006/snapshot.json`, and uses NGNN2 H256/NB8,
batch 16384, BF16, seed 20261006 and two CPU loader threads. The last successful
read counted 28 shards and 38,680,916 raw records, including 1,895,755 holdout
records (36,785,161 training records before filtering).

| Trial | Lambda | Epochs | Cosine LR | Training / 20-case parity | Gate vs net1 |
|---|---:|---:|---|---|---|
| c1 | .75 | 80 | 1e-3 → 1e-5 | Not run | Not run |
| c2 | 1.0 | 32 | 1e-3 → 1e-4 | Not run | Not run |
| c3 | .9 | 32 | 1e-3 → 1e-4 | Not run | Not run |

Planned output: `/home/ehrli/ngn-data/net1-refinements-20261006/`. The runner
retains each trial's best checkpoint/export, deletes its latest artifacts
after exact holdout parity, verifies reexport identity and shard hashes, and
checks C: before every training run and gate. It builds the current engine
and owned SPRT runner with Go 1.25.5, GOAMD64=v3 and `go build -p 2`, then
runs runner/UCI tests. Planned gates match G1–G3: 10+0.1, concurrency 8,
400 games/200 reversed-color opening pairs, Hash 64, Threads 1, no own book,
no score stopping/adjudication; one match at a time. Opening file:
`/home/ehrli/ngn-data/net1-gates-20261006/openings.txt`, SHA-256
`974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.

No refinement has been accepted. Retain
`/home/ehrli/ngn-data/nets/net1/net1.nnue`, SHA-256
`6a203ffe128541ef422f9344f2906bffbebea6d93931bb1f50e27f44bf2beff6`.
The eventual selection rule is highest qualifying gate Elo with CI lower >0
or point estimate >=+10 and zero anomalies, copied to `nets/net1b/` with its
digest. No `net1b` acceptance or completed refinement receipt was produced.

[Net1 orientation](2026-10-06-net1-gates.md) is complete and independently
recounted: three-anchor pool **2947.76 [2891.92, 3003.60]**; four-anchor pool
**2945.57 [2893.90, 2997.24]**. All engine, trainer and data execution remains
WSL-only. Resume with the prepared script in an isolated WSL worktree after
host disk recovery; the Mac performs editing and Git/SSH operations only.
