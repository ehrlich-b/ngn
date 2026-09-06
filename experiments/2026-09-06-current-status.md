# Current NNUE / multicore sprint status

Updated 2026-09-06. All source work, builds, tests, engines, training and profiling run on WSL. The Mac is transport only.

Latest production engine in /home/ehrli/repos/ngn-next: 168385f (real helper search integrated); later commits record review evidence. Deployed /home/ehrli/repos/ngn remains 53e4d1b. No new playing-strength gain has been established; the historical 2884.14 rating is from the previous native-Windows conditions, not a fresh WSL measurement.

| Area | Accepted evidence / current state | Next decision |
|---|---|---|
| WSL baseline | Corrected B0 A/A completed 200 legal, naturally terminated games; accepted independent replay and control receipts | Preserve fixed runner/artifacts |
| Search ownership | Private HCE caches, history and NNUE context; one owned TT; persistent idle worker pool configured 1–64 | Measure actual 1/2/4/8-thread behavior and equal-time strength |
| Lifecycle | Joined/cancellable UCI sessions; stop flag published before cancellation notification; Move Overhead implemented | Exercise complete helper lifecycle, deadline and panic cases |
| NNUE inference | Strict NGN-v1 format, independent raw/quantized export bridge, full scalar and incremental oracles, portable int32 contexts, bounded AVX2 output kernel with exact fallback | Measured AVX2 output ~8.7x; warm whole-NNUE search 1.52–1.66x versus scalar with identical outcomes/nodes; profiling remains |
| NNUE selection | Transactional EvalFile/EvalBackend controls; failed replacement preserves usable selection; new-game persists it | Both 100k and 1M four-game selected-backend smokes pass operational checks |
| Opening books | OwnBook default true preserves historical behavior; false bypasses both external and embedded books | Require false in evaluator/SMP tests |
| Training pipeline | Real-data pilot accepted: 80,311 training positions, 2,097,152 presentations, five candidates; all export/integer parity checks pass | 1M 128/512-update runs accepted; update 128 remains best and later validation worsens. The 1M candidate lost its four-game smoke 0–4; a fixed-holdout 3M total-position expansion is running. |
| Pilot network | Checkpoint 1 selected on validation; NGN SHA abd956983274bae1d91b522cd7ca3a1b2fa91ef3e69051ccb414b9adf0a5c67e | Four-game smoke lost 0–4 to HCE; operationally valid, no promotion; broaden verified corpus |
| Multicore execution | Actual joined helper search integrated; full combined short/race/vet/build/reference gates pass; real simultaneous NNUE transitions and partial-launch cleanup tested | 128-sample 1/2/4/8 HCE matrix accepted: 8 threads 6.41–6.69x aggregate NPS, median primary depth +2/+1 cold/warm; equal-wall games next |
| Candidate runner | Separate diagnostic runner accepted after 24 fixtures and four fully audited games | Legacy source/tests integrated at dbe08bf. M4c frozen v2 source accepted by root after 39 fixtures and real 1/2/4/8 preflights; four-game 8v8 control awaits the data job terminal. |
| Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 provisioned/handshaken; Chess-3 v4.0 built with prescribed PGO profiling | All four protocol handshakes pass; legal-search preflights and matched-condition games remain |
| Stockfish compatibility | Exact backend compatibility ladder researched; arbitrary .nnue files unsupported | Separate exact-architecture backend later; no compatibility handwave |

The portable-inference/worker combined tree 266da785 passed full short tests, full short race tests, vet, build and exact corrected-B0 HCE search snapshots (only source/build metadata differed). AVX2 passed independent lane/order/overflow/rounding/reference tests, v1 fallback, v3 race and architecture cross-build checks. The subsequent OwnBook slice passed full engine short/race tests. The actual helper-search combined tree passed full short/race/vet/build and corrected-B0 snapshots; root verified all seven integrated files match the validated source. A later test-only amendment passed focused race coverage with all three NNUE workers blocked inside actual pushed frames.

Sol owns major implementation. Root coordinates architecture, independent review, integration, experiment acceptance and WSL compute scheduling. No new playing-strength gain has been established.

The immediate sequence is:

1. Finish the reviewed 3M TOTAL retained-position data expansion (about 2.4M expected training examples). Preserve the original validation bytes and exclude every eligible key from the old validation/sealed chains from new training. Expanded holdout files are audit-only.
2. Run the reviewed four-game HCE8-vs-HCE8 operational control at 30+0.3, total Hash128 per role, Move Overhead100, concurrency one and the full eight-physical-core mask. Start only after the data wrapper and final hashes actually exit.
3. Freeze and validate the prospective HCE8-vs-HCE1 paired strength test. A source-only plan is examining SPRT early stopping and variable completed-game audit requirements.
4. If data conversion and leakage audits pass, prepare a separate 512-update training request using the expanded training set and the same original validation set. Training and clock-controlled games get separate compute windows.

Both pilot selected networks lost four-game smokes 0–4 to HCE, with all operational and chess audits passing. Material diagnostics show that the 1M model learned more plausible piece scale than the 100k model, but it remains weak. Extending 1M training from 128 to 512 updates worsened validation after update 128, so further count-only extension is rejected. See 2026-09-06-nnue-one-million.md.

The first 3M synthetic rejection harness contained a false-pass path: it could catch its own unexpected-success error. V1 is preserved. Root reviewed the narrow v2 repair and verified one positive case, four real rejections, and four always-success-stub guards. The exact bounded data-only v2 run is now authorized and running on CPUs 12,14. No training is included in that approval.

The 128-sample HCE scaling matrix is accepted: eight threads deliver 6.41–6.69x aggregate NPS and median primary depth +2/+1 for cold/warm probes. All samples restored root state and joined workers. This is scaling evidence, not a strength verdict; see 2026-09-06-smp-scaling-verdict.md.

The HCE transition bypass has accepted source/correctness evidence, including dynamic NNUE scratch overflow handling, but remains unintegrated. Its short wall-time effects are modest and fixture-dependent. Normal UCI nodes expose only the last completed iteration, so the original exact-budget timing gate failed and the runs were explicitly downgraded to diagnostics.

The subsequent exact 4bd54add-to-168385f comparison does not reproduce the large earlier middlegame gap: the short middlegame pooled median ratio is 0.986; the larger 524,288-requested-node probe is 1.012, with seven paired medians spanning 0.970–1.014. Both runs have zero normalized outcome mismatches and clean termination. This does not support blaming the multicore boundary. Earlier B0-to-current short wall differences still require separation of search work from setup, pacing and runtime scheduling; see 2026-09-06-hce-fastpath-diagnostic.md.

Root accepted the frozen M4c runner source after reading production changes and rejection tests, verifying 48 evidence artifacts, all 22 frozen source files, 16 control input identities, and actual 1/2/4/8 preflight receipts. Its four-game control remains an operational gate, with no rating claim.
