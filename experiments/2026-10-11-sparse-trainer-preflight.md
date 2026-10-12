# Sparse accumulator preflight after the dense profile

The session68 bounded dense profile identifies768MiB dense input storage
and substantial forward/backward GEMMs. This nominates one diagnostic
prototype, not a new trainer or net: independently decode occupied feature
indexes and sum weight rows with PyTorch embedding_bag. No engine or
third-party trainer code, architecture or recipe change enters NGN.

Use the already-frozen8192-row session66 panel, repeated once to B16384,
and epoch58 checkpoint in disposable in-memory copies. Rig12,14/GPU/
nice19/4GiB/swap0; no corpus reads or forge reader load. Forward geometry
must reconstruct the original dense inputs exactly on all panel rows.
Report FP32 and actual BF16 output/loss/gradient residuals, not a claim
of numerical equivalence. The sparse sum casts FT weights to BF16 under
autocast; changed reduction/gradient order is a recipe concern even if
residuals are small. Ordinary dense output heads are preserved.

Freeze8 warmup steps per side, then12 alternating64-update blocks,
starting both models from the identical checkpoint/optimizer settings.
Measure complete decode+forward+loss+backward+AdamW+clipping time with
synchronization only at block boundaries. Report paired log-ratio95% CI
and GPU peak allocated bytes. Timing proposal passes only if point gain
is at least20% and lower95>0; failure ends this particular primitive
proposal without more blocks. This is a fixed-panel diagnostic filter,
not a complete trainer-speed or quality gate. No checkpoint/export is
written and no full training or games starts automatically.

Any surviving proposal still needs actual full-pipeline timing including
source I/O, numerical/full trainer checks, and a separately nominated
trained-net comparison before adoption. A failed primitive may reopen
only with a different original kernel/mechanism, not added samples.
Native20-minute operational cap; preserve failures. Results stay in
session68-sparse-preflight; only small receipts return to experiments/.
