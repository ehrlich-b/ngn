# NGNN3 fake quantization arithmetic control

`NGNN(..., fake_quantize=True)` or `trainer.train --format 3 --fake-quantize`
opts into quantization-aware forward arithmetic. The default stays false;
`model.forward_float(...)` retains the original float expression. The trainer
saves the flag in its configuration and rejects a changed flag on resume;
legacy checkpoints default to false. There is no format or engine change.

The opt-in path merges shared/residual rows in float32, widens parameters to
float64 before multiplying by QA/QB, then rounds to nearest with ties to even.
This order matches export, even for float32 values adjacent to half steps.
It checks the same post-round int16/int32 limits and output-weight tolerance
as export; the existing `clip_quantized` parameter bounds are unchanged.
Integer-valued float64 intermediates preserve the bounded NGNN3 integer sums.
SCReLU clips at 0/QA before squaring; both divisions truncate toward zero in
their original order, and the final NGNN3 score clamps at +/-24999. Round and
truncation use identity straight-through derivatives; clipping uses ordinary
clamp derivatives. Mixed precision is disabled inside this path. Float64 is
an arithmetic correctness choice, not a training throughput claim.

From the repository root on the rig, with the existing interpreter and an
external writable output directory:

```bash
TMPDIR=/absolute/scratch experiments/wave2-1104/qat-model/run.sh /absolute/evidence
```

The executable runs exactly
`~/ngn-original-nnue-night-20261001/venv/bin/python -m unittest trainer.tests.test_quantized_forward -v`
with no GPU, two CPU cores, two tensor threads, stripped credential variables,
and a 10 CPU-second process limit. There is one QAT forward/backward batch,
plus the required float/off and scalar quantization controls. It reads no
corpus and runs no training loop. `result.json` and `cost.json` are frozen
actual outputs, copied from the external evidence directory. The command
checks stable results against `result.json`; durations and package versions
are recorded observations. `check_candidate.py` separately checks ancestry,
author/committer custody, committed artifacts and a clean checkout.

`fixture.py` constructs an owned H32 KB8/B8 factorized model. `cases.json`
contains 32 synthetic FENs, both STM values, mirrored/unmirrored kings, all
eight output buckets and their material boundaries, and independent scalar
arithmetic. The first/second-floor substitutions change two/five cases.
Half steps at QA/QB, their adjacent float32 values, int16/int32 ranges,
activation boundaries (-1/0/1 and 254/255/256), output-weight limits and the
final cp clamp are covered. The shared/residual channel rounds to one only
after merging. The export is 394436 bytes and is created only in temporary
scratch. Every parameter group has finite, nonzero interior gradients;
saturated activations/output rows have zero gradients. Off matches the
pre-QAT float expression and every parameter gradient bit for bit.

The frozen run used Python 3.12.3, torch 2.7.0+cu128 and NumPy 2.5.3 on CPU.
Its complete fragment took about one second, below one CPU-second; the one
QAT forward/backward sample was about 0.08 seconds. These are bounded plumbing
diagnostics, not strength or general performance results. H768 training,
corpus evaluation, GPU execution and changes to default play await separate
completed rig gates. This fixture establishes no Elo claim.
