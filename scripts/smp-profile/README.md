# Accepted V1.2 SMP cost profile

This protocol measures search throughput and cost before another SMP policy is
considered. It admits no search change and runs no matches. The accepted engine
and V1.2 evaluator must remain byte-for-byte unchanged from
`8ba878cb03a7eb81ca465b3d562953d0f45e81c7`.

The required external model is `rodent_4kb_768hl_8ob_v2.bin`, 4,744,768 bytes,
SHA-256 `c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053`.
There is no downloader, fallback evaluator, synthetic model or checksum bypass.
The repository does not bundle the artifact. Supply the exact model locally at
`inputs/rodent_4kb_768hl_8ob_v2.bin`; this path is ignored by Git. The runner
verifies its size and SHA-256 before building or measuring; the historical
manifest path need not exist.

Run from the repository root, with an existing model and evidence outside the
checkout:

```sh
PYTHONDONTWRITEBYTECODE=1 nice -n 10 python3 scripts/smp-profile/run.py \
  --model inputs/rodent_4kb_768hl_8ob_v2.bin \
  --output /absolute/path/to/evidence
```

`--check-harness` compiles both probes and checks the overlap arithmetic without
loading a model or running a search. It is explicitly not acceptance evidence
for V1.2 performance. Every invocation creates a new evidence directory,
preserving earlier attempts. The full command fails with a recorded `BLOCKED`
report when the model is missing or has the wrong identity.

## Frozen protocol

- Six positions, including both sides to move, reuse the accepted V1/V2/V3
  search basket in `positions.json`. Hash is one shared 128 MiB allocation;
  each cell gets a fresh engine, TT and private histories. Configuration and
  model loading precede the timed search; evaluator root preparation and
  helper startup/join remain inside the search interval.
- Native throughput: three blocks of all six positions at Threads 1/2/4/8,
  2,000 ms per search, no node cap. Cell order is shuffled within each block by
  seed 20261023. Actual elapsed time, exact joined node totals, master depth,
  move, score, per-worker counters, TT and qsearch probe/hit/cutoff counts are
  retained. Timing includes setup and helper join; move overhead is zero.
- Six 400,000-node controls use Threads 1 only. The accepted source collapses
  node-limited search to one worker; these cells never enter scaling ratios.
- Separate native profile cells run every width/position for 2,000 ms with
  CPU, mutex and block profiling. `scope=search` labels include helper samples.
  CPU top tables, mutex/block top tables and `Cache.Get`/`Cache.Set` annotated
  tables retain the TT stripe and age-lock call sites. Mutex and block delays
  are summed goroutine waiting time, not wall-time percentages; samples
  attributed to unlock stacks must be interpreted as lock-holder sites.
- Separate diagnostic cells use the same fixed-time matrix and an external
  Go overlay, adding only a worker-local map of visited position hashes when
  `hash & 63 == 0`. Native throughput and profiles contain no map sampling.
  Overlap is `(sum(worker unique hashes) - union unique hashes) / sum(worker
  unique hashes)`; within-worker revisits are reported separately. This is a
  deterministic 1/64 position sample, not a 1/64 visit sample, and hash overlap
  does not prove redundant search: depth, bounds, repetition context and windows
  can differ. Instrumented NPS is not used in scaling ratios.
- All subprocesses have credential/token/secret/password environment variables
  removed, use nice 10, shared language caches, and external temporary paths.
  `time_v.py` uses `os.wait4` for elapsed time, CPU usage, peak RSS and context
  switches. It samples host CPU counters, pressure, load, memory, cgroup limits,
  process/thread scheduler delay every 250 ms, preserving unavailable counters
  explicitly. Non-owned busy CPU is a coarse host-counter-minus-owned-CPU
  estimate, not attribution to individual background jobs.
- `GOMAXPROCS=2` is fixed by the execution contract, including width 4/8.
  Logical-worker efficiency is speedup/Threads, with speedup relative to width
  1 aggregate NPS. These are oversubscribed worker measurements, not 4/8-core
  hardware scaling or playing efficiency. Shared-host scheduling, staggered
  measurement times and search-tree variation limit causal inference.

## Decision and reopen condition

Harness compilation does not satisfy acceptance and must not replace the matrix
in an automated readiness plan. Execute the full command above with the pinned
local model and assess the generated report before selecting a mechanism.
A missing or mismatched model remains a hard failure; use no substitute.

The protocol closes without admitting a policy and records a reopen condition.
Before claiming eight-core scaling, repeat under explicit authorization for at least eight Go
execution slots and retain shared-host telemetry. Reopen mechanism selection
only for a repeatable dominant measured cost and at most one materially different
mechanism. Initial-depth staggering, completed-helper result selection and the
shelved per-stripe sequence publication protocol remain excluded. Any later
playing-efficiency claim needs a separately authorized independent game gate on
an appropriate host; this protocol supplies no strength or Elo claim.
