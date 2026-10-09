# NGNN4 batched affine training — session 16 topic candidate

Topic `night/ngnn4-batched`, baseline **9b6a3d6**. The architecture has not
trained a real net. Accepted runtime and the running factorizer are unchanged.

NGNN4 currently gathers `[L,2H]` W2 for every record, then performs per-record
BMM. At batch16384/H512/L16 that gathered FP32 tensor alone is **1 GiB**,
before its backward intermediates. Replace it with one GEMM over flattened
`[NB*L,2H]` weights and gather the selected material head from `[B,NB,L]`
activations. That activation tensor is **8 MiB** in FP32 for NB8/L16.
The calculation has the same real-valued affine function and gradients;
floating reductions may differ. Initialization, model shapes, parameters,
optimizer groups, checkpoint/export and NGNN1/2/3 forward paths are unchanged.
Do not transfer an in-progress NGNN4 training trajectory across this edit.
No third-party code or weights are introduced.

Before keeping the trainer edit on topic: forward and all parameter gradients
must agree with the old gathered path at NB1/2/4/8 and L16/32 under declared
FP32/BF16 tolerances, including shuffled heads, both STM orders, signed
upstream gradients and exactly zero gradients for unselected heads. AdamW
and clipping must yield finite tensors. Existing NGNN4 forward/export/resume
and the full CPU trainer suite must pass. CPU checks on 13,15 / nice19 wait
natively for affine speed controller **630484**, preserving its fixed sample.

The actual CUDA tests and fixed full-batch benchmark wait natively for CPU
completion and factorizer controller **630389**, with a bounded 12-hour wait.
They run on CPU13,15 / nice19 after factorizer's normal gate window; no new
real trainer follows. Run actual FP32/BF16 forward/backward/fused-AdamW
controls with no skips, then compare the old and new **complete model
forward+backward** at batch16384/NB8, H256/512 × L16/32, BF16, identical
synthetic sparse features/STM/buckets and model state. Warm both paths.
For each architecture use **20 fixed AB/BA blocks**, five steps per side,
CUDA events and synchronization. Clear gradients each step; no optimizer
update during timing. Report geometric old/new throughput ratio, 95% t
interval on log ratios (df19), both order strata and peak allocated memory.
Keep only if all four architectures have lower ratio >1, both strata >1
and lower peak memory; otherwise shelve without extending the sample.

This is a training-cost verdict for an unaccepted architecture, no Elo or
exact floating trajectory claim. A future architecture trial must freeze
its chosen trainer and still train the full owned recipe, verify integer
parity/reexport and pass fresh A/A40/gate400 against the immediate base.
Results: box `session16c/{cpu*,cuda*,benchmark.json}` and campaign status
stage `ngnn4-batched`. Preserve this queued short CUDA window before the
next real GPU trainer, datagen or claim. No game is launched here.

## Corrected CPU setup — session 17

Original corrected-patch root `session16c/` completed CPU with exit1 at
**21:09:19 EDT**. New gradient/NGNN4 checks passed; the sole full-suite
error was a source-relative NGNN3 fixture lookup from the controller root.
The fixture was present in `source/testdata/ngnn3/`. Its CUDA waiter ended
before any GPU tests or timing. Preserve all original logs; no cost verdict.

Fresh `session17b/` uses the **same base and candidate patch**, with CPU
unittests run from `source/`. No model/test/tolerance/sample change. CPU
**641098** and CUDA **641099** launched **21:22:01 EDT**; CPU completed
**21:23:32 EDT**, exit0, all65 tests pass (6 intentional CUDA skips), full
suite76.051s. The two local candidate files equal the CPU-tested source.
[CPU result and launch](2026-10-08-session17-results/batched-cpu.json).

CUDA641099 waits natively for factorizer630389, timeout **09:22 EDT October9**.
Actual CUDA tests and the unchanged four-architecture20-block fixed cost
rule remain pending; no topic commit or real architecture training. Results
`session17b/{cuda*,benchmark.json}`. Preserve this window, then the queued
sparse-transformer CUDA window, before any new trainer or exclusive claim.
[Corrected controller](2026-10-08-ngnn4-batched-recheck.sh),
[same benchmark with fresh output root](2026-10-08-ngnn4-batched-recheck-speed.py).
