# Retained NGNN4 on the current playing source — session34

Playing source `ac658e2` includes the reviewed null-frame speed keep on
`f8686be/be34532c`. Architecture source `e0f4fb9` has independently passed
48 identities and the fixed20-block VNNI512 speed rule versus retained256:
1.093028 [1.074243,1.112142]. Its exactly12-block same-width budget is
0.858256 [0.833901,0.883323], meeting the predeclared point>=0.85 rule.
These are cost results, not Elo or authorization to bypass data/factorizer.

Rebase the retained architecture onto `ac658e2` in `night/ngnn4-current`.
Preserve the playing TT/SEE/board/null frames; overlay only the original
NGNN4 engine/trainer/owned fixture files. Restore the original
`trainer/tests/test_ngnn4.py` from `ca3637a`: the intermediate row-affine
topic omitted this test file, although engine integer parity remained tested.
Use the original batched-affine trainer edit and its test, already reviewed
in session21 (H512/L16 throughput2.11410, lower95>2.11060, lower memory).
No sparse-transformer edit, changed recipe, actual training or strength
games follow this compatibility stage.

The frozen full patch, source archive, manifest and controller are installed
at `/home/ehrli/ngn-data/night-20261007/session34-architecture/`.
Detached checks use12,14/nice19, CPU-only trainer narrow/full tests, NGNN4
and null-frame controls with AVX2/purego, full short/race/vet, identical
explicit PGO, and36 exact old-format searches against the already checked
null-frame candidate probe (the exact immediate playing runtime).
One bounded CUDA FP32/BF16/fused-AdamW/export and batched-gradient check
then uses the otherwise idle GPU and disjoint12,14 CPUs. No prior fixed
cost sample is repeated. Require zero CUDA skips and all checks pass.
Both engine UCI and probe binaries and all changed source hashes are recorded.

On passing compatibility, commit/push this isolated architecture source;
no playing-line adoption or net strength claim. If accepted source changes
again, apply that playing change and verify its integration before a future
trial. Training remains after the higher-priority data/factorizer sequence;
the chosen recipe must retain whatever factorizer setting is then accepted.
