# Current NNUE / multicore sprint status

Updated 2026-09-06. All source work, builds, tests, engines, training and profiling run on WSL. The Mac is transport only.

Integration: /home/ehrli/repos/ngn-next at 168385f (real helper search integrated). Deployed /home/ehrli/repos/ngn remains 53e4d1b. No new playing-strength gain has been established; the historical 2884.14 rating is from the previous native-Windows conditions, not a fresh WSL measurement.

| Area | Accepted evidence / current state | Next decision |
|---|---|---|
| WSL baseline | Corrected B0 A/A completed 200 legal, naturally terminated games; accepted independent replay and control receipts | Preserve fixed runner/artifacts |
| Search ownership | Private HCE caches, history and NNUE context; one owned TT; persistent idle worker pool configured 1–64 | Measure actual 1/2/4/8-thread behavior and equal-time strength |
| Lifecycle | Joined/cancellable UCI sessions; stop flag published before cancellation notification; Move Overhead implemented | Exercise complete helper lifecycle, deadline and panic cases |
| NNUE inference | Strict NGN-v1 format, independent raw/quantized export bridge, full scalar and incremental oracles, portable int32 contexts, bounded AVX2 output kernel with exact fallback | Measured AVX2 output ~8.7x; warm whole-NNUE search 1.52–1.66x versus scalar with identical outcomes/nodes; profiling remains |
| NNUE selection | Transactional EvalFile/EvalBackend controls; failed replacement preserves usable selection; new-game persists it | Both 100k and 1M four-game selected-backend smokes pass operational checks |
| Opening books | OwnBook default true preserves historical behavior; false bypasses both external and embedded books | Require false in evaluator/SMP tests |
| Training pipeline | Real-data pilot accepted: 80,311 training positions, 2,097,152 presentations, five candidates; all export/integer parity checks pass | 1M corpus and nine-checkpoint training accepted; validation improved through final checkpoint; 1M candidate lost four-game smoke 0–4 |
| Pilot network | Checkpoint 1 selected on validation; NGN SHA abd956983274bae1d91b522cd7ca3a1b2fa91ef3e69051ccb414b9adf0a5c67e | Four-game smoke lost 0–4 to HCE; operationally valid, no promotion; broaden verified corpus |
| Multicore execution | Actual joined helper search integrated; full combined short/race/vet/build/reference gates pass; real simultaneous NNUE transitions and partial-launch cleanup tested | 128-sample 1/2/4/8 HCE matrix accepted: 8 threads6.41–6.69x aggregate NPS, median primary depth+2/+1 cold/warm; equal-wall games next |
| Candidate runner | Separate diagnostic runner accepted after 24 fixtures and four fully audited games | Source/tests integrated at dbe08bf; extend explicit capability profile for M4c |
| Go opponents | Counter 5.5, Zahak 10, Blunder 8.5.5 provisioned/handshaken; Chess-3 v4.0 built with prescribed PGO profiling | All four protocol handshakes pass; legal-search preflights and matched-condition games remain |
| Stockfish compatibility | Exact backend compatibility ladder researched; arbitrary .nnue files unsupported | Separate exact-architecture backend later; no compatibility handwave |

The portable-inference/worker combined tree 266da785 passed full short tests, full short race tests, vet, build and exact corrected-B0 HCE search snapshots (only source/build metadata differed). AVX2 passed independent lane/order/overflow/rounding/reference tests, v1 fallback, v3 race and architecture cross-build checks. The subsequent OwnBook slice passed full engine short/race tests. The actual helper-search combined tree passed full short/race/vet/build and corrected-B0 snapshots; root verified all seven integrated files match the validated source. A later test-only amendment passed focused race coverage with all three NNUE workers blocked inside actual pushed frames.

Sol owns the major implementation tracks. Root owns architecture decisions, independent review, integration, experiment acceptance and WSL compute scheduling. Inference speed and the initial operational games are measured; see 2026-09-06-nnue-performance-verdict.md and 2026-09-06-nnue-candidate-smoke.md. The 1M corpus and learning run are now independently accepted. The equal-time 1/2/4/8 diagnostic is also accepted. Next are the count-only learning follow-up and prospective playing-strength tests. Training loss and NPS are diagnostic evidence, not completion criteria.

The external B0/current HCE short-search run had 700 exact matching outcomes, but its wall timings include legacy B0 handleGo pacing: a 10 ms sleep delays consumption of the next queued command. Those ratios are UCI pipeline diagnostics, not isolated HCE speed evidence. The corrected readyok-barrier run is complete and exact; current/B0 short-search wall ratios are 1.014–1.325, so the HCE regression needs a measured correction. CPU profiles have since completed and identify avoidable NNUE transition work on the HCE path.

The 1M learning result and second game diagnostic are recorded in 2026-09-06-nnue-one-million.md. HCE profiles identify avoidable NNUE transition-record work on the HCE path; a narrow bypass is under Sol correctness/compiler validation. No candidate has been promoted.

Material diagnostics are independently accepted: 10 models × 18 FENs have exact integer-Go parity. The 1M net learned much more plausible material scale than the 100k pilot, but substantial positional imbalances remain. A 512-update follow-up uses the same corpus and architecture; only the schedule changes, motivated by validation improving through update 128. The 512-update run is complete and independently accepted: validation selected update128, then worsened throughout. More schedule extension is rejected; a3M data-only proposal is being prepared with fixed validation and explicit leakage exclusion. The SMP matrix is complete.

See 2026-09-06-smp-scaling-verdict.md for the first accepted multicore scaling result. HCE fast-path v2 passed its ordinary gates but root found an unproven invocation-frame bound; v3 now passes NNUE-only scratch growth, actual overflow-search regression, full short/race/vet/build and exact corrected-B0 snapshots. Root accepts source/correctness; integration awaits measured performance. The frozen SMP diagnostic used the previously validated168385f engine.
