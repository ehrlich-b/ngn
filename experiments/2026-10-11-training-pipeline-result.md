# Dense profile and sparse primitive result

The session68 bounded dense profile completed20:09:41/exit0 on rig12,14/
GPU/4GiB/swap0/nice19. Original trainer source and all completed training
artifacts remain unchanged. Sixteen seeded T80/T76/T74 blocks supply the
bounded loader view; T75 network latency is excluded to preserve forge's
game-lane load. No full corpus scan or checkpoint/export was performed.

128 instrumented updates process2,097,152 rows in2.01164s,1.043M rows/s.
CUDA-event totals: transfer8.04ms,decode241.57ms,forward/loss574.53ms,
backward833.30ms,optimizer/clipping21.84ms. CPU loader wait0.883s,
pinning0.248s and enqueue0.867s overlap GPU work and cannot be summed
as wall-time fractions. Peak GPU allocation1,706,304,000bytes; dense
feature storage805,306,368bytes(768MiB). Separate8-step operator profile
identifies FT forward/backward GEMMs, dense fill/cast and indexed output
gradient accumulation as substantial costs. Event/profiler overhead,
bounded blocks and missing T75 latency prevent a full-training speed claim.
[Raw profile](2026-10-11-session68-results/profile/summary.json).

The first attempt failed before updates because remote profile.py shadowed
Python's standard profile module during optimizer initialization. Its exit1,
traceback and manifests are preserved in module-shadow-failure. Renaming
the remote entrypoint to pipeline.py manually reruns the identical bounded
diagnostic and completes0; the failed attempt is never relabeled.

One original embedding_bag sparse accumulator preflight followed the
prospective rule, using the existing8192 frozen rows repeated to B16384.
Exact feature reconstruction and label/material geometry pass on the full
panel. BF16 forward output/loss matches exactly there, but w1 gradient
max residual is1.90735e−6. High/TF32-eligible FP32 differs by0.10485cp
MAE/0.81104cp maximum. These are diagnostic numerical limits, not proof
of corpus-wide equivalent training trajectories or integer quality.

The fixed12 alternating64-update blocks complete20:13:23/exit0.
Sparse/dense complete-update throughput ratio is **0.57755
[0.57336,0.58177]**, about42.2% slower. **SHELVED** by the predeclared
point>=1.20/lower95>1 filter; no extension, full trainer integration,
new training run, net export or game nomination. No Elo implication.
Reopen only for a different original accumulation/backward mechanism,
with its own numerical and full-throughput evidence. The built-in primitive
does not justify replacing the dense trainer on this GPU.
[Raw blocks and result](2026-10-11-session68-results/sparse/summary.json).
