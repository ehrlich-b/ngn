# Current NNUE / multicore sprint status

Updated 2026-09-06 while the prospective Counter/HCE match is running, after Stockfish 18 SMALL full-refresh integration and peer review of the incremental oracle consumer and Counter transition optimization. All project work runs on WSL; the Mac is transport only. Sol owns implementation and root coordinates and reviews.

The deployed engine remains **53e4d1b**. The development engine's latest search change is **168385f** (real multicore search). The Counter 5.5 loader, incremental context, and search adapter are integrated through **0ffbc52**; the Stockfish SMALL loader and full-refresh evaluator are integrated through **48b67dd** without search wiring. A useful eight-versus-one-thread playing-strength gain is now accepted. A fresh absolute WSL rating and an NNUE gain have not been established. The previous 2884.14 rating belongs to historical native-Windows conditions.

## Active work

| Area | Accepted result | Next step |
|---|---|---|
| Real multicore search | Private worker state, shared synchronized TT, joined helper lifecycle, full short/race/vet/build gates; 128-sample matrix measured 6.41–6.69x aggregate NPS at eight threads | Eight-versus-one-thread SPRT accepted after 102 games; 2/4-thread contract, 15 focused fixtures and concrete sequential strength manifests accepted; their runs await the exclusive compute slot; combined longer-clock validation remains |
| Match validity | Four-game 8v8 operational control passed: 562 legal plies, correct worker-width receipts, no operational failures or survivors | All 102 games and 14,485 legal plies audited; all six supervised stages exited 0, no warnings or survivors |
| Minimal trained NNUE | Export/scalar/incremental/optimized integer equivalence established; 3M data and 512-update training accepted | Count-only experiments stopped: selected 3M network lost its audited four-game HCE screen 0–4 |
| Counter pretrained control | Exact Counter 5.5 loader, portable and incremental evaluators, historical score adapter, engine/UCI wiring, private worker contexts, dynamic capacity, and transactional backend selection accepted | Recovered four-game diagnostic accepted: Counter 2 wins, 0 losses, 2 draws, 430 legal plies. Original runner remains FAILED after a concurrent-pair-order audit bug; corrected audit passed. Fresh prospective 400-game-cap SPRT is running on the original accepted binary; no strength verdict yet |
| Stockfish compatibility | Exact Stockfish 18 SMALL official file loads; malformed-file, official-file, race, vet and build checks pass | Full-refresh scalar evaluator now matches the actual pinned upstream oracle across 145 positions and all recorded lanes/layers; package, official-file, race, vet and build gates pass. Incremental context, genuine upstream transition-oracle source and 703-row Go consumer are source-reviewed; runtime validation awaits compute |
| HCE speed investigation | Direct-search comparison has exact outcomes/PVs/final nodes; long-budget paired medians show about 2–3% overhead for one position | Further timing remains lower priority than pretrained-evaluator validation |
| Leading Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 and Chess-3 v4.0 provisioned with protocol handshakes | Legal-search preflight and matched games remain |

## Accepted multicore test

Started **2026-09-06T13:03:25Z**, outer PID **2048736**. Output:

`output/m4c-hce8v1-sprt-20260906-attempt1` in the `ngn-m4c-sprt-runner` worktree.

Both sides use the same pinned 168385f binary. Candidate Threads/GOMAXPROCS=8; reference=1. Time control 30+0.3, Hash 128 MiB per engine, Move Overhead 100 ms, book off, one game at a time, full physical-core mask 0,2,4,6,8,10,12,14. The match held exclusive project compute through auditing and sealing; root has now released the machine to the Counter validation lane.

The predeclared test uses normalized-Elo SPRT hypotheses 0/20, alpha=beta=0.05, complete paired games, first-boundary-crossing stopping, and a cap of 400 games. A cap without crossing is inconclusive. It is not an ordinary +20 Elo confidence claim. Interim results cannot change the stopping rule.

Completed 102 games / 51 pairs: eight threads 63 wins, 3 losses, 36 draws (81/102, 79.41%). The first boundary crossing occurred on the final pair: LLR 3.024581092967444, above 2.9444389791664403. Pair 50 was 2.943521991535266, just below the boundary despite the rounded display. The outer process exited 0 and was absent; all six supervised stages passed. Root verified 94 sealed files (561,677,494 bytes), independently recomputed every PGN pair score and LLR prefix, checked 6,950 multicore per-go width receipts and 355,356 process identity observations, and found no survivors.

Root acceptance: `output/m4c-sprt-root-review-20260906/terminal-review-v1.json`, SHA `353872594a7e4d93deed193cd28c444f73db0cf8a67964d5936a9ffdebe0e9eb`. Final manifest SHA `acf486c8a3add49b254a44c1d25f15d276a430aaa8348839e54f65429b641960`.

The frozen runner passed 55 fixtures and 246 independent C++/Python SPRT vectors. Root accepted all 200 ordered openings and 1,200 legal opening plies. Approved manifest SHA:
`6d901b315382cf5d4ad77fe90869dbdbd82fc136f5244cdd2c08a9d02afc3c38`.

## Evaluator decisions

The minimal NGN-v1 architecture and training pipeline work numerically, but the trained networks have not become competitive with HCE. The 3M selected checkpoint improved fixed validation loss to 0.058133 and then lost four games by natural mate. That result triggered the prospective stop rule. More examples or updates alone are no longer the active strategy.

The next control uses a known pretrained Counter network with exactly reproduced source arithmetic. Root accepted its first slice in
`output/counter-pretrained-root-review-20260906/receipt-v1.json`.
Standalone incremental state is runtime accepted at isolated commit 5e89fd12914f1fc5e7819a85146385b9c65b4cad. The search adapter is accepted and integrated through 0ffbc52. Its final immutable run passed portable and AMD64-v3 arithmetic/oracle gates, the full short engine/NNUE suite, pinned adapter oracles, targeted race tests, vet, and build. Earlier preserved attempts caught a local error-variable scope bug, an unreachable singular-verification fixture threshold, and a test that copied a Position mutex; each correction was narrow and the final source has no unresolved gate failure. Root acceptance is `output/counter55-search-integration-root-review-20260906/terminal-review-v1.json`; build provenance is `build-provenance-review-v1.json` in the same directory. Playing strength and deployment remain unverified.

The in-process Counter/HCE diagnostic runner v4 passed all 61 fixtures. Its actual four-game match completed Counter 2 wins, 0 losses and 2 draws, using the same accepted binary, one thread, Hash 128 MiB, Move Overhead 100 ms and 30+0.3. Two concurrent games used separate physical cores. The original runner ended FAILED because its chess auditor assumed A-white then B-white file order, while fastchess wrote completion order. Corrected auditor v5 passed nine pairing/count fixtures and independently audited all 430 plies, two checkmates and two threefold draws while preserving each game's final-EPD association. Root accepted the recovered diagnostic; the original FAILED record remains unchanged. This is not an Elo result. Root receipt: `output/counter55-inprocess-runner-root-review-20260906/recovered-four-game-review-v1.json`, SHA `bf763bf11dbbbe75aeeda32c22272a559b5bd0d979ee9ca4361fe1438c158cb7`.

The next evaluator test is a fresh, predeclared Counter-versus-HCE prospective SPRT with the same binary: one thread per side, 30+0.3, Hash 128 MiB, Move Overhead 100 ms, one game at a time on CPU 12, normalized 0/20 hypotheses, alpha=beta=0.05, complete-pair first-crossing stopping and a 400-game cap. Source capability review and all 44 focused runner fixtures passed. The approved manifest SHA is 0309ed545522f0180b7dd48c9297432124d1164a5ed835171ed4a487365d5d1b. The run launched at 2026-09-06T17:30:18Z in ngn-counter55-sprt-capability/output/counter55-vs-hce-sprt-20260906-attempt1. Both backend preflights passed. The retained foreground runner is still active; no terminal verdict is available. Sol owns the run through auditing and process cleanup. Its binary, configuration and stopping rule remain frozen.

A source review of the installed Go 1.25.5 compiler found that AMD64-v3 may fuse the Counter output multiply-add, whereas the canonical upstream reference was built for v1. Source v5 explicitly rounds each float32 product before accumulation and includes a strict-loader numeric witness that distinguishes separate +0 from fused -2^-46. The first execution reproduced the defect in the actual pre-fix AMD64-v3 test binary: output bits a8800000 instead of 00000000. The corrected AMD64-v3 witness and both official Counter reference tests passed. It does not explain the earlier integer NGN-v1 training losses.

The first standalone incremental run exited 1 at the transition FEN assertion after e2e4: Counter retains e3 even when no adjacent pawn can capture, while NGN records -. Features, accumulator lanes and raw score had matched before that assertion. Root checked NGN's parser and move path against the same adjacent-pawn policy. The v6 correction changes one test file to compare NGN-canonical FENs, preserves capturable en-passant rights and the other position fields, and includes negative controls. Attempt 1 remains preserved; no timeout, resource failure or child survivor occurred. Attempt 2 completed all eleven planned stages and exited 0. The intentionally failing pre-fix FMA observation was reproduced; every corrected-code stage passed. Root rehashed all 1,268 sealed files (27,989,869 bytes), verified the actual outer exit and absent PID, checked all stage receipts, and required all five key parity/rounding/EP tests to pass under both ordinary and race builds. Standalone acceptance: output/counter-pretrained-incremental-root-review-20260906/terminal-review-v2.json, SHA c046567ad639197d500c58b56cc0b1bc8596f0a4579436a427ff73d8e53cb089. This accepts the standalone evaluator. The separately accepted search adapter now closes integration correctness; playing strength remains unverified.

Stockfish support is an exact architecture port. The official SF18 SMALL model is verified at 3,519,630 bytes with SHA
`37f18f62d772f3107e1d6aaca3898c130c3c86f2ab63e6555fbbca20635a899d`.
The loader and scalar full-refresh evaluator are now runtime accepted. Root acceptance is `output/sf18-small-fullrefresh-20260906/root-terminal-review-v1.json`, SHA `fbd7879065c413a6e5accf51d7a9bb9b7596dc76ed1ae72752f530a1101afedb`. Actual pinned scalar Stockfish generated 145 oracle rows; both official Go tests passed. Package and race runs each passed 19 tests with two expected official-input skips; the explicit official gate passed those two tests. Vet and build passed. All five Go stages exited 0 with no survivors. Earlier instrumentation-patch and test-inventory-order failures are preserved; the Go-only continuation reused the successful reference build and oracle.
Arbitrary Stockfish or other .nnue files are not supported. Stockfish selects SMALL only conditionally and may fall back to BIG; SMALL compatibility alone does not reproduce its general evaluation. The incremental source v2 stays within the package and includes no engine/UCI/score-policy wiring. Root and independent review caught compact updates accepting captured kings, which could leave a missing king for later feature indexing, and an ineffective Go copylocks marker. Sol fixed both and added exact-state rejection and mixed real/null/full-unwind coverage. These incremental fixes are source-reviewed, not yet runtime accepted. Full-refresh parity has cleared; actual upstream incremental-transition parity remains the next prerequisite.

Two additional source reviews are closed. The Counter transition decoder removes generic NGN-v1 position-fact preparation while retaining checked board updates, private contexts and exact float32 arithmetic. Its comparison harness uses identical test sources on parent and candidate, six positions, cold and warm searches, exact result/PV/node snapshots, and explicit caller/context restoration. A separate CPU profile must provide at least 20 child CPU seconds. No performance or strength gain is claimed before execution.

The Stockfish incremental producer exercises actual upstream move/null/undo state and records all layers across 143 cases and 703 rows, including a 132-real-ply growth/unwind case. The Go consumer checks both full and compact transitions, the public Context.EvaluateAll result, every recorded internal lane and the widened no-overflow prerequisite. Source acceptance for the final consumer and Counter harness is output/nnue-source-peer-review-20260906/root-source-acceptance-v1.json, SHA b91168a4dbd9607bc5a76a7c4ffd76967b1141438cdbb409468424d87baf4fbf. Compilation, oracle generation, package tests and profiling remain held while the strength match owns compute.

Stockfish BIG now has a separate immutable loader implementation at `ngn-sf18-big-loader`, source-reviewed with independent parser/test review. Root verified the exact 108,919,594-byte official file, its full digest, and the pinned archive's Git LFS object identity. This slice has no feature extraction, evaluation, search or UCI wiring. BIG differs from SMALL in threat features, tensor ordering and arithmetic; the staged path and memory costs are documented in `2026-09-06-sf18-big-dual-plan.md`. Official parsing and all runtime gates remain pending.

The actual SMALL incremental C++ producer and Go continuation launchers have now passed source review. They require upstream reuse/incremental/refresh coverage, all 703 Go comparisons, distinct successful stage receipts, final input rehashing and external process completion checks. The Counter profile plan now runs all three decoder tests plus exact cold/warm snapshots before its fixed profile and 28 timing processes. Search-focused CPU attribution is separated from benchmark setup. No new runtime or performance result is claimed from these preparations.

The direct HCE diagnostic found substantially less overhead than the earlier short UCI timings: paired session median ratios 1.023 cold and 1.027 warm for the longer budget, on one position. These include public search admission and callback reporting. The separate HCE transition fast path remains unintegrated because measured effects are modest and fixture dependent.

## External Counter admission correction

The saved Counter 5.5 handshake advertised Threads max=1 because it ran under `taskset -c 8`, a one-CPU mask. Pinned release source sets the maximum from `runtime.NumCPU()` and implements a real worker pool and multicore scheduler. The old external runner plan inferred an engine limit incorrectly. Corrected plan v3 keeps the first comparison configured at one thread per side, requires a fresh handshake bound to the actual admission mask, and leaves matched eight-versus-eight as a later gated comparison. External diagnostic runner source review also found Python boolean/integer type aliases in manifest validation, NGN-only book checks applied to Counter, incomplete acquisition provenance, and a missing exact one-worker launch fixture. The bounded correction is now source accepted; external runner runtime validation remains pending. No external opponent games have run.

## Remaining release gates

Complete useful multicore strength results, integrate and test the pretrained evaluator, establish an NNUE improvement over HCE, confirm the combined release at a longer clock, compare identified Go opponents under matched conditions, and deploy with rollback. NPS, validation loss, loader compatibility and short favorable scores do not satisfy those gates.

See the existing detailed evidence documents:
- `2026-09-05-next-stage-roadmap.md`
- `2026-09-06-smp-scaling-verdict.md`
- `2026-09-06-nnue-three-million.md`
- `2026-09-06-counter55-loader-oracle-result.md`
- `2026-09-06-counter55-search-adapter-result.md`
- `2026-09-06-hce-fastpath-diagnostic.md`
