# Current NNUE / multicore sprint status

Updated 2026-09-06. All source work, builds, tests, engines, training and profiling run on WSL. The Mac is transport only.

Integration: /home/ehrli/repos/ngn-next at 4bd54add148f5b3fb53675f4f107d46d5e3fc69e. Deployed /home/ehrli/repos/ngn remains 53e4d1b. No new playing-strength gain has been established; the historical 2884.14 rating is from the previous native-Windows conditions, not a fresh WSL measurement.

| Area | Accepted evidence / current state | Next decision |
|---|---|---|
| WSL baseline | Corrected B0 A/A completed 200 legal, naturally terminated games; accepted independent replay and control receipts | Preserve fixed runner/artifacts |
| Search ownership | Private HCE caches, history and NNUE context; one owned TT; persistent idle worker pool configured 1–64 | Review actual helper-search implementation |
| Lifecycle | Joined/cancellable UCI sessions; stop flag published before cancellation notification; Move Overhead implemented | Exercise complete helper lifecycle, deadline and panic cases |
| NNUE inference | Strict NGN-v1 format, independent raw/quantized export bridge, full scalar and incremental oracles, portable int32 contexts, bounded AVX2 output kernel with exact fallback | Isolated kernel and whole-search profiling |
| NNUE selection | Transactional EvalFile/EvalBackend controls; failed replacement preserves usable selection; new-game persists it | Candidate playing preflight |
| Opening books | OwnBook default true preserves historical behavior; false bypasses both external and embedded books | Require false in evaluator/SMP tests |
| Training pipeline | Real-data pilot accepted: 80,311 training positions, 2,097,152 presentations, five candidates; all export/integer parity checks pass | Broaden corpus after controlled diagnostic; do not select on training loss |
| Pilot network | Checkpoint 1 selected on validation; NGN SHA abd956983274bae1d91b522cd7ca3a1b2fa91ef3e69051ccb414b9adf0a5c67e | Operational NNUE-vs-HCE games, then a larger verified corpus |
| Multicore execution | Worker-pool groundwork integrated; actual joined helper search passed first race/reference gates; partial-launch repair under final combined review | Correctness/race gates before scaling matches |
| Candidate runner | Separate diagnostic runner reviewed; operational-error and cleanup corrections under Sol implementation | Fixture review, then tiny paired full-game operational smoke |
| Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 provisioned/handshaken; Chess-3 v4.0 built with prescribed PGO profiling | All four protocol handshakes pass; legal-search preflights and matched-condition games remain |
| Stockfish compatibility | Exact backend compatibility ladder researched; arbitrary .nnue files unsupported | Separate exact-architecture backend later; no compatibility handwave |

The portable-inference/worker combined tree 266da785 passed full short tests, full short race tests, vet, build and exact corrected-B0 HCE search snapshots (only source/build metadata differed). AVX2 passed independent lane/order/overflow/rounding/reference tests, v1 fallback, v3 race and architecture cross-build checks. The subsequent OwnBook slice passed full engine short/race tests. Full integrated checks will run again with actual helper search.

Sol owns the major implementation tracks. Root owns architecture decisions, independent review, integration, experiment acceptance and WSL compute scheduling. The next empirical gates are an isolated inference/whole-search performance comparison and audited operational games, followed by predeclared strength and equal-time 1/2/4/8-thread tests. Training loss and NPS are diagnostic evidence, not completion criteria.
