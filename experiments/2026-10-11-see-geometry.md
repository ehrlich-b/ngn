# SEE post-move geometry — session61

Prepare a single correctness candidate on accepteded34480/a4d90b41,
topicnight/see-geometry; no value,pruning-threshold or other search change.
Root's09:00 amendment toplan-2026-10-09.md explicitly gives it priority
behind data step2/SPSA and ahead of the existing heuristic queue.

Proof: first recapture previously used attackers collected with the mover
still occupying its source. Removing only its bit did not reveal the enemy
slider behind it. EP also selected the captured pawn's square as the exchange
square instead of the move destination. The repaired initial occupancy
removes source and any EP captured pawn,occupies destination,and computes
the first attacker set there through the existing occupancy-aware helper.
Subsequent recaptures already use that helper; material values stay unchanged.

Regression panel has6geometry cases and their color/rank reflections:
source-exposed rook capture(-400),quiet rook(-500),bishop capture(-240),
EP attack only on removed square(+100),EP destination defended bypawn(0),
EP removal exposing rook(0). Each uses an actual legal move and verifies
that SEE leaves the position unchanged. Baseline fails10/12 numerical
cases; EP-rook removal is a positive control. Existing SEE tests also pass
on the repair. Initial test-authoring errors were corrected before the
final baseline panel and remain asfixture1 logs,not engine evidence.

Required evidence: final baseline failure then repaired panel/existingSEE,
engine short/race,full short/race,vet,accepted-PGO build onrig13,15/2GiB/
swap0/nice19. Topic commit preserves checked work only; playing adoption
awaits its declared real-clock game rule.

Games require a manually nominated actual immediate base package with
identical source/net/options except this SEE geometry patch,own A/A40
then fixed400/200 reversed pairs at10+0.1 on one trusted free lane.
KEEP only positive printed AND unrounded paired lower95,zero flags/crashes/
illegal/no-move/watchdog results,complete sample/intact pins. Otherwise
shelf playing adoption; no rescued sample,extension,pooling or SEE constants.
If source/options/net has changed,perform only the necessary combination
checks and write a concrete binary/net/argv manifest before launching.
No games,adoption,release,claim or publication happens from this proof alone.
