# Current NNUE / multicore sprint status

Updated 2026-09-06 after root acceptance of the 8-versus-1-thread strength test and acceptance of the corrected Counter incremental execution. All project work runs on WSL; the Mac is transport only. Sol owns implementation and root coordinates and reviews.

The deployed engine remains **53e4d1b**. The development engine's latest search change is **168385f** (real multicore search). Standalone Counter compatibility and Stockfish SMALL loader packages are integrated at **ed9ee07** and **e88d36c**. Neither package is connected to search yet. A useful eight-versus-one-thread playing-strength gain is now accepted. A fresh absolute WSL rating and an NNUE gain have not been established. The previous 2884.14 rating belongs to historical native-Windows conditions.

## Active work

| Area | Accepted result | Next step |
|---|---|---|
| Real multicore search | Private worker state, shared synchronized TT, joined helper lifecycle, full short/race/vet/build gates; 128-sample matrix measured 6.41–6.69x aggregate NPS at eight threads | Eight-versus-one-thread SPRT accepted after 102 games;2/4-thread strength and combined longer-clock validation remain |
| Match validity | Four-game 8v8 operational control passed: 562 legal plies, correct worker-width receipts, no operational failures or survivors | All 102 games and 14,485 legal plies audited; all six supervised stages exited 0, no warnings or survivors |
| Minimal trained NNUE | Export/scalar/incremental/optimized integer equivalence established; 3M data and 512-update training accepted | Count-only experiments stopped: selected 3M network lost its audited four-game HCE screen 0–4 |
| Counter pretrained control | Exact Counter 5.5 loader and portable full-refresh evaluator accepted; 6,144 accumulator lanes, raw scores and historical score adaptation match actual pinned upstream | First standalone execution reproduced the pre-fix AMD64-v3 rounding defect; the corrected evaluator passed both official reference tests. Transition parity stopped on non-capturable en-passant FEN representation. The single-test correction and complete standalone rerun are accepted, including actual upstream transition parity and race checks; search adapter gates are next |
| Stockfish compatibility | Exact Stockfish 18 SMALL official file loads; malformed-file, official-file, race, vet and build checks pass | The first full-refresh execution stopped before compilation on malformed instrumentation patch syntax/context. The corrected patch preserves the intended C++ changes and passes apply-check against pinned source; the fresh v3b run is queued. Incremental package source v2 is reviewed after captured-king and ownership-marker fixes; runtime gates depend on full-refresh parity |
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
The first-slice source is committed and integrated. Standalone incremental state is now runtime accepted and committed in its isolated worktree at 5e89fd12914f1fc5e7819a85146385b9c65b4cad. Search integration is undergoing its full seven-stage validation: early runs caught a local error-variable scoping defect and a singular-verification fixture whose hardcoded TT score lay far below the nonzero synthetic Counter evaluation. The scope correction preserves arithmetic and error behavior. The fixture now derives its TT score from the root evaluation plus 256 while retaining actual verification-transition, repush, full-refresh parity and root-restoration checks; the focused test passed. Complete integration and race acceptance remain pending.

The in-process Counter/HCE diagnostic runner v4 passed all 61 fixtures, with actual outer exit 0 and no survivors. Root acceptance is output/counter55-inprocess-runner-root-review-20260906/terminal-review-v4.json, SHA aaa8e09db9bf2106101f2ed6264bed4c9118b73150d4338dd61f979edeacc92d. Its first comparison will use the same accepted binary for both roles, one thread, Hash 128 MiB and Move Overhead 100 ms at 30+0.3 for four games/two color-swapped pairs. Two concurrent games receive distinct physical cores through fastchess per-game affinity. This is a diagnostic screen, with no SPRT or strength verdict.

A source review of the installed Go 1.25.5 compiler found that AMD64-v3 may fuse the Counter output multiply-add, whereas the canonical upstream reference was built for v1. Source v5 explicitly rounds each float32 product before accumulation and includes a strict-loader numeric witness that distinguishes separate +0 from fused -2^-46. The first execution reproduced the defect in the actual pre-fix AMD64-v3 test binary: output bits a8800000 instead of 00000000. The corrected AMD64-v3 witness and both official Counter reference tests passed. It does not explain the earlier integer NGN-v1 training losses.

The first standalone incremental run exited 1 at the transition FEN assertion after e2e4: Counter retains e3 even when no adjacent pawn can capture, while NGN records -. Features, accumulator lanes and raw score had matched before that assertion. Root checked NGN's parser and move path against the same adjacent-pawn policy. The v6 correction changes one test file to compare NGN-canonical FENs, preserves capturable en-passant rights and the other position fields, and includes negative controls. Attempt 1 remains preserved; no timeout, resource failure or child survivor occurred. Attempt 2 completed all eleven planned stages and exited 0. The intentionally failing pre-fix FMA observation was reproduced; every corrected-code stage passed. Root rehashed all 1,268 sealed files (27,989,869 bytes), verified the actual outer exit and absent PID, checked all stage receipts, and required all five key parity/rounding/EP tests to pass under both ordinary and race builds. Standalone acceptance: output/counter-pretrained-incremental-root-review-20260906/terminal-review-v2.json, SHA c046567ad639197d500c58b56cc0b1bc8596f0a4579436a427ff73d8e53cb089. This accepts the standalone evaluator; search integration and playing strength remain unverified.

Stockfish support is an exact architecture port. The official SF18 SMALL model is verified at 3,519,630 bytes with SHA
`37f18f62d772f3107e1d6aaca3898c130c3c86f2ab63e6555fbbca20635a899d`.
The accepted loader has no evaluator yet. Root acceptance:
`output/sf18-small-loader-20260906/root-review-v1.json`.
Arbitrary Stockfish or other .nnue files are not supported. Stockfish selects SMALL only conditionally and may fall back to BIG; SMALL compatibility alone does not reproduce its general evaluation. The incremental source v2 stays within the package and includes no engine/UCI/score-policy wiring. Root and independent review caught compact updates accepting captured kings, which could leave a missing king for later feature indexing, and an ineffective Go copylocks marker. Sol fixed both and added exact-state rejection and mixed real/null/full-unwind coverage. These are source-reviewed fixes, not executed tests; full-refresh and actual upstream transition parity remain prerequisites.

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
- `2026-09-06-hce-fastpath-diagnostic.md`
