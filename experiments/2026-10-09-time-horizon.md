# Retain a 25-move bank horizon — isolated session36 candidate

The reviewed old-base120+1 bank trajectories show the bank falling to
8–12 seconds near full-move40 in the longer games. The accepted allocator
estimates only ten remaining moves from about full-move15. Its normal
target then uses10% of the usable bank plus0.8 increment. These traces
are diagnosis, not evidence that another allocation wins.

One candidate, implemented from NGN's existing allocator on clean
`ac658e2/be34532c`: raise the estimated remaining-move floor from10 to25.
This limits the bank component to4% per normal target and retains more
bank for later moves. Preserve the existing phase estimates above the
floor, explicit GUI moves-to-go, increment share, safety reserves,
hard-cap formula, stability/effort factors and all stopping behavior.
No immediate-admission patch or other search/net change is included.
At120+1/full-move30, the normal target changes from12.785s to5.594s;
the initial opening target stays4.795s. Derived hard limits naturally
follow the existing formula. This is a heuristic, not a correctness fix.

Isolated topic `night/time-horizon` fromac658e2,
`output/campaign-time-horizon/`; runtime files stay uncommitted until
the game verdict. [Durable patch](2026-10-09-time-horizon.patch).
Four tests cover late-game bank retention/unchanged hard-cap formula,
explicit moves-to-go, initial opening target and increment-aware reduced
banks. None has run while WSL entry fails. Run these plus full short,
full race, vet and separate explicit-PGO baseline/candidate builds on
the box; engines/builds/tests never run on the Mac. The
[checks controller](2026-10-09-time-playing-checks.sh) prepares it after
the separate admission checks, from another clean base directory.
Both are staged on Windows but have no confirmed launch.

This follows the first admission candidate, factorizer/provenance and
final-vector SPSA in the game queue. Rebase only this horizon patch onto
the then-immediate accepted base and freeze that accepted net; repeat
no completed diagnosis. Write the actual binary/command/affinity/opening
manifest before a fresh A/A40 at60+0.6. Candidate gate is paired
SPRT[0,+6], alpha/beta.05, minimum400/cap4000 at60+0.6; accept only
completedH1 AND paired lower95>0 AND zero flags/crashes/illegal/no-move
outcomes/intact inputs. H0 or inconclusive cap shelves without extension.
No10+0.1 verdict, combined time package or automatic adoption.
