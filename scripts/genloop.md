# NGN generation driver

`genloop.py` is a Python 3 stdlib-only foreground orchestrator. Run stages on
WSL from an isolated worktree with NGN-built executables. It imports no trainer
or chess library; trainer subprocesses need the existing NumPy/PyTorch Python.
The commands below describe future runs, not authorization for large jobs.
Current authorization permits only development/tests, datagen <=2 minutes and
<=4 workers, with no detached workloads or scheduled jobs.

```bash
RUN=/home/ehrli/ngn-data/generation-0001
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
python3 scripts/genloop.py datagen --generation "$RUN" \
  --workers 2 --nodes 5000 --seconds 60
python3 scripts/genloop.py train --generation "$RUN" --python "$PY" \
  --engine build/ngn --epochs 1 --device cpu --threads 2
python3 scripts/genloop.py gate --generation "$RUN" --engine build/ngn \
  --sprt build/sprt --openings /path/to/ngn-owned-openings.txt \
  --games 8 --concurrency 2
python3 scripts/genloop.py status --generation "$RUN"
```

For a target generation, use `datagen --positions N`, optionally `--seconds S`
to limit each attempt. `--net PATH` selects an NGN-produced net; omission uses
HCE. The driver receives SIGTERM when the sum of 32-byte shard records reaches
N, the time limit expires, or a disk guard fires. Whole-game writes and the
sampling interval can overshoot N. Signals discard unfinished games and leave
closed, hashed shards. With a target, a time-limited stop is unfinished and must
be resumed; with only seconds, the timed generation is complete.

Both Windows C: and the WSL filesystem must have >=6 GiB free by default.
`--min-free-gib` changes the threshold; `--windows-mount` defaults to `/mnt/c`.
An actual mounted drvfs volume supplies C: free space, with Windows PowerShell
as fallback. An unavailable disk query fails closed. Guards run before datagen
and every `--poll-seconds` (default 2 seconds); they are a reaction threshold,
not a reservation against other writers. No other session's processes are
signaled. All subprocesses are joined; a stuck owned process group has a
30-second shutdown deadline, then a forced cleanup with an explicit error.

Every generation has an atomically replaced `manifest.json`, an append-only
`events.jsonl`, and separate attempt logs with exact argv, times and exit codes.
Each stage freezes input SHA-256s and options and records its source commit.
Completed stages verify artifacts and return without rerunning. `--resume`
continues an unfinished stage only with identical options and input bytes;
changed configuration requires a new generation directory. An exclusive lock
prevents overlapping stages. A recorded live process or unclosed/corrupt shard
blocks crash recovery for inspection; the orchestrator never adopts or signals
a process from an earlier invocation. Datagen resumes into fresh run directories
with fresh seeds, retaining earlier closed shards and counting their records.

Training freezes a seeded split of whole nonempty shards, with at least one
shard in each partition. Shards stay immutable. Fit resumes from the last
atomic epoch checkpoint; completed export/parity steps are verified and skipped.
One-bucket models export both NGNN1 and equivalent NGNN2; multi-bucket models
export NGNN2 because NGNN1 cannot represent their heads. Twenty evenly spaced
held-out records become board/STM FENs with neutral move rights. `trainer.refeval`
and the engine's UCI `eval` must agree exactly on all twenty for each export.
The manifest contains net/checkpoint/data/log digests, loss history and parity
receipts. Training progress is available at epoch boundaries in `history.json`.

Gate defaults to the completed NGNN2 export; `--candidate-net` selects another
owned net. `--base-net` selects an owned baseline, otherwise baseline is HCE;
`--base-engine` defaults to the candidate binary. UCI options are Hash=64,
Threads=1, OwnBook=false, then optional repeated `--newoption NAME=VALUE` /
`--baseoption NAME=VALUE`, then explicit evaluator selection. Games must be
even. The NGN opening file digest is frozen. Score stopping is disabled by
setting `mingames` above the requested count; clocks are always 10+0.1, with
default 400-ply cap and no score adjudication. The gate rejects voids, illegal
moves/openings, no-move results, time forfeits, missing games and incomplete
pairs. It records W/D/L, pentanomial counts, paired Elo and the runner's nominal
95% CI formula at full precision, including its variance floor and clipped
tails. This is a development comparison, not an absolute-rating claim or the
complete legal/terminal audit required by the claim protocol.

`--dry-run` prints the stage plan and options without subprocesses or writes,
including when binaries do not exist. `status` reads shard sizes, saved rate,
target ETA, epoch history, last gate tally and current disk space. The saved
rate is labeled as such; status does not start engines, update the manifest,
or poll continually. Unit tests require only stdlib, on WSL:

```bash
python3 -m unittest discover -s scripts/genloop_tests -v
```
