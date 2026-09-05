# Review prompt: explain NGN's classical strength gap

First pass reviewed runtime `71399c063ee9688f499e71e1253ebaf81b211813`.
For a new pass, record the actual source commit, working diff and deployed binary
hash before starting. Current accepted playing source is `70eb527`; `4986330`
includes later offline fixes/docs. The separate uncapped-qsearch candidate has
its frozen patch and artifacts in `2026-09-04-qsearch-budget-repair.md`.
The objective is to identify reproducible correctness defects and substantial
implementation gaps relative to successful classical engines, especially Go
engines. Do not propose NNUE, a language rewrite, or novel chess research as the
escape route. The question is what NGN implements incorrectly, incompletely,
inefficiently, or with poorly validated interactions.

Read `TODO.md` and `CLAUDE.md`, then inspect executable code before accepting
claims in comments, reviews, or experiment summaries. Earlier audits declared
areas clean that subsequently contained proven defects. Historical evidence is
useful but is not an invariant. Preserve the deployed release and live matches.
Use isolated probes; do not apply speculative production changes during review.

## Reference sources

Use primary source code pinned to a revision. Local copies are gitignored under
`output/review-2026-09-04/`; retrieve these revisions if copies are unavailable:

- CounterGo `v1.38.0` / Counter 3.8:
  `b172b99b3128d82c44ef1d8d8a278aa09f2d721b`,
  https://github.com/ChizhovVadim/CounterGo
- Chess-3 `v4.0`: `a33531629cbe82eef6810f982ec814e79d52a3b3`,
  https://github.com/paulsonkoly/chess-3
- Blunder `v8.5.5`: `89230a74a966b3400610271eca49c4be241e94cd`,
  https://github.com/deanmchris/blunder

Verify evaluator type and engine version in source. Distinguish published
ratings, authors' estimates, and our local calibration. Do not compare ratings
across lists as if they shared a scale. Use other classical engines only when
they answer a specific unresolved question; pin their pre-NNUE revisions.

## Angle 1: chess rules and state integrity

Trace FEN/UCI input through move generation, make/unmake, search, and game
adjudication. Exercise castling through check, EP discovered checks, promotions,
underpromotions, fifty-move mate priority, repetition, and insufficient material.
Compare legal moves and rebuilt position/hash/accumulators with an independent
oracle after random legal sequences, not just a handful of perft roots.

Audit game-history versus search-history counts, irreversible boundaries and
null moves. Search-cycle draws are a heuristic distinct from game threefold.
Check same-position reentry through IID/singular search, array limits, and every
early return. A perft pass does not establish search correctness.

Compare **internal search BestMove and every completed root iteration** against
the independent legal set, before any UCI fallback or PV filtering. Audit every
consumer of pseudo-legal moves, not just `GenerateLegalMoves`. The September 4
loss study proved a castle-out-of-check root choice that a legal emitted-move
audit missed: the UCI guard substituted a legal move that hung a knight. Include
terminal positions with a nonempty pseudo-legal list and zero legal moves.

## Angle 2: search semantics and interactions

Audit alpha-beta/PVS windows, bound types, mate-distance encoding, aspiration
retry behavior, qsearch horizons, in-check behavior and terminal precedence.
Trace actual depth at each recursive call: reductions, all extension types,
full-depth verification, excluded moves, and depth/ply caps. Check whether the
stored TT depth describes the search actually performed. Inspect TT use across
different halfmove clocks/history, and qsearch reuse at differing capture caps.

Trace history/correction learning inputs and consumers, stop propagation, and
state restoration across normal children, null moves and same-ply searches.
Demonstrate an error with a concrete FEN/window/state or label it a heuristic
hypothesis. Differences from another engine alone do not establish a bug.

## Angle 3: why evaluation tuning failed

Reconstruct the real pipeline: data generation, quiet filtering, outcome labels,
deduplication, split, feature extraction, optimizer, integer export, loading and
the evaluator invoked by search. Verify exact trace/live-evaluator agreement on
diverse positions and perturbed parameters, not only original weights. Check
finite-difference gradients against the *playing model*, not merely against
another implementation of the same possibly incomplete surrogate.

Check dormant tunables; omitted features; frozen terms; material/PST
identifiability; phase interpolation; integer rounding amplified by feature
counts; draw scaling; score perspective; cache invalidation; and K calibration.
Check whether related positions from one game leak across train/validation and
whether holdouts were reused. Separate trainable feature limitations from data,
optimizer, integration, speed and search-interaction failures.

For each historical tune, recover its revision, dataset/objective, actual exported
and tested vector, and completed game evidence where available. Date any newly
found bug before blaming it for an older failure. Do not infer that classical
tuning cannot work, or that NNUE would work, from these failed attempts.

## Angle 4: compare complete classical mechanisms

Build a small matrix with NGN and reference file:line evidence. Compare formulas,
units, gates, update rules, representations and interactions—not feature names.
Prioritize move ordering and LMR together, TT replacement and reuse, quiescence,
time allocation, king danger, mobility, pawn structure, threats, and draw scaling.

For each proposed transplant, explain the missing behavior, why it matters in
NGN, prerequisites in the reference engine, and which existing NGN mechanism it
would replace. Include subtraction/simplification as candidates. Check
`experiments/TRIED-LEDGER.md` and later run records before proposing old work.
An old negative experiment may be reopened only with a specific new falsifiable
premise, not because its verdict is inconvenient.

## Angle 5: Go runtime and usable search

Compare work per node, allocation/GC, frame storage, move generation/staging,
SEE work, hashing, caches, locks, diagnostics and time checks. Inspect hardcoded
node/depth limits and root exits that may waste clock at longer time controls.
Measure on equivalent hardware/settings; raw NPS and nominal depth across
engines are not strength measurements. Distinguish exact speed improvements
from pruning that merely makes the tree smaller. Check current native workers
before remote compute; do not load the Windows/WSL box during timed matches.

## Angle 6: the measurement and development process

Independently inspect the match harness, score POV, clocks, process failures,
opening/color pairing, terminal adjudication, PGN completeness, and exclusions.
Check whether self-play conclusions transfer to external opponents and longer
clocks. Identify completed experiments left unconfirmed, stale binaries, tests
that mirror implementation, and lane closures made from invalid evidence.
Do not turn every review into a new testing framework or another long tune.

## Required output

1. Ranked findings with classification: reproduced defect, code-proven mismatch,
   measured performance issue, heuristic hypothesis, or falsified suspicion.
2. Each finding includes NGN file:line, exact reproduction or falsifier, reference
   source/revision when relevant, impact and limits, and whether it was already
   known. Distinguish newly discovered bugs from the unshipped T17 repair.
3. A compact mechanism comparison, including what stronger engines *omit*.
4. A tuning failure account that distinguishes proved causes from missing records.
5. At most three next experiments, with prerequisites, the cheapest decisive
   check, game acceptance rule, compute cap and stopping condition. No invented
   Elo priors, summed gains, or unsupported claim that the engine is now clean.

Give the strongest evidence against your preferred explanation. A useful review
narrows the next implementation decision and supplies executable evidence.
