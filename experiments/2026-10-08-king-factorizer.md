# King-bucket factorizer preparation — campaign session 7

Session starts **13:41:34 EDT**, deadline **15:21:34 EDT**. Optional trainer
capability only; no new strength candidate or GPU training job is admitted.
The longer-H512 controller and queued exclusive v3 admission keep their
existing source, recipe and frozen artifacts. The pending move-delta probe
owns its fixed timing sample and tuner restoration.

## Mechanism

NGNN3 currently learns eight independent 768-row input transformers. Add
`--king-factorizer`: one trainable shared [768,H] transformer plus the existing
bucket-specific [KB*768,H] residuals. For each bucket the effective weights
are the float32 sum of its residual and the shared rows. Forward uses that
sum, so gradients from every visited king context update the shared rows.
AdamW decays both components. This is NGN's own implementation of the general
shared-plus-residual idea; no external source, network or trainer is used.

Shared rows initialize to zero without consuming RNG. Initial effective
weights, output weights and the RNG stream equal the existing recipe. The
disabled path retains the original parameter state and arithmetic. This
training parameterization changes the optimization; it does not itself
establish a loss or playing-strength improvement.

Fold in float32 before the existing single quantization operation, preserving
forward's addition and avoiding separately rounded components. The export
is an ordinary NGNN3 file, with unchanged inference/storage requirements.
Clipping bounds the effective sum and the shared component; the residual
is bounded relative to the shared component. Checkpoints retain both tensors
and bind the factorizer flag; legacy missing flags mean false. Enable it only
with NGNN3 and at least two king buckets.

## Prospective preparation checks

Before execution: isolated `night-20261007/session8/source/`, base archive
**99b5dd7**, exact overlay/controller/source hashes. Existing Python 3.12,
NumPy/PyTorch; CPUs **13,15**, nice **19**, GPU excluded. Require:

- Zero-initialized factorizer preserves baseline export bytes and RNG.
- Nonzero shared rows equal a separately folded model's forward, residual
  gradients and summed shared gradients across four/eight input buckets,
  one/eight output buckets, both perspectives and all king squares.
- Merge precedes rounding; folded exports are byte-identical and independently
  evaluated. Extreme shared/residual rows clip to both int16 endpoints;
  malformed shapes, wrong formats and nonfinite sums fail.
- Two-epoch synthetic CPU fit equals one epoch plus resume for every model
  tensor and export. Shared rows actually update. Reexport is byte-identical;
  changed flags fail in both directions, and legacy checkpoints still resume.
- The complete trainer suite passes with existing fixture reproduction and
  default/archive/NGNP split behavior. GPU-only tests are explicitly skipped.
- An H16 factorized fixture folds/export identically and agrees exactly with
  NGN's existing Go integer parity control on **64 cases**. Reuse the frozen
  baseline test executable from the pending speed controller; do not build,
  replace or run a timing/game binary.

The [controller](2026-10-08-king-factorizer-tests.sh) writes test/parity/source
receipts and a terminal exit/status event. Reconcile all trainer/fixture
inputs against local bytes before committing this optional capability.
No required Go source checks are repeated because Go/assembly are unchanged.

## Subsequent experiment boundary

Before any real factorizer training trial, run its CUDA FP32/BF16 forward,
gradient, fused AdamW and export smoke after the existing exclusive admission
window, then freeze the immediately accepted net's recipe. Add only this flag
to the same recipe, data, seed and schedule, with exact trainer/engine parity,
fresh A/A40 and fixed paired gate400 at 10+0.1. Accept only penta lower 95%
bound >0 and zero flags/crashes/illegal/no-move results; otherwise shelve
without extension. No candidate or recipe is queued by this preparation.

## Results

Pending preparation checks; no strength verdict.
