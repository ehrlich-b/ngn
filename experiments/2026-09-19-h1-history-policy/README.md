# H1 completed-winner history policy gate

Status: the A/A and first two factorial cells completed cleanly. During the
completed direct 11/00 pilot, the controller's telemetry-only `/proc` sampler
hit a transient `ProcessLookupError`; the separately supervised match still
finished with zero exits, no violations and no survivors. Its score remains
uninspected; the byte-frozen, no-replay recovery passed its held check and
static review, with release promotion pending.

H0 passed its predeclared mechanism screen. H1 therefore tests exactly the
three-component package specified in the Sol plan: completed final-quiet-winner
labels (including root/PV attempts), the `H=8192` response
`(target-h)*min(depth²,400)/512`, and the normalized LMR term
`-clamp(history/2500,-2,+2)`. The main and predecessor keys, history-prune
threshold, LMR base and other modifiers, killer/counter policies, capture
history, evaluator and clock remain fixed.

Four compile-time policies are immutable:

| Policy | Producer | LMR consumer |
| --- | --- | --- |
| 00 | current cutoff-only gravity | current step term |
| 10 | completed-winner response | current step term |
| 01 | current cutoff-only gravity | normalized term |
| 11 | completed-winner response | normalized term |

The default build is policy 00. Experimental tags are `h1producer` and
`h1consumer`; no runtime option can mix an unreviewed policy.

## Frozen game protocol and decision

All roles use Rodent V1.2 default, Threads1, `GOMAXPROCS=1`, Hash128,
OwnBook=false, Move Overhead100, 10+0.1, concurrency four on physical CPU masks
0/2/4/6, no ponder/tablebases/adjudication, and the accepted candidate-match
supervision plus legal/terminal audit. A fresh 50-pair policy-00 A/A must have a
100,000-resample whole-pair bootstrap interval containing zero.

The factorial pilot then runs these fixed 64-pair, color-reversed comparisons
on the same sequential canonical openings 1–64:

1. policy 10 versus 00;
2. policy 11 versus 01;
3. policy 11 versus 00.

The first two estimate the producer effect under each consumer. Their
interaction is `Elo(11/01) - Elo(10/00)`, bootstrapped by resampling the same
opening indices jointly. This classification cannot select a different policy
or constant. If the direct 11/00 pilot's upper 95% bound is below zero, the
package is clearly bad and is shelved before confirmation. Otherwise policy 11
versus 00 proceeds regardless of the other pilot cells.

Confirmation is a fixed 200-pair match on independent canonical source lines
201–400, converted before scores with the pinned converter. Policy 11 is
accepted only if all operational/legal gates pass and its whole-pair bootstrap
lower 95% Elo bound is strictly positive. Otherwise it is shelved at the cap.
There is no score-based extension, best-cell selection, divisor tuning or reuse
of pilot openings in confirmation.

The A/A, pilot, and confirmation use bootstrap seeds `2026091901`,
`2026091902`, and `2026091903`, respectively, with 100,000 resamples each.
Every pilot cell uses the same seed, and the interaction resamples matching
opening indices jointly. This gate is relative one-thread evidence, not an
absolute rating, 3300 proof, default change, installation or release approval.
