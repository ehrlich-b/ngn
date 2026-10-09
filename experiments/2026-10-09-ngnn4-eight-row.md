# NGNN4 eight-row sharing — prospective pure-speed candidate

Prepared session30, October9, from **aafa7fd** on
`night/ngnn4-eight-row`, worktree `output/campaign-ngnn4-eight-row/`.
The verified four-row affine kernel remains the baseline. One change in
`engine/ngnn4_kernels_amd64.s`: eight output rows share each32-byte input
load, using eight independent AVX2 accumulators and two four-row horizontal
reductions. The upper four rows use a fixed four-row byte displacement.
All rows, input clipping, quantization, file layout, output arithmetic,
trainer, architecture, search and time management retain baseline behavior.

This is original NGN implementation; no third-party code or weights opened.
The [-64,64] weights and [0,255] inputs bound every paired int16 product
and full int32 dot exactly as in the verified four-row implementation.
Supported L16/32/48/64 are divisible by8; H16..2048 are divisible by16.
The existing deliberately misaligned/guarded independent-int64 controls
cover signed extrema, cancellation, row order and destination/input guards.
No numerical tolerance or new behavior is introduced.

## Fixed verification and verdict

[Controller](2026-10-09-ngnn4-eight-row-controller.sh),
[fixed sample](2026-10-09-ngnn4-eight-row-check.py),
[one-file patch](2026-10-09-ngnn4-eight-row.patch).
Box directory `/home/ehrli/ngn-data/night-20261007/session30-eight-row/`.
Natively follows the completed session25 TT-scan lane, then C:>=32GiB.
Tests/builds taskset12,14; identities/timing14, nice19, GOMAXPROCS1,
Go1.25.5/GOAMD64=v3, original616b1ebe PGO, Ryzen9800X3D/WSL2.
SPSA0–11 and SEE games13,15 continue. Detached setsid/nohup, append-only
status.jsonl and `controller.{exit,done}`; estimated30–40min from admission.

Before timing: baseline NGNN4 controls, candidate AVX2/purego guarded
all-width controls,320 integer cases, full short/race/vet and PGO builds.
Pin archive/patch/controller/check/net/reference fixtures before launch,
then record exact built binaries/source/Go/profile in `build-receipt.json`.
**48 exact search identities**: twelve each NGNN4 H512/L16 depth12,
NGNN3 H512 depth14, NGNN2 depth14, HCE depth11.

Warm both, then **exactly20 balanced AB/BA blocks**, six positions ×3s per
binary. Keep only if geometric aggregate candidate/base NPS ratio's95%
Student-t interval (df19) has **lower>1 and both order strata>1**; otherwise
shelve without extension. Incomplete evidence has no verdict. Preserve all
raw records. Source remains uncommitted pending this verdict; no automatic
accepted-runtime integration.

Only on speed keep, the new mechanism reopens the architecture budget:
same candidate binary, same H512, owned random67043325 NGNN4 H512/L16
fixture versus acceptedbe34532c NGNN3, **exactly12 balanced3s blocks**
(df11 interval). Point ratio>=0.85 admits later narrow training after the
higher-priority data lane. Print both strata and interval. This comparison
has different search trees and is budget evidence only. The previous four-row
budget0.81555 and all original raw evidence remain preserved; no extension
of that experiment, trained net, recipe change or GPU trial is implied.

Review receipts/raw source once when terminal. On speed failure restore
only this one-file patch on its own topic. No retry or larger timing sample.
