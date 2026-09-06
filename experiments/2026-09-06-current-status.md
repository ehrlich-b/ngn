# Current NNUE / multicore sprint status

Updated 2026-09-06. All source work, builds, tests, engines, training and profiling run on WSL. The Mac is transport only.

Latest production engine in /home/ehrli/repos/ngn-next: 168385f (real helper search integrated); later commits record review evidence. Deployed /home/ehrli/repos/ngn remains 53e4d1b. No new playing-strength gain has been established; the historical 2884.14 rating is from the previous native-Windows conditions, not a fresh WSL measurement.

| Area | Accepted evidence / current state | Next decision |
|---|---|---|
| WSL baseline | Corrected B0 A/A completed 200 legal, naturally terminated games; accepted independent replay and control receipts | Preserve fixed runner/artifacts |
| Search ownership | Private HCE caches, history and NNUE context; one owned TT; persistent idle worker pool configured 1–64 | Measure actual 1/2/4/8-thread behavior and equal-time strength |
| Lifecycle | Joined/cancellable UCI sessions; stop flag published before cancellation notification; Move Overhead implemented | Exercise complete helper lifecycle, deadline and panic cases |
| NNUE inference | Strict NGN-v1 format, independent raw/quantized export bridge, full scalar and incremental oracles, portable int32 contexts, bounded AVX2 output kernel with exact fallback | Measured AVX2 output ~8.7x; warm whole-NNUE search 1.52–1.66x versus scalar with identical outcomes/nodes; profiling remains |
| NNUE selection | Transactional EvalFile/EvalBackend controls; failed replacement preserves usable selection; new-game persists it | 100k and 1M four-game smokes passed operational checks but each lost 0–4; 3M selected c6 also lost its separate audited screen 0–4; stop minimal count-only extensions |
| Opening books | OwnBook default true preserves historical behavior; false bypasses both external and embedded books | Require false in evaluator/SMP tests |
| Training pipeline | Real-data pilot accepted: 80,311 training positions, 2,097,152 presentations, five candidates; all export/integer parity checks pass | 3M data and 512-update training accepted: 2,406,580 training positions, unchanged original validation, c6/update384 selected at loss 0.058133 versus prior 1M best 0.067230. Selected c6 subsequently lost its four-game screen 0–4; validation improvement did not establish competitiveness. |
| Pilot network | Checkpoint 1 selected on validation; NGN SHA abd956983274bae1d91b522cd7ca3a1b2fa91ef3e69051ccb414b9adf0a5c67e | Four-game smoke lost 0–4 to HCE; operationally valid, no promotion; broaden verified corpus |
| Multicore execution | Actual joined helper search integrated; full combined short/race/vet/build/reference gates pass; real simultaneous NNUE transitions and partial-launch cleanup tested | 128-sample 1/2/4/8 HCE matrix accepted: 8 threads 6.41–6.69x aggregate NPS, median primary depth +2/+1 cold/warm; equal-wall games next |
| Candidate runner | Separate diagnostic runner accepted after 24 fixtures and four fully audited games | Legacy source/tests integrated at dbe08bf. M4c frozen v2 source accepted by root after 39 fixtures and real 1/2/4/8 preflights; four-game 8v8 control accepted: 562 legal plies, all per-go width receipts exact, no operational failures. Prospective 8v1 SPRT implementation remains under review. |
| Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 provisioned/handshaken; Chess-3 v4.0 built with prescribed PGO profiling | All four protocol handshakes pass; legal-search preflights and matched-condition games remain |
| Stockfish compatibility | Exact backend compatibility ladder researched; arbitrary .nnue files unsupported | Separate exact-architecture backend later; no compatibility handwave |

The portable-inference/worker combined tree 266da785 passed full short tests, full short race tests, vet, build and exact corrected-B0 HCE search snapshots (only source/build metadata differed). AVX2 passed independent lane/order/overflow/rounding/reference tests, v1 fallback, v3 race and architecture cross-build checks. The subsequent OwnBook slice passed full engine short/race tests. The actual helper-search combined tree passed full short/race/vet/build and corrected-B0 snapshots; root verified all seven integrated files match the validated source. A later test-only amendment passed focused race coverage with all three NNUE workers blocked inside actual pushed frames.

Sol owns major implementation. Root coordinates architecture, independent review, integration, experiment acceptance and WSL compute scheduling. No new playing-strength gain has been established.

The immediate sequence is:

1. Diagnose the B0-to-4bd HCE interval using the accepted direct-search harness, actual final node counts and nonempty legal PVs. Its one-position/two-budget run is independently accepted: 140 rows, exact outcomes/PVs/final nodes, long-search paired session median ratios 1.023 cold and 1.027 warm. Public admission and callback reporting are included; this is not recursion alone.
2. The 3M c6 four-game screen is independently accepted operationally and lost 0–4 to HCE. Its predeclared stop rule now ends count-only extensions of this minimal architecture. Counter pretrained compatibility is the next evaluator path.
3. Finish independent math, rejection and variable-completed-game audits for the prospective HCE8-vs-HCE1 SPRT. The accepted plan uses normalized Elo hypotheses 0/20, alpha/beta 0.05, first completed-pair boundary stopping and a 400-game cap; no 8v1 game has launched.
4. Implement the separately reviewed Counter 5.5 strict loader and portable full-refresh oracle slice. The exact official pretrained model is available. Incremental state, search integration, optimization and games follow separate evidence gates. The Stockfish18 small/big compatibility work remains separate.

The HCE8-vs-HCE8 control completed with one win, one loss and two draws (two mates, two insufficient-material draws). All 562 plies were legal; 538 per-go 8/8 width receipts equal the played plies minus 24 opening plies. Root checked 14,008 live child identity observations and all 85 final artifacts. Actual outer exit was zero and all five supervised stages had no survivors. See output/m4c-runner-root-review-20260906/control-review-v1.json (SHA e4f607bcfdfdc1cb324187ba544c23bf7bf6dbe789bee87765527e85dce403b0).

The 3M training run consumed exactly 512 updates and 8,388,608 presentations. All 44 stages completed, all 18 exported-artifact parity checks matched Go integer inference on 16 frozen FENs each, and all nine raw/quantized NGN file pairs were byte-identical. Fixed original validation selected c6/update384 before any games. Root verified 11,579 files totaling 917,126,901 bytes. See 2026-09-06-nnue-three-million.md. This is evidence of better validation fit and a working pipeline; chain-disjoint records do not prove original-game-disjoint holdouts.

Both pilot selected networks lost four-game smokes 0–4 to HCE, with all operational and chess audits passing. Material diagnostics show that the 1M model learned more plausible piece scale than the 100k model, but it remains weak. Extending 1M training from 128 to 512 updates worsened validation after update 128, so further count-only extension is rejected. See 2026-09-06-nnue-one-million.md.

The first 3M synthetic rejection harness contained a false-pass path: it could catch its own unexpected-success error. V1 is preserved. Root reviewed the narrow v2 repair and verified one positive case, four real rejections, and four always-success-stub guards. The exact bounded data-only v2 run completed on CPUs 12,14 and is independently accepted. No training was included in that approval.

The 128-sample HCE scaling matrix is accepted: eight threads deliver 6.41–6.69x aggregate NPS and median primary depth +2/+1 for cold/warm probes. All samples restored root state and joined workers. This is scaling evidence, not a strength verdict; see 2026-09-06-smp-scaling-verdict.md.

The HCE transition bypass has accepted source/correctness evidence, including dynamic NNUE scratch overflow handling, but remains unintegrated. Its short wall-time effects are modest and fixture-dependent. Normal UCI nodes expose only the last completed iteration, so the original exact-budget timing gate failed and the runs were explicitly downgraded to diagnostics.

The subsequent exact 4bd54add-to-168385f comparison does not reproduce the large earlier middlegame gap: the short middlegame pooled median ratio is 0.986; the larger 524,288-requested-node probe is 1.012, with seven paired medians spanning 0.970–1.014. Both runs have zero normalized outcome mismatches and clean termination. This does not support blaming the multicore boundary. Earlier B0-to-current short wall differences still require separation of search work from setup, pacing and runtime scheduling; see 2026-09-06-hce-fastpath-diagnostic.md.

Root accepted the frozen M4c runner source after reading production changes and rejection tests, verifying 48 evidence artifacts, all 22 frozen source files, 16 control input identities, and actual 1/2/4/8 preflight receipts. Its four-game control is accepted as an operational gate, with no rating claim.

The 3M c6 screen completed 438 legal plies and four natural HCE checkmates. Root verified 86 final artifacts totaling 129,400,199 bytes, every one of 4,000 live child identity observations, five clean supervised stages, no canonical errors/warnings or embedded-book signatures, and no survivors. Receipt: output/candidate-smoke-3m-root-review-20260906/receipt.json, SHA 0f38118dd4ddfa1609248034497758d1b91e7070e63ae5e7f0e5c103f31b4ea2. No Elo estimate follows from four games.
