# New-code provenance check — 2026-10-05

**Bounded finding: no substantive new implementation clone or copied distinctive chess table was identified in the available references. Complete provenance clearance remains open: important trainer/data-tool upstreams are unavailable, and published hash constants plus externally attributed search parameters remain in the engine source closure.**

Audited detached HEAD `fb0c443c01131b83d17a07f62fa0d5f1b02888e5` = local `origin/indep/main`, against baseline `74d63eec8f1ee3d18f4c7a3014e486ae0e1f964f`. Scope is all 93 paths from `git diff --name-only 74d63ee..HEAD` excluding `experiments/` and `docs/`: 79 source/text-fixture files measured below, four prose files, two binary fixtures, and eight deleted paths. Rename detection additionally exposes the two old relocation paths listed below. No Go command, engine, trainer, WSL connection, commit, or push was performed. This report is the sole worktree write.

## Evidence and method

Reused `scripts/provenance_similarity.py`'s tokenizer and suffix automaton. **T** is the longest contiguous normalized shared token run; **%** is the directional fraction of this NGN file covered by the union of all shared runs of at least 20 tokens across the entire corpus. Identifiers are normalized; the script's Go/C keyword set, operators and literals remain; integers are canonicalized across decimal/hex. **I** is the longest run with identifiers retained. Comments/whitespace are ignored. Python uses its standard lexical frontend so `//` stays division; its strings/docstrings remain tokens. Shell comment lines are removed. The script's ordinary normalization also erases assembly mnemonic/register names, so instruction-preserving results are separately reported. These are similarity percentages, **not copied-code percentages**. Whole-file results include unchanged baseline regions and tests; speed additions are isolated separately. Thirty small deterministic brute-force checks confirmed the reused automaton's longest-run/coverage results.

497 distinct source texts were compared (identical hashes deduplicated): 268 unique texts from all 273 pinned files, 121 additional old-main texts, 88 additional research texts, and 20 local variants/partial patches. All pinned file hashes match the prior verification manifest. Reference abbreviations:

- `Z`: Zahak `72ff8066e2d8348d8d3847fab62c4a6b8344240d`; `C38`: Counter `b172b99b3128d82c44ef1d8d8a278aa09f2d721b`; `C55`: Counter `63c487ca724c620f71c129d62129c6fb9109c872`.
- `SF`: Stockfish `cb3d4ee9b47d0c5aae855b12379378ea1439675c`; `S`: Stash `13e0a81a4af7fb6af43405228535c5058ee71657`; `D`: Disservin chess-library `53e6a841dcda7059a2af363d85f785ef1817304a`.
- Pinned root: `/Users/ehrlich/repos/ngn-clean/output/indep-verification/references`, as bound by [the verification receipt](2026-10-05-indep-verification.json). The older audit is absent here but exists at `/Users/ehrlich/repos/ngn/experiments/2026-10-05-provenance-audit.md`.
- `O`: Git `e4f138ca425ac2585ec2cfd54ae782dfb2177887`, all source under `rodenteval/`, `rodentv12eval/`, `countereval/`, `nnue/`, including tests. `K`: research/owned-continuation `1831e39769c32fc3d3aedf33b9f3b9da0ca21963`, including `nnue/ngnk4`, expanded SF18 code, and `training/nnue`.
- `L`: `/Users/ehrlich/repos/ngn-research` evaluator/training variants, including NGN's Bullet harness and partial Bullet patch. `X`: permitted `ngn/output` recorded native-opponent sources and training variants. Local variants are measurement evidence, not independent upstream proof.

No complete local nnue-pytorch or Bullet checkout, Rodent V upstream checkout, sfbinpack/binpack-rust decoder, or marlinformat implementation was found in the permitted roots. Bullet evidence is NGN harnesses plus patch context/additions; old binpack evidence is NGN's sampler/packer using sfbinpack, not the crate source. Direct Rodent comparison therefore uses the historical NGN compatibility packages. The inaccessible `/home` trainer was skipped. No network sources were fetched. Cross-language syntax and unseen source versions limit negative findings.

## Per-file measurements

Witness notation is `NGN line span → reference path:line span` for T, with reference prefixes expanded above. Zero means no shared token; below-20 runs have 0% coverage. JSON evaluation fixtures are text/data, not executable code.

| File | T / % | I | Longest normalized witness |
|---|---:|---:|---|
| `cmd/blunder-finder/main.go` | 32 / 18.78 | 19 | `141–141 → Z/engine/bitboard.go:385–393` |
| `cmd/datagen/main.go` | 35 / 22.20 | 16 | `48–53 → K/training/nnue/k4label/uci.go:75–80` |
| `cmd/datastat/main.go` | 27 / 8.25 | 17 | `93–95 → O/countereval/context_transition_oracle_test.go:100–102` |
| `cmd/datastat/main_test.go` | 27 / 30.60 | 20 | `56–62 → K/training/nnue/k4pack/pack_test.go:160–166` |
| `cmd/magicgen/main.go` | 23 / 7.41 | 13 | `34–36 → O/nnue/sf18big/format_test.go:688–690` |
| `cmd/speedprobe/main.go` | 31 / 10.81 | 10 | `37–38 → K/training/nnue/k4label/run.go:478–479` |
| `cmd/speedprobe/main_test.go` | 22 / 24.00 | 13 | `11–13 → K/training/nnue/k4label/contract_test.go:69–71` |
| `cmd/speedprobe/network.go` | 19 / 0.00 | 11 | `8–12 → C55/internal/dataset/analyze.go:45–49` |
| `cmd/sprt/main.go` | 50 / 6.26 | 18 | `258–262 → O/nnue/sf18small/incremental_oracle_test.go:318–321` |
| `cmd/sprt/sprt_test.go` | 23 / 8.04 | 14 | `31–35 → K/training/nnue/k4pack/pack_test.go:152–156` |
| `engine/bitboard.go` | 46 / 25.51 | 46 | `113–127 → Z/engine/bitboard.go:46–56` |
| `engine/board_relocation_test.go` | 25 / 18.48 | 9 | `38–39 → SF/src/nnue/features/half_ka_v2_hm.h:61–62` |
| `engine/book_embedded.go` | 22 / 3.16 | 9 | `179–181 → K/nnue/sf18big/reference_oracle_test.go:333–335` |
| `engine/book_test.go` | 18 / 0.00 | 8 | `20–24 → O/nnue/sf18big/format_test.go:297–301` |
| `engine/enpassant_geometry.go` | 16 / 0.00 | 8 | `5–7 → Z/book/defs.go:76–78` |
| `engine/enpassant_hash_test.go` | 21 / 16.21 | 12 | `14–17 → O/nnue/sf18small/incremental_oracle_test.go:368–371` |
| `engine/enpassant_repetition_correctness_test.go` | 39 / 23.93 | 29 | `352–358 → K/training/nnue/k4label/uci_test.go:101–106` |
| `engine/evaluation_core.go` | 30 / 6.95 | 16 | `100–110 → X/owned-continuation-artifacts-20260928/native-rodent/source/numeric.go:7–14` |
| `engine/evaluation_geometry_test.go` | 26 / 11.94 | 12 | `11–14 → C38/eval/evaluation.go:512–515` |
| `engine/evaluator_selection.go` | 27 / 14.11 | 14 | `48–52 → K/training/nnue/k4label/uci.go:50–54` |
| `engine/fen.go` | 22 / 3.79 | 17 | `10–13 → O/nnue/sf18small/oracle_test.go:194–197` |
| `engine/move_test.go` | 37 / 16.15 | 14 | `23–28 → O/nnue/sf18small/context_test.go:265–269` |
| `engine/movegen_test.go` | 41 / 26.38 | 16 | `65–71 → K/training/nnue/k4pack/pack_test.go:230–235` |
| `engine/moveorder.go` | 47 / 22.16 | 14 | `346–350 → O/rodentv12eval/load.go:59–63` |
| `engine/ngnn1.go` | 31 / 10.64 | 21 | `54–61 → K/nnue/ngnk4/format.go:44–52` |
| `engine/ngnn1_bench_test.go` | 30 / 37.69 | 29 | `68–70 → O/rodentv12eval/output_dot_test.go:171–173` |
| `engine/ngnn1_kernels.go` | 16 / 0.00 | 9 | `6–7 → C38/uci/protocol.go:205–206` |
| `engine/ngnn1_kernels_amd64.go` | 14 / 0.00 | 6 | `40–40 → C55/internal/trainer/trainer.go:41–42` |
| `engine/ngnn1_kernels_amd64.s` | 36 / 54.83 | 11 | `135–141 → O/rodenteval/output_dot_amd64_v3.s:33–39` |
| `engine/ngnn1_kernels_generic.go` | 21 / 10.19 | 6 | `13–16 → O/rodenteval/apply_updates_amd64_v3.go:28–34` |
| `engine/ngnn1_kernels_test.go` | 30 / 18.09 | 21 | `192–196 → O/countereval/oracle_parity_test.go:73–77` |
| `engine/ngnn1_test.go` | 44 / 16.99 | 20 | `394–400 → O/nnue/context_int32_test.go:160–166` |
| `engine/ngnn1_uci_test.go` | 22 / 10.45 | 16 | `53–57 → O/nnue/dot_int32_test.go:202–206` |
| `engine/ngnn2_test.go` | 39 / 17.15 | 21 | `250–257 → K/nnue/ngnk4/format_test.go:122–129` |
| `engine/perft_reference_test.go` | 39 / 36.12 | 15 | `21–25 → O/rodenteval/oracle_parity_test.go:82–86` |
| `engine/piece_test.go` | 37 / 32.16 | 14 | `25–30 → O/nnue/sf18small/context_test.go:265–269` |
| `engine/private_hash_proof_test.go` | 26 / 25.29 | 17 | `111–114 → O/nnue/sf18small/evaluate.go:266–269` |
| `engine/private_position_proof_test.go` | 35 / 24.88 | 16 | `143–143 → O/nnue/sf18small/incremental_oracle_test.go:673–674` |
| `engine/search.go` | 52 / 9.77 | 26 | `542–573 → Z/engine/square.go:79–130` |
| `engine/search_control.go` | 28 / 30.15 | 14 | `105–112 → O/countereval/context.go:170–176` |
| `engine/slider_attacks.go` | 27 / 16.32 | 9 | `60–63 → C38/common/bitboard.go:143–147` |
| `engine/slider_magic_test.go` | 21 / 15.16 | 14 | `21–23 → O/countereval/evaluate.go:22–24` |
| `engine/slider_magics_generated.go` | 9 / 0.00 | 5 | `70–71 → C38/evalpesto/weights.go:41–44` |
| `engine/square_test.go` | 37 / 37.44 | 14 | `21–26 → O/nnue/sf18small/context_test.go:265–269` |
| `engine/testutil_test.go` | 40 / 27.12 | 21 | `101–105 → C38/common/position.go:439–443` |
| `engine/uci.go` | 55 / 14.67 | 22 | `1107–1109 → Z/engine/bitboard.go:385–398` |
| `engine/uci_evaluator.go` | 30 / 12.58 | 11 | `134–140 → K/training/nnue/k4label/uci.go:49–55` |
| `engine/uci_ownbook_test.go` | 26 / 9.66 | 12 | `12–14 → K/nnue/ngnk4/context_test.go:32–34` |
| `engine/worker_evaluator.go` | 26 / 23.06 | 15 | `60–66 → O/nnue/format.go:124–131` |
| `internal/datagen/datagen.go` | 29 / 14.44 | 12 | `296–297 → K/training/nnue/k4sample/src/bin/scan.rs:415–420` |
| `internal/datagen/datagen_test.go` | 62 / 26.80 | 22 | `16–28 → K/training/nnue/k4label/k4key_test.go:8–20` |
| `internal/datagen/ngnn2_test.go` | 34 / 42.83 | 21 | `34–40 → O/nnue/context_engine_test.go:163–169` |
| `internal/ngnp/record.go` | 62 / 15.72 | 12 | `193–196 → K/training/nnue/k4pack/pack.go:502–505` |
| `internal/ngnp/record_test.go` | 58 / 30.82 | 22 | `16–26 → K/training/nnue/k4label/k4key_test.go:10–20` |
| `internal/ngnp/shard.go` | 27 / 13.46 | 15 | `103–107 → K/training/nnue/k4finalize/finalize.go:611–615` |
| `internal/ngnp/shard_test.go` | 42 / 42.65 | 21 | `52–60 → O/countereval/context_drift_oracle_test.go:56–64` |
| `internal/oracle/stockfish_client.go` | 28 / 7.76 | 21 | `264–270 → K/training/nnue/k4finalize/finalize.go:534–538` |
| `internal/sliderray/ray.go` | 32 / 26.23 | 32 | `5–5 → K/nnue/sf18big/threats.go:496–496` |
| `internal/uci/uci.go` | 26 / 5.84 | 19 | `334–344 → Z/search/types.go:397–404` |
| `internal/uci/uci_test.go` | 33 / 19.59 | 15 | `65–73 → K/training/nnue/k4pack/pack_test.go:210–218` |
| `main_test.go` | 21 / 2.85 | 14 | `29–34 → O/nnue/context_int32_test.go:334–339` |
| `scripts/datagen_run.sh` | 12 / 0.00 | 7 | `48–48 → Z/fathom/tbprobe.c:292–292` |
| `scripts/netinfer-probe.py` | 23 / 2.55 | 14 | `28–29 → O/nnue/sf18small/testdata/upstream_incremental_oracle/generate_incremental_oracle.py:129–131` |
| `scripts/provenance_similarity.py` | 28 / 2.06 | 18 | `98–101 → K/nnue/sf18big/testdata/upstream_reference_oracle/freeze_representative_sequences.py:28–31` |
| `scripts/speedcheck.py` | 15 / 0.00 | 14 | `17–17 → Z/engine/nnue_instructions_amd64.go:9–9` |
| `testdata/ngnn1/random_h32_evals.json` | 5 / 0.00 | 5 | `9–11 → O/nnue/dot_int32_test.go:191–192` |
| `testdata/ngnn2/random_h32_b8_evals.json` | 5 / 0.00 | 5 | `10–12 → O/nnue/dot_int32_test.go:182–183` |
| `trainer/__init__.py` | 0 / 0.00 | 0 | `docstring only; no shared token` |
| `trainer/export.py` | 16 / 0.00 | 14 | `12–15 → K/nnue/sf18big/testdata/upstream_reference_oracle/generate_reference_oracle.py:31–34` |
| `trainer/make_random_net.py` | 25 / 3.47 | 9 | `61–62 → O/countereval/oracle_parity_test.go:86–88` |
| `trainer/model.py` | 21 / 1.89 | 8 | `93–94 → O/nnue/context.go:213–216` |
| `trainer/ngnp.py` | 14 / 0.00 | 9 | `180–182 → K/nnue/sf18big/testdata/upstream_reference_oracle/generate_reference_oracle.py:31–34` |
| `trainer/refeval.py` | 15 / 0.00 | 14 | `46–46 → Z/engine/bitboard.go:385–388` |
| `trainer/tests/__init__.py` | 0 / 0.00 | 0 | `docstring only; no shared token` |
| `trainer/tests/check_material.py` | 18 / 0.00 | 8 | `28–28 → O/countereval/context_transition_oracle_test.go:210–210` |
| `trainer/tests/synthetic.py` | 15 / 0.00 | 9 | `53–53 → Z/engine/nnue.go:96–99` |
| `trainer/tests/test_ngnn2.py` | 18 / 0.00 | 7 | `42–43 → O/nnue/sf18small/testdata/upstream_incremental_oracle/generate_incremental_oracle.py:83–85` |
| `trainer/tests/test_trainer.py` | 20 / 0.57 | 17 | `97–98 → K/training/nnue/k4label/contract.go:271–272` |
| `trainer/train.py` | 21 / 1.70 | 11 | `113–115 → K/nnue/sf18big/testdata/upstream_reference_oracle/freeze_representative_sequences.py:30–32` |

## Interpretation and flags

- **Inference/AVX2:** `ngnn1.go` reaches T31/10.64%, I21: open/error/defer/reader dispatch against K4's loader. Portable kernels reach only T16/0%. Assembly's T36/54.83% is inflated by erased instructions/registers (loader/prologue at lines 135–141). Retaining uppercase instructions/registers yields **T17/0%**, a loop tail/prologue boundary against `O/countereval/feature_updates_amd64_v3.s:17–23`; retaining all identifiers gives I11. No old Rodent/K4 king-bucket table, scale-206 arithmetic, modular-int32 output kernel, or distinctive comment was found transplanted. The new int32/int64 arithmetic and runtime-width slices differ from K4's int16/fixed-layout kernels. Chess768 orientation, SCReLU, and `[QA,QB,SCALE]=[255,64,400]` are shared design choices, not evidence of an implementation copy. NGNN2's NB=8 material bucket coincides with the customary `(pieces−2)/4` formula used by K4; it carries no copied bucket table.
- **Data layout overlap — flag:** `internal/ngnp/record.go:193–196` has **T62** against `K/training/nnue/k4pack/pack.go:502–505`: occupancy at bytes 0–7, 16 packed-piece bytes at 8–23, signed score at 24–25, then byte fields. This is more specific than a lone error-check idiom and should be disclosed as packed-format design overlap. Whole-file I is only 12, fields/tail semantics differ, and NGNP uses black codes 6–11 versus old K4's color bit 8; the records are not interchangeable. No decoder/transformation implementation clone was established; full binpack/marlinformat source comparison is unavailable. The other T62 (`internal/datagen/datagen_test.go:16–28`) is imports, a standard ParseFEN test helper, and the next test signature.
- **Inherited API resemblance — disclose:** `engine/bitboard.go` retains 18 Zahak method names (e.g. `GetWhitePieces`, `GetBlackPieces`, `GetBitboardOf`, `Move`) and identical simple occupancy getters: T46/I46, lines 113–127. This API/decomposition resemblance predates these merges. `worker_evaluator.go` shares `Reset/Pop/PushMove/PushNull` lifecycle names with historical contexts; no larger matching body accompanies them. Identical four cardinal offsets in `sliderray` versus the old SF threat helper are standard geometry, not a distinctive learned/copied table.
- **Speed delta versus Zahak:** isolated added hunks from `d6a140e..0d8fb9b` give bitboard relocation **T11/0%** (153 added tokens) and evaluation geometry/occupied iteration **T14/0%** (273 added tokens). The whole bitboard T46 is unchanged getter code. Search's whole-file T52 is a repeated field-declaration shape; UCI T55 is a formatting argument list. These are not new search/board algorithm clones.
- **Comments and names:** no exact non-license, non-build comment of at least 20 characters matches the directly inspected pinned/history Go/assembly references; the broader local-source scan found no match at 45 characters. No distinctive newly added function-name decomposition was identified beyond the disclosed API compatibility.

Against only the available trainer-related sources (Counter's own Go trainer, NGN K4/Bullet harness variants, partial Bullet patch, old fitting script), every Python trainer file has **0% coverage by runs ≥20**. This is a bounded result, not clearance against the missing PyTorch/Bullet upstreams:

| Trainer file | T / % against available training references |
|---|---:|
| `trainer/__init__.py` | 0 / 0.00 |
| `trainer/export.py` | 15 / 0.00 |
| `trainer/make_random_net.py` | 15 / 0.00 |
| `trainer/model.py` | 13 / 0.00 |
| `trainer/ngnp.py` | 13 / 0.00 |
| `trainer/refeval.py` | 12 / 0.00 |
| `trainer/tests/__init__.py` | 0 / 0.00 |
| `trainer/tests/check_material.py` | 13 / 0.00 |
| `trainer/tests/synthetic.py` | 15 / 0.00 |
| `trainer/tests/test_ngnn2.py` | 13 / 0.00 |
| `trainer/tests/test_trainer.py` | 14 / 0.00 |
| `trainer/train.py` | 15 / 0.00 |

## Remaining data/numbers in the engine source closure

Static non-test import inspection gives **root → engine → internal/sliderray** only, with no third-party Go module requirement. There are 40 candidate non-test Go files; ordinary builds select one of the two mutually exclusive kernel wrappers (39 Go files), plus the new assembly on amd64/v3. This is a source closure, not a fresh linker receipt. `internal/oracle` (Stockfish subprocess), datagen/trainer, tests and all fixtures are outside it. No `go:embed` or default bundled network was found.

| Retained input | Evidence / classification |
|---|---|
| Published hash mixing constants | `moveorder.go:34,107,155`: `9E3779B97F4A7C15`, `C2B2AE3D27D4EB4F`, `FF51AFD7ED558CCD`, `C4CEB9FE1A85EC53`, `BF58476D1CE4E5B9`, `94D049BB133111EB` (hex). Comments explicitly call the latter pairs Murmur3 and SplitMix64 finalizers. **Externally designed algorithm constants remain**; this is not NGN-generated tuning data. No equality to numeric literals was found in the pinned engine corpus, so that corpus does not establish their complete source history. |
| Attributed search/correction curves and defaults | `search.go:27–28,77–88,118–131,1630–1694,2042,2117`: Counter-grounded futility **103**, depth **8**; LMP **4+depth²**, non-improving divisor **2**; reverse-futility **88**, depth cap **7** (comment cites historical SF11/Ethereal/Weiss, not provenance of 88); null **3+1+depth/6**; singular **48*depth/32**; Stockfish-style LMR **log(d)*log(m)/2.0**, 64×64 bounds and non-PV **+1**. **External ideas/calibration attribution remains; comments alone do not prove each current literal was copied.** |
| Correction/order/time defaults with external attribution | `moveorder.go:7,62–78,461–478`, `time.go:462–498`: correction clamp **1024**, depth divisor **8**, cap **limit/4**; capture bands **80000/−40000**, offset **1000**, history divisor **8**; node-effort **0.85/1.25**, neutral **1.0**. Corresponding comments name Stockfish/Counter/Weiss/Ethereal. Aspiration divisor **80** and correction multiplier **3/64** were explicitly substituted NGN placeholders; the former **81** and **6245/131072** do not remain in these live formulas. |
| Foreign format identifiers | `syzygy.go:459–505`: WDL **0x5d23e871**, DTZ **0xa50c66d7**. They identify a third-party file format; no tablebase is embedded, and `syzygyProbeTrusted=false` gates usable probes. Their source-file presence does not establish survival of every symbol after linking. |
| Slider tables/multipliers | 128 NGN-generated multipliers; **0/128** equals any numeric literal across all pinned source files. Attack rows are constructed from coordinate rays. Source generation seed **0x4e474e4d41474943** is recorded; the previous verification binds/replays the generator. No generator or engine was executed in this audit. |
| HCE/hash/book data | HCE PSTs are regenerated from analytic coordinate formulas; no PeSTO/legacy PST literal seed. MG bases **100/315/335/510/975**, EG **100/330/350/540/985** do not equal the ten Counter PeSTO bases. Zobrist values are generated from SHA-256 of `ehrlich` plus Go's PRNG. Embedded opening choices are the pre-existing NGN-authored FEN/move weights; no foreign book provenance was demonstrated. Standard square/move geometry and attack masks are generic chess data. |
| Go toolchain baseline | Runtime/standard-library dependencies remain, including SHA-256, CRC32, math and PRNG implementations with their ordinary published constants/tables. Those are separate from copied chess assets; their exact linked internal tables were not rebuilt or exhaustively audited. |

Thus **no identified foreign neural weight, PeSTO table, Rodent king-bucket table, copied Counter magic array, Polyglot 781-value array, external opening-book file, or tablebase dataset remains as an embedded engine input** in the inspected source. That does not mean zero third-party-derived numbers, and externally supplied runtime files require their own provenance check.

## Remaining scope accounting

`NOTICE`, `README.md`, `TODO.md`, and `trainer/README.md`: prose/documentation, T/% not applicable. NOTICE deliberately preserves historical external attribution; that text is not binary evaluator data.

| Binary fixture | Bytes | SHA-256 | Finding |
|---|---:|---|---|
| `testdata/ngnn1/random_h32.nnue` | 49376 | `9bb171cd06035f7b77089fbd31b215812057a4b44950190fd741d133dd6f52c6` | Recorded seed 20261005; NGNN/NGN2 header and CRC checked; paired JSON SHA matches; outside engine closure. |
| `testdata/ngnn2/random_h32_b8.nnue` | 50304 | `b08c8c2c88188fd3885d4fed4b95232c3da0fd733b7fea0711ecc7d78bb25992` | Recorded seed 20261005; NGNN/NGN2 header and CRC checked; paired JSON SHA matches; outside engine closure. |

The two JSON fixtures each contain 64 evaluation cases and have T5/0% in the main table. The random-network generator source was inspected; networks were not regenerated or evaluated.

Deleted paths have no HEAD token sequence (T/% not applicable):

- `engine/polyglot.go`
- `engine/polyglot_book_move_contract_test.go`
- `engine/polyglot_external_contract_test.go`
- `engine/polyglot_random.go`
- `engine/testdata/polyglot_book_move_oracle.json`
- `engine/testdata/polyglot_book_move_oracle.provenance.md`
- `engine/testdata/polyglot_special_move_oracle.json`
- `engine/testdata/polyglot_special_move_oracle.provenance.md`

The two additional no-rename diff origins are `engine/stockfish_client.go` → `internal/oracle/stockfish_client.go` (tooling), and `engine/testutil.go` → `engine/testutil_test.go` (test-only). Their current destinations are measured above. External Polyglot implementations, constants and associated fixtures are absent.

Reproduction/evidence: scratch measurement script `/tmp/ngn-new-provenance-measure-20261005.py`; full per-source hashes, per-family measurements and witnesses in `/tmp/ngn-new-provenance-results-20261005.json`. Ordered reference name/hash manifest SHA-256: `785112e773a44c2aecf84eb5c6e004d521000d8353b52a2fd6ddcbcdc236e80d`. These scratch files are not committed artifacts; the report carries the findings and every scoped path.

Verdict (eight lines):

1. New inference: no substantive clone detected in inspected sources.
2. AVX2: instruction-preserving maximum 17 tokens, 0% coverage ≥20.
3. Trainer: no ≥20-token run against available training references; upstream gaps remain.
4. Data tools: packed-format overlap flagged; no larger implementation clone established.
5. Speed: new Zahak maxima 11/14 tokens, both 0% coverage ≥20.
6. Tables/weights: no identified copied chess table or embedded foreign model remains.
7. Binary numbers: published hash constants, attributed parameters and format IDs remain.
8. Full provenance clearance: open; report-only audit, no builds/WSL/commits/pushes.
