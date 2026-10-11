# Restricted singular fail-low correction learning — session61

Prepare a separate correctness candidate on accepteded34480/a4d90b41,
topicnight/singular-correction. It changes no SEE code,correction formula,
values,table sizes or pruning policy. Root's09:00 plan amendment orders its
games after data step2,SPSA and SEE geometry,before an admission winner's
long-clock transfer; existing heuristic queue follows both repairs.

Proof: alphaBetaPV recognizes the excluded-move root as inSingular and
suppresses ordinary TT cutoffs/stores there,but its completed-node pawn,
non-pawn and minor correction-learning sites still run. An upper bound
for the restricted move set need not bound the unrestricted position.
Guard only the completed-node learning by!inSingular. Preserve all beta
cutoff/fail-high learning: a restricted lower bound is sound for the full
move set. Ordinary nodes and descendants outside ExcludedPly are unchanged.

Regression searches exercise both colors with actual legal exclusions and
worker evaluator/root oracle. Each restricted fail-low previously writes
[-6,-6,-6] into the three root correction slots; repair leaves[0,0,0].
Ordinary fail-lows,restricted/ordinary fail-highs and exclusions at another
ply all retain nonzero writes. Ten controls in total. Initial fixture
incorrectly treated a legitimate score0 as stoppedSearchScore; corrected
to assert info.Stopped,then final baseline confirms only the two intended
restricted fail-low failures. Initial fixture1 logs are historical,not proof.

Require final baseline failure,all10 repaired controls/existing singular/
ProbCut/correction tests,engine/full short/race,vet,accepted-PGO build on
rig13,15/2GiB/swap0/nice19. Checked topic work stays separate from playing
source until the real-clock gate completes.

Manually nominate the actual immediate base,own A/A40,fixed400 paired
10+0.1 games on one trusted lane,identical package except this patch.
KEEP only printed AND unrounded paired lower95>0,zero operational faults,
complete400/200 pairs,intact pins. Otherwise shelf; no extension,pooling,
automatic adoption or mixing with other correction tables. Changed package
needs necessary combination checks and a concrete pre-game manifest.
