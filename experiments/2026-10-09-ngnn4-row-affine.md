# NGNN4 contiguous-row affine — session22 candidate

Session starts **10:35:31 EDT October9**, deadline **12:15:31 EDT**.
The merged plan still ranks provenance and broader data first. Host C: is
10.74 GiB free, below the25GiB floor; no new heavy work starts. Existing
session21 controllers hold their lanes, SPSA continues on0–11, hopper stays
untouched, and the GPU is idle. This is original inference code preparation
to reopen the NGNN4 budget lane, previously deferred at0.845141 NPS ratio.

Topic `night/ngnn4-row-affine`, `output/campaign-ngnn4-row-affine/`, starts
from accepted runtime **39eb128/be34532c**. Overlay only the engine/trainer/
fixture changes from the retained NGNN4 lineage **5b882e2..14a203a**; preserve
the current TT, board representation and trainer fixes. This forms the frozen
architecture baseline. No foreign engine source, code or weights are opened.
The source remains uncommitted pending the speed verdict, and main receives
only this manifest, the candidate patch and the controller.

One change: use natural serialized W2 rows for the affine dot instead of the
loader's four-input/eight-output layout. Four output rows share each32-byte
input vector, accumulate independently, and reduce their eight int32 lanes
at the end. Each adjacent byte-pair product is bounded by32640 and the full
absolute dot by66,846,720, so neither saturation nor reassociation changes
any legal result. Keep the old packed kernel as a differential reference;
the file format, trainer arithmetic, input clamp, output kernel, dimensions,
search, timing and network fixtures are unchanged.

## Predeclared verification and speed verdict

All computation on WSL, nice19, Go/GOAMD64=v3, original accepted PGO616b1ebe.
Native wait for session21's SEE/pawn **checks.done**, then require C:>=32GiB.
Build/test CPUs12,14; fixed sample CPU14, GOMAXPROCS1. This preserves the
SPSA0–11 and provenance/data13,15 lanes. Failed SEE/pawn checks release the
CPU lane but do not imply that either candidate is accepted.

- Guarded and deliberately misaligned row/portable/independent int64
  comparisons for every H16..2048 in steps16 and L16/32/48/64. Include signed
  maximum dots, cancellation, row order and immutable inputs. Existing packed,
  input/output kernel and320-case H512/L16 integer controls must still pass.
- Candidate AVX2/purego and full short/race/vet pass; baseline NGNN4 controls
  pass after overlay onto39eb128. PGO-build separate speedprobe binaries.
- **48 exact search identities:** twelve each NGNN4 H512/L16 depth12, accepted
  NGNN3 H512 depth14, embedded NGNN2 depth14, and HCE depth11.
- Warm both binaries, then **exactly20 balanced AB/BA blocks**, six positions
  ×3s per binary. Keep on topic only if geometric aggregate NPS ratio's
  two-sided95% t interval (df19) has lower>1 and both order strata>1.
  Otherwise shelve without extension. Microbenchmarks describe cost only.
- Only on a speed keep, run a fresh architecture budget comparison: same
  candidate binary and H512, owned random NGNN4 H512/L16 fixture versus
  accepted NGNN3/be34532c. Warm both and take **exactly12 balanced AB/BA
  blocks**, six positions ×3s per net, CPU14. Print ratio, df11 interval and
  both strata; point ratio>=0.85 admits a later narrow recipe. Tree mix differs
  between these nets; this is budget evidence, not strength or Elo.

The new kernel is the explicit reopen condition for the old architecture
cost deferral. No trained net, GPU trial, games, accepted-runtime merge,
net/default promotion or claim follows automatically. Any future architecture
trial still follows the data lane and needs its own actual-base manifestation,
parity/reexport, fresh A/A40 and standard lower95>0 gate400.

## Frozen inputs and completion signal

Box directory `/home/ehrli/ngn-data/night-20261007/session22-row/`.
Baseline archive SHA256
`9977b61113e544d04060e4d081a776d7135a104b8ec64d6044c3d334bc589a60`.
Candidate patch SHA256
`e38cdbc5f3e2cf23056b2857ec0b89b3b219afe989baeca0c900467e419efacb`.
At launch, copy the existing owned67043325 random fixture and accepted
be34532c net into this directory; pin them and the exact controller/check
alongside the archive/patch in `frozen-inputs.sha256`. Record built binaries,
source files, PGO and Go version in `build-receipt.json` before searches.

[Controller](2026-10-09-ngnn4-row-controller.sh),
[fixed sample](2026-10-09-ngnn4-row-check.py),
[candidate patch](2026-10-09-ngnn4-row-affine.patch).
Detached launch uses `setsid nohup ... </dev/null >controller.log 2>&1 &`.
Completion is `controller.done/exit` plus status.jsonl stage
`ngnn4-row-affine`. Estimated runtime after admission30–45min; no polling
while held. Review `build-receipt.json`, `timing/receipt.json` and, if the speed
rule passed, `architecture-cost.json` once at completion. Preserve failed
attempts; a capacity-wait timeout is no engine failure and permits no restart.

Local checks are shell/Python syntax and patch application only. Builds,
engine tests, searches and timing remain pending on the rig.
