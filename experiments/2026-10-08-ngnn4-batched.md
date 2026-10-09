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
