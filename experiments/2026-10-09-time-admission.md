# Immediate iteration admission — session19 topic candidate

Accepted-runtime diagnostic: twelve warm KB8/be34532c timed120+1 searches,
CPU13,15 / nice19 / GOMAXPROCS1, sourcea49ebfc. All twelve stop by soft abort
inside an active iteration. Aggregate post-completion tail **24.14%**,
20.987s of86.954s; median18.88%. Every result is legal and restores root state.
[Receipt](2026-10-09-session19-results/time-trace-summary.json).
This measures actual searches with fresh120s banks, not bank trajectories in
full games or an Elo benefit. The old borrowed-evaluator C0 result is not reused.

One isolated time-policy change on `night/time-admission` froma49ebfc:
check the clock immediately at every new-depth admission. Keep the existing
soft formula, previous-iteration projection, active1024-call throttle, active
soft abort, hard/emergency limits, incomplete-iteration discard, and moves-to-go
formula. It does not repeat the old C1 package that removed active soft stops.
No TT, source scout-depth, evaluator, net or search-parameter change is stacked.

Deterministic tests require expired-soft and hard admissions to fail the base
and pass the candidate at counter0; an early admission reads the current clock
and continues; active checks remain throttled1024 calls with their existing
soft abort. Run engine/full short/race and vet, plus the same12 timed probes
for a mechanism comparison; no threshold on that noisy trace selects a winner.
Topic files remain uncommitted until the real-clock strength verdict.

Games are **not launched** while factorizer/TT verdicts are unresolved.
Freeze the actual accepted source/net before one fresh A/A40 and paired real-clock SPRT[0,+6] (alpha/beta0.05, minimum400 games, cap4000)
at **60+0.6**, concurrency8 / CPU0–11 / nice19 / GOMAXPROCS1,
Hash64 / Threads1 / OwnBook=false, existing974e4b5a openings and9b0002f8 harness.
Accept iff a completed H1 decision, penta lower95>0, zero flags/crashes/illegal/
no-move outcomes, exact pairs and intact inputs. H0/inconclusive cap: shelve
without extension or sign-only adoption.
No10+0.1 proxy can accept this time candidate. Record exact binaries/commands
before games, preserve queued CUDA checks and drain/restore unfinished old-base
SPSA only at a durable checkpoint for the game window.

Priorities: finish the audit-derived TT candidate first; broader data training
is ahead of any optional architecture trial. Other time changes are separate
future candidates and are not included in this patch.

CPU verification **PASS** at00:34:39: both boundary regressions fail baseline
and pass candidate; engine/full short/race, vet and PGO builds pass. Local
source equals the tested source. Same12 warm probes now have mechanism data
in the [review](2026-10-09-session19-results/time-review.json); no strength
games are launched. Before any games, the prospective rule is powered SPRT
[0,+6]/cap4000, following the merged plan's small-change measurement item.

The candidate trace has10 soft aborts and2 soft admissions, aggregate tail
**20.24%** (17.742/87.677s), median16.21%. The one-condition admission fix
recovers only part of the observed tail in this sample. Keep it as a distinct
candidate; do not add an EBF predictor or remove active soft stops without
a separate declaration. Games, not this noisy comparison, decide its strength.
