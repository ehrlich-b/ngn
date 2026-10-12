# Bounded dense training-pipeline profile

Session68 follows Astra section2/item6 and the October11 19:55 amendment:
GPU work is diagnostic only after data step2's null. Measure the existing
KB8/H512/B16384/BF16 pipeline before considering sparse features. No new
training treatment, checkpoint, exported net, selection or game nomination.

Rig12,14/GPU/nice19/4GiB/swap0, original session62-loader-index/source.
Use the completed epoch58 checkpoint only as an in-memory starting state;
discard all updates. Preserve every completed training artifact. Seed
2026101108, choose the first16 blocks of a seeded permutation of the
T80/T76/T74 blocks listed by the existing config source counts. T75 is
served by forge and is excluded to preserve its scored lane's load.
This profile cannot estimate T75 network latency. Read only those
blocks, compute the unchanged content holdout and canonical conversion,
and use the original Archive.batches and prefetched helpers on this
bounded view. No full corpus or holdout scan/hash refresh. This reproduces
the loader mechanism, not the exact epoch58 presentation sequence.

Freeze16 warmup,128 timed and8 operator-profile updates before execution.
Use the same model, AdamW/lr.0001/weight-decay.01, WDL.75/K400, BF16
autocast and high matmul precision, two CPU threads. Report end-to-end
timed throughput, CPU loader wait/pin/enqueue time, CUDA-event stage time
for transfer/decode/forward-loss/backward/optimizer-clipping, source counts
and GPU peak allocation. These times overlap and must not be summed as
wall-time percentages. CUDA events/profiler add overhead; this is a
bottleneck diagnostic, not an acceptance-grade speed comparison.

Separate8-step operator profile records shapes and device/CPU totals to
identify dense matrix multiplication versus decode kernels. No full
Chrome trace or checkpoint is needed. Native20-minute operational cap;
failure preserves stage/log, with no automatic retry. Concrete manifest
precedes reads; terminal profile.exit/done and summary.json remain at
session68-training-profile. Sparse implementation is only a proposal if
compute shares support it, and needs its own numerical/full-throughput
comparison and eventual net games. No new architecture/data-volume run.
