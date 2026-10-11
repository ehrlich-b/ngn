# Frozen data step2 diagnostic result

Full58 selected epoch58/net4636de46; read-only rig12,14/GPU diagnostic
completed16:52:10EDT/exit0. Manifest preceded reads; panel preceded evals.
Seed2026101106,16 random blocks/source, original occupied-board holdout
partition,2048 rows/source,8192 total; panel SHA256
cf2e8935652a32974924d5413fa9b6ec896d21dc781f524f8b12465794fa2bdf.
All sampled material and king buckets are represented. Raw panel and
evaluation rows remain in rig session66-data-diagnostic; small manifests,
exposure histograms and summary are [mirrored](2026-10-11-session66-results/diagnostic/).

Integer-minus-FP32 mean−2.241cp, absolute mean6.017cp, absolutep95
15.161cp, maximum30.157cp. Sampled blended loss is0.00901121 for FP32,
0.00900758 BF16 autocast and0.00903308 deployed integer; accepted
a4d90b41 integer scores0.00893673 on the same panel. Integer export adds
about0.24% to this sample's float loss. This does not identify a large
quantization defect or justify a quantization training treatment now.
FP32 used the diagnostic process's default highest matmul precision;
the trainer sets high precision (TF32 eligible) for validation. The report
is an ideal-float/export comparison, not an exact replay of trainer loss.

| Source | Exact raw corpus share | New integer loss | Accepted integer loss | Integer/float MAE cp |
|---|---:|---:|---:|---:|
| T80 |21.20%|0.00874507|0.00851847|6.36|
| T76 |15.52%|0.00885770|0.00871701|5.91|
| Full T74 |51.17%|0.00913419|0.00909649|5.92|
| Fixed10% T75 |12.10%|0.00939534|0.00941497|5.88|

The frozen sample suggests a small tradeoff toward T75 and away from the
older sources; it does not establish significance, Elo or a sampling fix.
Complete material/king-bucket strata and sample sizes are in summary.json.
Source balance in the panel differs from corpus balance. Block sampling,
duplicate boards and the existing game/mirror-disjoint limitation remain.
Exposure histograms are sampled;14.848B volume-based presentation shares
are estimates because the completed trainer did not retain actual source/
bucket presentation counters. No full-corpus rescan or training replay.

The fixed400 game gate remains the data acceptance test. No selected
checkpoint, playing net, training recipe or future training nomination
changes on this diagnostic. GPU stays idle after this bounded diagnostic
under Astra's explicit post-training instruction, until an informative
experiment is prospectively nominated; no width/epoch/source lottery.
