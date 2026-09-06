# Current NNUE / multicore sprint status

Updated 2026-09-06 at the start of the prospective 8-versus-1-thread match. All project work runs on WSL; the Mac is transport only. Sol owns implementation and root coordinates and reviews.

The deployed engine remains **53e4d1b**. The development engine's latest search change is **168385f** (real multicore search). Standalone Counter compatibility and Stockfish SMALL loader packages are integrated at **ed9ee07** and **e88d36c**. Neither package is connected to search yet. No new playing-strength gain or fresh WSL rating has been established. The previous 2884.14 rating belongs to historical native-Windows conditions.

## Active work

| Area | Accepted result | Next step |
|---|---|---|
| Real multicore search | Private worker state, shared synchronized TT, joined helper lifecycle, full short/race/vet/build gates; 128-sample matrix measured 6.41–6.69x aggregate NPS at eight threads | Prospective eight-versus-one-thread strength test is running |
| Match validity | Four-game 8v8 operational control passed: 562 legal plies, correct worker-width receipts, no operational failures or survivors | Finish the 8v1 match under its frozen stopping rule, then independent chess/trace/SPRT audits |
| Minimal trained NNUE | Export/scalar/incremental/optimized integer equivalence established; 3M data and 512-update training accepted | Count-only experiments stopped: selected 3M network lost its audited four-game HCE screen 0–4 |
| Counter pretrained control | Exact Counter 5.5 loader and portable full-refresh evaluator accepted; 6,144 accumulator lanes, raw scores and historical score adaptation match actual pinned upstream | Sol implementing incremental context and legal-transition oracle as source work |
| Stockfish compatibility | Exact Stockfish 18 SMALL official file loads; malformed-file, official-file, race, vet and build checks pass | Plan scalar full-refresh evaluation and an actual Stockfish intermediate-layer oracle |
| HCE speed investigation | Direct-search comparison has exact outcomes/PVs/final nodes; long-budget paired medians show about 2–3% overhead for one position | Further timing held while strength games use the machine |
| Leading Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 and Chess-3 v4.0 provisioned with protocol handshakes | Legal-search preflight and matched games remain |

## Prospective multicore test

Started **2026-09-06T13:03:25Z**, outer PID **2048736**. Output:

`output/m4c-hce8v1-sprt-20260906-attempt1` in the `ngn-m4c-sprt-runner` worktree.

Both sides use the same pinned 168385f binary. Candidate Threads/GOMAXPROCS=8; reference=1. Time control 30+0.3, Hash 128 MiB per engine, Move Overhead 100 ms, book off, one game at a time, full physical-core mask 0,2,4,6,8,10,12,14. The match has exclusive project compute through auditing and final sealing.

The predeclared test uses normalized-Elo SPRT hypotheses 0/20, alpha=beta=0.05, complete paired games, first-boundary-crossing stopping, and a cap of 400 games. A cap without crossing is inconclusive. It is not an ordinary +20 Elo confidence claim. Interim results cannot change the stopping rule.

The frozen runner passed 55 fixtures and 246 independent C++/Python SPRT vectors. Root accepted all 200 ordered openings and 1,200 legal opening plies. Approved manifest SHA:
`6d901b315382cf5d4ad77fe90869dbdbd82fc136f5244cdd2c08a9d02afc3c38`.

## Evaluator decisions

The minimal NGN-v1 architecture and training pipeline work numerically, but the trained networks have not become competitive with HCE. The 3M selected checkpoint improved fixed validation loss to 0.058133 and then lost four games by natural mate. That result triggered the prospective stop rule. More examples or updates alone are no longer the active strategy.

The next control uses a known pretrained Counter network with exactly reproduced source arithmetic. Root accepted its first slice in
`output/counter-pretrained-root-review-20260906/receipt-v1.json`.
The source is committed and integrated; incremental state and search adaptation are the next distinct implementation boundaries.

Stockfish support is an exact architecture port. The official SF18 SMALL model is verified at 3,519,630 bytes with SHA
`37f18f62d772f3107e1d6aaca3898c130c3c86f2ab63e6555fbbca20635a899d`.
The accepted loader has no evaluator yet. Root acceptance:
`output/sf18-small-loader-20260906/root-review-v1.json`.
Arbitrary Stockfish or other .nnue files are not supported.

The direct HCE diagnostic found substantially less overhead than the earlier short UCI timings: paired session median ratios 1.023 cold and 1.027 warm for the longer budget, on one position. These include public search admission and callback reporting. The separate HCE transition fast path remains unintegrated because measured effects are modest and fixture dependent.

## Remaining release gates

Complete useful multicore strength results, integrate and test the pretrained evaluator, establish an NNUE improvement over HCE, confirm the combined release at a longer clock, compare identified Go opponents under matched conditions, and deploy with rollback. NPS, validation loss, loader compatibility and short favorable scores do not satisfy those gates.

See the existing detailed evidence documents:
- `2026-09-05-next-stage-roadmap.md`
- `2026-09-06-smp-scaling-verdict.md`
- `2026-09-06-nnue-three-million.md`
- `2026-09-06-counter55-loader-oracle-result.md`
- `2026-09-06-hce-fastpath-diagnostic.md`
