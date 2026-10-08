# SPSA exact checkpoint continuation — campaign session 12

Session starts **15:31:02 EDT**, deadline **17:11:02 EDT, October 8**.
Accepted engine/net remain **5b882e2 / be34532c**. The live tuner stays on its
frozen driver; neither its checkpoints nor its executables are migrated.

The session-11 source audit proves that the old driver resumes theta/iteration
but restarts its seeded perturbation/opening stream. It also accepts changed
recipes and skips malformed checkpoint rows. This is a tooling correctness
repair for a future, separately declared tune, with no strength claim.

Version-2 checkpoints bind the exact engine and tuner executable SHA-256s,
effective ordered opening list, Go/platform/GOMAXPROCS, seed/RNG algorithm,
iteration budget, concurrency/pair mode, clock/move cap, learning schedule,
report cadence, power mode and ordered parameter registry/schedule constants.
The wrapper counts underlying Int63 calls, including rejection sampling;
resume replays the completed iterations' draws and checks the recorded count.
The fresh seeded Intn stream and optimizer update rule remain identical.

All rows must have the same recipe, complete bounded theta/rounded maps,
valid cadence/time/health and matching draw counts. Reject missing, empty,
legacy, mixed, malformed or incomplete history; never fall back silently.
A fresh run refuses a nonempty history. Writes propagate errors, sync the
complete JSONL row, and stop the campaign on failure. Completed histories
return the final vector without starting engines. Resume restores optimizer
inputs given the same outcomes; real-clock results and engine state are not
an uninterrupted replay.

Predeclared verification, all on WSL CPUs **13,15**, nice **19**:

- Original/new seeded Intn streams under three seeds and ordinary/rejecting bounds.
- Six full/interrupted trajectory comparisons: K=1/6, openings=0/5000/2^30+1,
  250 iterations, split at 75, identical inputs and float theta at every report.
- Recipe changes, corrupt history and write-failure controls; failure leaves theta unchanged.
- Narrow SPSA tests and race tests, engine short/race, full short/race and vet.
- An isolated future tuner build and actual CLI refusal checks using a shell
  marker as the engine; rejection must occur before that marker can execute.
- Source/artifact hashes and unchanged live training/tuning/admission/CUDA inputs.

No matches, timing experiment, trainer, datagen, screen or claim is launched.
The tests use synthetic scores. The built tuner is a future artifact only;
do not replace `/spsa/spsa` or the live resume command. Legacy campaigns continue
using their existing frozen binary; a future tune starts a new output file and
needs its own base/recipe manifest, >=20k games and converged-vector game gate.

Controller: [2026-10-08-spsa-exact-tests.sh](2026-10-08-spsa-exact-tests.sh).
Isolated box root `/home/ehrli/ngn-data/night-20261007/session12/`.
Inputs are the current source archive plus only the three SPSA candidate files.
The controller formats and freezes the extracted source, validates it, and
records individual check exits and hashes. Preserve failures, no automatic retry.
Launch detached once; record its PID/start event and use a native completion
notification for the short verification job. Existing hour-scale jobs are not polled.

## Result

Manifest/controller **1d93cfd** was committed and pushed before execution.
Detached controller **587761** started **15:38:06 EDT**, completed
**15:44:50 EDT**, exit zero. All eight checks pass: narrow SPSA/race,
engine short/race, full short/race, vet and isolated future-tool build.
All **65** test/subtest pass rows reconcile: six exact full/interrupted
trajectories, 28 recipe-mismatch and 24 corrupt-history subcases, plus RNG,
write/stop and artifact controls. All **11** actual CLI refusals exit before
executing the shell engine marker. Tests and builds ran exclusively on WSL.

Future tuner SHA-256 **944a30a3**, retained in `session12/future-spsa`;
it is not installed. All **2,040** extracted input hashes reconcile after
verification. The three candidate Go files match the Mac working copy exactly.
Mac tar added three `._*` metadata files; Go ignores them and they are excluded
from the committed patch. No external source or network was added.
Final review also verifies the unchanged live tuner/engine binaries and all
**14** training, **3** admission and **27** CUDA dependency input pins.

[Check receipt](2026-10-08-session12-results/test-receipt.json),
[final raw-log/hash review](2026-10-08-session12-results/final-review.json),
[CLI refusals](2026-10-08-session12-results/cli-receipt.json),
[launch custody](2026-10-08-session12-results/launch-observed.json),
[build settings](2026-10-08-session12-results/buildinfo.txt).
Full logs, candidate source and hashes remain in box `session12/`.

No accepted playing runtime, net, tune or strength result changes.
The current frozen tuner still restarts its RNG on resume; do not replace it
or feed its legacy JSONL to the new driver. Future tunes must preserve the
exact built tuner/engine and use a fresh output path and prospective recipe.
The output parent directory must already exist. Resume restores optimizer
inputs from a complete checkpoint, not the uncheckpointed games since then.
A partial/corrupt history is rejected and preserved; diagnose before any
separately declared restart. No new tune or long job is queued.

Final [box snapshot](2026-10-08-session12-results/box-state.json), **15:46 EDT**:
longer H512 **62/116** epochs (last **295.41 s**), GPU **61% / 4,197 MiB**;
SPSA durable **400 / 4,800 games**, saved **15:31:02 EDT**.
Trainer **541007**, tuner **577984**, dependency controllers
**522940 / 552631 / 577875** and hopper **1168** remain live and untouched.
Host C: free **146.5 GiB**. No completed claim meets 3600.
