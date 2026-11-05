# Trainable NGNN6 residual head

`trainer.residual_model.ResidualNGNN.from_ngnn3` starts from the deterministic
passed zero lift. H32 and H768 retain the original transformer, king map and
linear head byte for byte. The shared bottleneck has 16 units; eight projection
rows and eight biases start at zero. The accepted H768 head uses some linear
weights outside the older float trainer's ±1.98 bound, so NGNN6 retains the full
specified int16 range. Float64 head biases retain every int32 source bias.

Forward executes the exact wire arithmetic in float64, which can represent all
bounded integer intermediates exactly. Parameters round to nearest with ties to
even; divisions truncate toward zero. The residual takes unsquared clipped
activations. Branches truncate independently, add, then clamp. Backward uses
identity derivatives for rounding and truncation and normal clipping derivatives.
This is a surrogate gradient, not a derivative of the discrete evaluator.

Four explicit AdamW groups contain inherited weights, inherited biases, residual
weights and residual biases. Bias groups have zero decay. `clip_quantized`
respects the full inherited ranges and constrains residual biases using the
quantized row's complete input-box bound. Export checks dimensions, fixed scales,
finite values, integer representations and both affine ranges before writing
NGNN6 with the specified CRC. Existing NGNN1/2/3 export behavior is preserved.

The owned `records.ngnp` subset has 16 records: eight material buckets for White,
then those buckets for Black. `weights.npz` holds all trained parameters and
buffers with no pickle dependency. `trained-h32.ngn6` is their exact export.
`expected.json` holds hashes, exact integer accumulators and scores from the
independent NumPy int64 evaluator, training settings, losses and gradient sums.

The fixture uses exactly two batches of eight records, CPU threads=2, seed
20261105 and AdamW lr=0.02 without weight decay. Its synthetic target is initial
cp + 200, using MSE on cp/400. Inherited groups use lr=0 to isolate the new head;
they remain in the optimizer. The first step has nonzero R gradients and zero
A/HB gradients. After R changes, the second step has nonzero A/HB gradients.
Every bucket/turn case has a nonzero integer residual. This diagnostic establishes
no loss generalization, speed or Elo claim and does not change default play.

From the repository root, the bounded runner executes these exact fragments:

```sh
python3 experiments/wave2-1104/residual-trainer/run.py host
python3 experiments/wave2-1104/residual-trainer/run.py python
python3 experiments/wave2-1104/residual-trainer/run.py parity
python3 experiments/wave2-1104/residual-trainer/run.py packet
```

`python` runs `/work/cache/ngn-nnue-training/venv/bin/python -m unittest
trainer.tests.test_residual_model`. `parity` then runs `go test -p 1 -count=1
./engine -run '^TestWave2ResidualParity' -timeout 15s -v`. Both execute at nice 10,
with two-CPU affinity, GOMAXPROCS=2, tensor/BLAS threads=2, credential variables
removed, GPU visibility disabled and shared language caches. `result.txt` is
captured output of those two commands in that order. Go verifies the trained
fixture through compact and wide production format-6 evaluators and also runs
the inherited conversion parity cases. The inherited optional H768 search test
skips without an explicit external converted model; this task checks actual H768
warm-start bytes and all sixteen integer scores in Python, and adds no search run.

The existing Torch venv is a CUDA-enabled build, but every model and tensor here
uses CPU. NumPy 2.2.6 was missing from that Python 3.11 environment; it was installed
as an ordinary dependency into `/work/cache/ngn-nnue-training/numpy-cp311` without
modifying the venv. The runner adds that location to PYTHONPATH (override with
`NGN_RESIDUAL_NUMPY`). No model, GPU package or engine download is needed.

`packet` deterministically retrains in memory and verifies the committed files.
Explicit regeneration is `generate.py --write` under the same environment.
The public trainer also supports `--format 6 --residual-source OWNED.ngn3
--hidden 768 --buckets 8 --precision fp32 --device cpu --threads 2`; it uses the
existing data, checkpoint, resume and export paths, pins the source hash in the
checkpoint, and enables all four groups at the requested learning rate. Source
width must match `--hidden`; factorizer and map overrides are rejected. Set
`--batch 8 --positions-per-epoch 16 --epochs 1` for a two-update diagnostic.
