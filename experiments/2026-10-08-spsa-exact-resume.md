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

Pending verification. No accepted runtime, net or search vector changes.
