# Data step 2: frozen source and quantization diagnostic

Session66 follows Astra's training-pipeline diagnostic after full58 and
selected reexport/parity. Read-only diagnostic; no model, training recipe,
checkpoint selection or playing adoption changes. Rig12,14/GPU/nice19,
MemoryMax4GiB/swap0; separate correctness checks13,15/2GiB and games0–11/
3GiB keep total admitted NGN memory9GiB. C: remains above25GiB. Forge
readers serve the existing immutable archives, with no new CPU game job.

Freeze before reading: seed2026101106,16 randomly selected non-repeated
262144-record blocks per source, source order from trained/config.json's
archive_split. Identify holdout by the original occupied-board hash,
seed20261006/modulus1024. Select2048 holdout rows without replacement
per source using the frozen seed; save the8192-row NGNP panel and row
metadata before any evaluation. This is source-stratified, not an
unbiased pooled validation estimate. Report material/king-bucket strata
and their sample sizes; no game-disjoint or mirror-disjoint claim.

Count train/holdout, material buckets and both perspective king buckets
in all scanned blocks. Corpus record shares are exact; sampled exposure
histograms and volume-based expected58-epoch presentations are estimates,
not a replay of the actual14.848B training presentations. The completed
trainer did not retain per-source/per-bucket presentation counters.
Do not scan the whole corpus or replay training to manufacture them.

Evaluate the selected58 checkpoint in FP32 and the deployed integer net
`4636de46` on this identical panel, plus the accepted integer `a4d90b41`.
Report signed/absolute/95th-percentile/max integer-minus-float CP residual,
WDL.75/K400 blended losses, result-only loss, by source/material/king
bucket. Compare against BF16 autocast as a numerical diagnostic only;
training validation selected on FP32, so FP32 is the relevant comparator.
No significance/Elo claim from these losses. Report a source weighting or
quantization proposal only if the measured diagnostic supports it; any
future intervention needs a separate prospective nomination and gate.

Native30-minute operational cap; failure preserves panel/stage/log,
with no automatic retry, training or games. Lane session66-data-diagnostic,
terminal diagnostic.exit/done and summary.json; selected artifacts stay
on the box and only small receipts/report return to the repository.
