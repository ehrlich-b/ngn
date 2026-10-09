# Sparse transformer training — prospective session 17 candidate

Topic `night/sparse-transformer`, base **9b6a3d6**, three-file patch.
The accepted runtime **5b882e2 / be34532c** and live factorizer/SPSA are unchanged.

The current decoder constructs a dense [2B,768*KB] feature tensor although a
position has at most32 occupied squares. At B16384/KB8 this is768 MiB in FP32,
then a dense transformer GEMM. Add an optional `--sparse-transformer` path:
same perspective/king/mirror indices, packed occupied indices and bag offsets,
then a sum of selected transformer rows using PyTorch's differentiable
embedding sum primitive. No third-party chess code, nets or data is introduced.
Model shapes, initialization/RNG, parameters, optimizer groups, export, engine
and default dense path stay unchanged. The flag is bound on resume; never
transfer an in-progress dense training trajectory to sparse sums.

Sparse and dense represent the same real-valued function and gradients.
Floating reduction order differs. Under CUDA BF16 the sparse path explicitly
casts effective transformer weights, matching dense GEMM's autocast precision;
bias remains FP32. Validation uses FP32, as in the existing trainer. The
optional shared factorizer rows remain differentiable and fold normally.

CPU checks require exact decoded feature reconstruction, labels and bucket
indices; forward and every parameter gradient against dense at NB1/2/4/8,
NGNN1/2, NGNN3 with factorizer off/on, and NGNN4 L16; both STM/mirror states;
empty/repeated bags; unchanged seeded state/export; sparse CLI resume exactness
and rejection of changing the flag. FP32 tolerance rtol/atol3e-5 (empty bags
1e-6). Full trainer suite must pass. CPU13,15 / nice19 / CUDA hidden waits
natively for fused-output controller641437 to finish its fixed timing sample.

Actual CUDA FP32/BF16 forward/all-gradient/fused-AdamW controls follow CPU pass
and the corrected batched CUDA controller641099's normal terminal window.
BF16 forward tolerance rtol.015/atol.05, gradients rtol.025/atol.1. Zero skips,
finite updated/clipped parameters; no tolerance change after observing results.
Then benchmark complete **decode + model forward + backward**, batch16384,
H512/NB8: NGNN3 factorizer off/on and NGNN4 L16 factorizer off. Same seeded
owned synthetic records and model state on both sides, old gathered W2 for
both NGNN4 sides, BF16, no optimizer updates while timing. Warm each path.
Exactly20 fixed AB/BA blocks per configuration, five steps per side, clear
gradients each step, CUDA events/synchronization and peak allocated memory.
Keep on topic only if all3 configurations have geometric dense/sparse throughput
ratio lower95>1 (two-sided t on log ratios, df19), both order strata>1 and lower
peak memory. Otherwise shelve without extension. No Elo, exact floating
trajectory, or final NGNN4 recipe claim; interaction with any subsequently
accepted batched-W2 edit still needs cost review before recipe selection.

Results `/home/ehrli/ngn-data/night-20261007/session17s/`; stage
`sparse-transformer`. Native dependencies bounded12h, no interrupted live job,
no real training, games, auto-merge or promotion. Preserve the factorizer and
batched cost window before this short CUDA window, and all queued CUDA windows
before a new real trainer or exclusive claim. Local uncommitted source stays
on the topic until the complete fixed rule is reviewed.

CPU controller651273 completed **21:49:52 EDT**, exit0: **68 tests /6
intentional CUDA skips**,82.299s full suite; all narrow feature/gradient/
initialization/export/resume controls pass. The three local files equal
the tested source once at session18 review. CUDA651274 remains waiting
for batched641099 after factorizer630389. CPU evidence alone supplies no
cost verdict; the candidate stays uncommitted on its topic.
[CPU receipt](2026-10-08-session18-results/completed-topics.json).
