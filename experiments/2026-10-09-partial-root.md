# Retain a completed root improvement at a tournament clock stop

Session37, isolated `night/partial-root` from accepted playing source
`ac658e2/be34532c`. The reviewed120+1 traces measured discarded tails;
the root loop currently discards even finished comparisons when a later
child stops. This candidate changes the returned move at that stop only.

In each aspiration attempt, first require the last whole iteration's
incumbent to complete first with a score strictly inside that attempt's
original window. Retain a later strictly better in-window score only after
its child, any reduced-depth verification and full-window re-search finish,
the root position is restored, and root-node accounting succeeds. On a
tournament clock stop, return the best such finished comparison. A stopped
child, fail-low/high, reduced-only bound, unfinished incumbent or external/
node stop never supplies the replacement. Each aspiration attempt starts
fresh. The reported depth still counts whole iterations; the returned score
belongs to the retained move. No incomplete-root TT/history learning or
completed-iteration report occurs.

No allocation, admission, clock-throttle, soft/hard stop or other search/net
policy is included. This is a heuristic, not a correctness repair. Runtime
stays uncommitted and unadopted until games; the
[patch](2026-10-09-partial-root.patch) preserves the candidate and tests.

CPU checks follow the already queued two time candidates on12,14/nice19,
after the GPU pilot. Narrow tests cover exact incumbent/improvement
comparisons, aspiration bounds, all excluded stop classes, whole-depth
accounting, legal result, root restoration and no root TT store. Then full
short/race suites, vet and explicit accepted-PGO build. These are code
checks only; no strength evidence or automatic adoption.

This follows admission and the25-move horizon in the strength queue, after
factorizer/provenance/final-vector SPSA priorities. Freeze/rebase this patch
alone on the then-immediate accepted source/net. Before games record actual
binaries, options, openings and commands. Fresh paired A/A40 at60+0.6 must
contain zero in its printed and recomputed95% intervals with zero faults.
Then paired SPRT[0,+6], alpha/beta.05, minimum400/cap4000 at60+0.6; accept
only completedH1 AND paired lower95>0 AND zero flags/crashes/illegal/no-move
outcomes/intact inputs. H0 or inconclusive cap shelves without extension.
No10+0.1 verdict, stacked time package, claim or publication.
