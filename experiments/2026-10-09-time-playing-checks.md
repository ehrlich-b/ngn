# Current-base time candidate checks — session36

Source is a clean Git archive of accepted playing `ac658e2`, plus exactly
the existing [two-file time patch](2026-10-09-time-playing.patch).
One change: iteration admission observes the clock immediately. No other
time/search/net change. `night/time-playing` remains isolated and runtime
changes remain uncommitted pending the strength verdict.

After the bounded session35 NGNN4 pilot completes and releases12,14,
run four admission tests, full short/full race over all packages, vet,
and separate baseline/candidate Go1.25.5/GOAMD64v3 explicit-PGO builds.
Use taskset12,14/nice19/GOMAXPROCS2; preserve SPSA0–11 and factorizer13,15.
C: admission25.5GiB retains the25GiB floor and room for small build outputs.
Reuse the completed old-source mechanism witnesses/bank trajectories;
do not repeat them. Record exact input/source/binary digests and checks
in `session36-time-checks/candidate-receipt.json`, and append to `status.jsonl`.

Then check the separate [25-move horizon](2026-10-09-time-horizon.md)
candidate from another cleanac658e2 archive. Apply only its horizon patch,
run its four narrow tests and the same full checks/PGO build, and record
`horizon-receipt.json`. The two candidate directories/binaries remain
separate; no combined patch or game run. This prepares the next allocation
experiment while game verdicts and WSL restoration are pending.

These are code checks only. No strength games, default changes or
automatic adoption. Actual-base60+0.6 A/A40 and poweredSPRT[0,+6]
remain behind factorizer/provenance/final-vector SPSA; freeze the then-
accepted net, exact binaries, affinity/concurrency and complete game
manifest before games. Retain the declared minimum400/cap4000/H1 AND
lower95>0/zero-fault rule without extensions.

[Controller](2026-10-09-time-playing-checks.sh) and
[one-entry launcher](2026-10-09-session36-launch.sh) are staged on Windows
at `C:/Users/ehrli/ngn-session36/`, alongside the clean source/patch.
WSL entry currently fails. No successful execution, PID or pass is claimed.
After access recovers, inspect all started/terminal/PID markers first,
then invoke the launcher once. It delegates the already staged session35
factorizer/pilot launch and queues only the new time checks.
