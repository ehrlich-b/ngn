# Counter correction-history interaction

Active thread goal: **raise the rating to 3300**. This experiment is a step
toward that objective, not a replacement completion condition. Use one-thread
external strength as the working configuration; absolute completion will require
fresh, configuration-matched rating evidence, not adding internal Elo estimates.
The earlier roughly 3000–3100 orientation remains uncertain and unchanged.

Immediate base: `270c73563139996d03f3c34b2840015b8e130df9`. Production retains
the accepted output AVX2 kernel; the separately archived copy/update fusion is
not included. The previous goal turn counts as progress because its completed
measurement rejected the next speed candidate under its declared rule and
changed the ranked next action. It produced no accepted engine gain.

## Bounded next step

Sol agents own source analysis, diagnostic instrumentation and independent
measurement review. All builds, engines and tests remain on authorized WSL.
No own-network training, deployment, broad tuning or new absolute calibration
is part of this initial diagnostic.

Inspect the existing three-term correction family with the Counter evaluator:
where it is applied, what targets update it, how TT refinement interacts, and
which pruning/reduction decisions consume it. Reuse the deterministic first
three accepted Counter1 game histories at plies 23 and 48, retaining one engine
across both searches per game. This is sparse-prefix retained-state replay,
not a reconstruction of every timed game search.

The diagnostic must preserve baseline search identity and distinguish raw
adapted eval, the three correction contributions, corrected static, TT-refined
static and downstream comparisons. Its purpose is to justify one coherent
Counter-only family experiment, not select a parameter by its sampled score.
Do not stack a correction change with TT, futility, extension or LMR changes.

Any candidate touching evaluation/search decisions requires a fresh identical-
configuration A/A and a predeclared real-clock paired game gate against this
immediate base. Freeze its source, inputs, environment, sample and decision rule
before candidate games. Reuse the audited runner where possible; preserve failed
attempts and never extend a sample after seeing its result. Diagnostic node
counts, choices and reference scores cannot establish playing strength.

## Status

The passive diagnostic is complete. Root admitted one Counter-only minor-
correction-off candidate for implementation and a separately frozen game gate.
Sol owns implementation and independent review. No games have started; the
3300 goal remains active and unproved.

Source review finds no correctness defect or obviously excessive scale. Exact
integer saturation is ±48 cp per term, ±144 cp total; the three tables learn
the same residual against the already-corrected pre-TT static. This prevents
the simplistic claim that each independently learns the full raw error and
triples it. The narrower concern is overlap between minor-piece and non-pawn
keys; the classical minor-term gain was only about +3.7 Elo, with no proved
Counter-specific harm. An ordinary nonzero contribution alone does not establish
a reason to change production policy.

The passive diagnostic reports aggregate magnitudes and local comparison
changes on the baseline's reached nodes. These are not an alternate search
trajectory or evidence of stronger moves. Sol is separately implementing inactive
compatibility for the exact existing Rodent V1.1 Anand network in parallel.

The admitted candidate's reviewed game plan is the existing 10+0.1,
concurrency-four, one-thread-per-engine protocol:
fresh AA100 then fixed candidate400, adopting only with positive paired95
lower bound and all integrity checks. No games have been authorized yet.

## Completed diagnostic

The single monitored run passed in 4.317 seconds, with empty stderr, no survivors
and 329,728 KiB peak sampled process-tree RSS. All six control/instrumented
trajectories exactly match each other and the frozen untouched baseline.
Games 1 and 2 share the ply-23 position/trajectory: these are six declared
observations, not six independent positions. No sample was substituted.

Across 1,884,322 reached static observations, the minor term is nonzero on 83.3%
and has mean absolute magnitude 14.92 cp, versus 9.55 cp pawn and 3.53 cp
non-pawn. Aggregate correction mean magnitude is 27.10 cp, with no table entry
at its saturation limit. Holding the reached path, actual improving value and
other eligibility conditions fixed, minor removal changes 37,024 RFP, 16,879
NMP and 57,245 qsearch beta-stand-pat comparisons. These counts are local
comparison opportunities, including comparisons after which an earlier
pruning return could prevent execution—not actual alternate-branch counts.

The [frozen summary](2026-09-13-counter-correction-artifacts/diagnostic/summary.json)
changes the priority assessment: minor influence is not negligible in this
sample. It supplies no evidence of better moves or Elo direction. Across all
five distinct sampled root trajectories, mean absolute minor contribution is
7.74–23.86 cp and local qsearch beta-comparison changes span 3.16–10.92%.
The effect is not confined to the duplicated trajectory. Root judged this
materiality plus the structural key-overlap hypothesis sufficient to admit one
real-clock minor-only ablation. The Sol measurement agent recommended awaiting
directional evidence; root distinguishes admission to a falsifiable game
experiment from adoption of the change.

The candidate disables only the Counter backend's minor correction application
and learning. Pawn and non-pawn correction, other evaluator backends, table
storage/lifecycle, TT policy, pruning, timing, SMP and the output kernel remain
unchanged. No new UCI option or parameter sweep is admitted. All owned builds
and oracle checks must finish before the timed A/A begins. A failed or
inconclusive fixed gate shelves this candidate without extension.

Full raw observations remain WSL at
`/home/ehrli/repos/ngn-correction-profile-20260913/output/evidence-v1/correction-profile.json`,
SHA-256 `545462126185600889cedf50ebafc08915be5f1f829be2ca7edd80c6287cda66`.
The compact archive includes observer source, commands, summarizer, summary
and supervisor receipt. Its `wsl-SHA256SUMS` inventories the full WSL bundle;
the raw JSON and unchanged helper probes are intentionally not duplicated.
