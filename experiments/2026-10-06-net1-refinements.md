# Net1 refinements — 2026-10-06

**Failed after c1 training; no gates and no accepted refinement. Retain net1.**
The recovered WSL runner started at 22:30:35 UTC from `11abce9` and completed
c1 before the next Windows disk check failed. PowerShell returned exit 1 with
`WSL UtilAcceptVsock: accept4 failed 110`; this was an interoperability failure,
not evidence that the disk cutoff had been reached. The runner exited without
starting c2, c3 or any gate. Its `current-job.json` remains stale at `train-c1`.
At the subsequent read-only audit C: had **157.501862 GiB free** and the runner
and trainer processes were absent. No restart or continuation was attempted.

| Trial | Lambda | Epochs | Best epoch | Native / .75 holdout loss | Result loss | Gate |
|---|---:|---:|---:|---:|---:|---|
| c1 | .75 | 80 | 75 | .007687067454 | .099955820116 | Not run |
| c2 | 1.0 | 32 planned | — | — | — | Not run |
| c3 | .9 | 32 planned | — | — | — | Not run |

c1 trained NGNN2 H256/NB8 from scratch on **36,785,161** frozen gen0/gen0b
training records, with **1,895,755** held-out records, batch 16384, BF16,
seed 20261006, AdamW LR .001, cosine decay to .01 of initial LR and two CPU
loader threads. All 80 epochs finished in **930.367018 seconds**. Best-epoch
training loss was **.006755488562**; epoch 80 holdout loss was
**.007689085584**. The checkpoint contains the complete history through its
best epoch, consistent with the separate 80-epoch history file.

The T80 task's WSL audit independently checked the raw `training-results.json`,
`history.json`, checkpoint epoch/model, recorded file digests and parity log,
then reexported the checkpoint and verified byte identity. The original engine
parity test passed **20/20 exact cases**, H256/NB8, AVX2. Best network SHA-256:
`870c67294d484dd6331ffaf278cc54072c1525b19766e931d50fb6021ee57aca`.
Best checkpoint SHA-256:
`a5ed306ef4853aca84130218f73638eafc783d1165731b837e4b5e689025bff8`.

Raw files: `/home/ehrli/ngn-data/net1-refinements-20261006/`; failed runner log:
the same path with `.runner.log`. The completed audit is
`/home/ehrli/ngn-data/t80-20261006/refinements-audit.json`. No
`gate-results.json`, `selection.json`, `completed.json` or `nets/net1b/`
exists. c1's smaller holdout loss is not a playing-strength result.

The [prepared runner](2026-10-06-net1-refinements.py) planned fixed 400-game,
200-pair gates at 10+0.1, concurrency 8, Hash64, Threads1 and no score
adjudication/stopping. Its selection rule was paired CI lower >0 or Elo >=+10,
zero anomalies, highest qualifying Elo. None of those gates ran.

Accepted baseline remains `/home/ehrli/ngn-data/nets/net1/net1.nnue`, SHA-256
`6a203ffe128541ef422f9344f2906bffbebea6d93931bb1f50e27f44bf2beff6`.
[Net1 orientation](2026-10-06-net1-gates.md): three-anchor pool
**2947.76 [2891.92, 3003.60]**, four-anchor pool
**2945.57 [2893.90, 2997.24]**. All NGN execution stayed on WSL.
