# Move the frozen SPSA continuation to forge — session38

Current-plan directive: CPU-only NGN work moves to forge; only CPUs8–11 and
at most5GiB belong to NGN. Every job uses Bryan's user-systemd scope with
AllowedCPUs8–11, CPUQuota400%, MemoryMax5G and MemorySwapMax0, plus explicit
taskset8–11/nice10. Work lives under `/work/lanes/ngn-spsa-20261009`, shared
Go caches under `/work/cache/go/ngn-*`. No other worktree/job is touched.

Build the exact frozen legacy engine source from rig `kb8bin/src`, retaining
its embedded be34532c net. Copy Go1.25.5 into `/home/ehrlich/.local/ngn-go1255`
so compiler and GOAMD64v3 match the frozen rig environment. Build the frozen
legacy tuner source from rig `spsa/src`; the currently committed modern tuner
cannot resume this legacy JSONL and is not substituted.

One necessary execution adaptation: preserve pairs6/iters2000/rend.05/
afrac.1/report25/seed20261008/TC10+0.1 and all update mathematics, while
`-concurrency 2` reuses two engine pairs for six perturbations in three waves.
This keeps12 games per optimizer step and the24k endpoint. Setting pairs2
would change the gradient averaging and cut the endpoint, so it is avoided.
The patch changes worker scheduling only. Engine defaults, including Hash128,
remain unchanged; four engines fit the5GiB envelope. A race-enabled test
requires every one of six perturbations exactly once and exclusive use of
both workers. The known legacy RNG reset on resume remains recorded.

After setup/checks pass, stop only the identified rig SPSA process group,
preserve its final durable checkpoint/history with SHA256, and copy that
history and openings to forge with matching SHA256s. Interrupted uncheckpointed
iterations contribute no durable game count. Build/transfer/machine/checkpoint/
exact launch receipts are written before continuation. Never pool games from
the two machines into a strength verdict. Tuning data is optimizer history;
only converged-vector games on the rig after the endpoint can accept parameters.
No engine/net adoption, rig gate, Mac engine/build/test or external message.
