# NGNN4 VNNI affine — prospective session31 speed candidate

Session31 starts **18:15:10 EDT October9**, deadline **19:55:10**.
Verified four-row architecture base **aafa7fd**, isolated topic
`night/ngnn4-vnni`, worktree `output/campaign-ngnn4-vnni/`.
One change: replace three AVX2 multiply/reduce/add instructions per row
with one EVEX256 unsigned-byte/signed-byte dot accumulation when hardware
and OS state support it. Four contiguous rows, input packing, horizontal
reduction, serialized weights, architecture, trainer and search stay identical.
The original AVX2 routine remains the fallback; purego/race remain portable.

The implementation is original NGN code, written from the arithmetic and
[Intel instruction specification](https://www.intel.com/content/www/en/us/architecture-and-technology/64-ia-32-architectures-software-developer-manuals.html).
No chess-engine code or foreign weights were opened or copied. Go assembler
syntax was inspected on the rig; an initial explicit K0 operand was rejected
at compile time and corrected to the assembler's implicit unmasked form.
The narrow all-width guarded differential, hardware/OS admission and forced
AVX2 fallback sweeps then passed on WSL. No timing preceded nomination.

Runtime admission requires CPUID leaf7 AVX512F/VL/VNNI, OSXSAVE and XCR0
bits1,2,5,6,7; XGETBV is called only after OSXSAVE is established. Tests
reject each missing feature/state bit. Unsigned activations0..255 and
signed weights−64..64 imply the complete absolute dot at maximum input
width4096 is66,846,720; every lane, partial sum and horizontal sum fits
int32 exactly. There is no saturation or arithmetic tolerance.

## Frozen checks and decision

[Controller](2026-10-09-ngnn4-vnni-controller.sh),
[checker](2026-10-09-ngnn4-vnni-check.py),
[source patch](2026-10-09-ngnn4-vnni.patch).
Box root `/home/ehrli/ngn-data/night-20261007/session31-vnni/`.
Archive **aafa7fd**, patch, script, checker, fixtures and accepted net are
pinned before launch; built binaries/source/profile/Go appear in the receipt.
Ryzen9800X3D/WSL2, Go1.25.5/GOAMD64v3, PGO616b1ebe, nice19.
Tests/builds taskset12,14; search/timing14, GOMAXPROCS1. C:>=32GiB admission.
SEE13,15 and SPSA0–11 keep their allocations; hopper remains untouched.

Require actual VNNI execution with no skip, independent int64 all-width
misaligned/guarded controls, forced AVX2 fallback, purego controls,320 exact
trainer cases, full short/race/vet and PGO builds before search measurement.
**48 exact identities**: NGNN4 H512/L16 depth12, NGNN3 depth14,
NGNN2 depth14 and HCE depth11, twelve positions each.

Warm both; **exactly20 balanced AB/BA blocks**, six positions×3s per
binary. Keep on its architecture topic only if geometric NPS ratio's95%
Student-t interval(df19) has **lower>1 and both order strata>1**.
Otherwise shelve the complete sample and restore only this topic's patch;
no extension or retry. Incomplete evidence has no verdict.

Only speed keep reopens architecture cost with **exactly12 balanced3s
blocks** comparing NGNN4/NGNN3 at the same H512. Point ratio>=0.85 admits
later narrow training after higher-priority data; print df11 interval and
both order strata. Owned random NGNN4 fixture67043325 versus trained
NGNN3be34532c has different trees: budget evidence only, no Elo verdict.
Prior four-row cost0.81555 and failed eight-row sample remain intact.
No accepted playing-runtime adoption, trainer or game follows automatically.

Launch detached with setsid/nohup; exact PID and estimated completion go
in the session record. Terminal `controller.{exit,done}`, raw timing and
optional architecture-cost receipts trigger the next review.
