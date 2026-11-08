# Bounded wide SPSA

`cmd/spsa -iters 1200 -chunk-iters 100 -report 1` runs at most 100 new
iterations while retaining the 1200-iteration learning schedule. Repeat the
same command with `-resume`; chunk size is an execution limit and is excluded
from the checkpoint recipe. Other flags, binaries, and fixed options must stay
identical. A chunk must end on the report cadence unless it completes the
campaign. Checkpoints remain synced after every reported iteration.

`spsa_chunks.py` runs the pinned November campaign on NewPC. Its dedicated
runtime directory is `/home/ehrli/ngn-data/night-20261007/burn-1104/spsa`.
Prepare `ngn` from source `6d3168b8c5f15532cc7a352d05465a77ab50f361`
with `/usr/local/go/bin/go`, `GOAMD64=v3`, and `default.pgo`; build the updated
`cmd/spsa` separately. Copy the controller to `controller.py` in that directory.
The manifest records `source`, `engine_sha256`, `tuner_sha256`, and
`controller_sha256`. Inputs are checked before every invocation.

Start the controller with the user manager as transient service
`ngn-1104-spsa`, `Nice=0`, `KillMode=control-group`, `RuntimeMaxSec=86400`,
`TimeoutStopSec=15`, and a clean environment. It refuses other hosts, root,
changed pins, or less than 25 GiB free on C:. Engine processes inherit nice 0;
the tuner binds itself and the engines to CPUs 0–11. Build and test processes
should instead use CPUs 12–15 at nice 10.

Before campaign games, the controller stops a separate history at iteration 2
and resumes it to iteration 3, checking the unchanged prefix and advancing RNG.
It takes the existing `session138/cpu0-11.lock` with `flock` around every
invocation. Production chunks contain 100 iterations, with the lock released
and a 60-second rest between chunks. Each iteration plays six color-balanced
pairs at 10+0.1 with the pinned H768 net, book, Threads=1, Hash=64, and
OwnBook=false. The complete expanded registry is tuned together.

The wall-clock budget includes startup proof, lock waits, and rests. The
controller stops before 24 hours or at 1200 iterations, writes numbered chunk
vectors, and writes `final-vector.json` at terminal. A fault stops execution,
kills the tuner session, preserves the history, and records its reason in the
final vector, `events.jsonl`, the unit log, and the shared status stream.
There is no automatic retry, result adoption, or follow-up strength run.
