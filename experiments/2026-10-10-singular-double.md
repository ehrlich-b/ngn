# Bounded double singular extension — session45

Fable L5b suggests a second singular ply. The existing source supplies one
positive extension and one negative extension. This trial changes only the
positive branch on isolated `night/singular-double`, base56b35ef/be34532c.
No third-party engine source or implementation was consulted.

After the ordinary excluded-move verification, spend one extra ply when all
of these hold: non-PV, depth6–12 (also retain the tunable singular minimum), gap
strictly greater than64cp, and no check/passed-pawn extension has already
restored the normal decrement. NGN chose64 and12 for this untuned trial.
Preserve the existing negative extension, per-node cap and cumulative path
budget. The added ply is a search heuristic; tests cannot accept it for play.

The live witness preloads alternative children with exact zero scores and a
chosen quiet child's exact entry at depthd. Ordinary single extension cuts
at that entry; double extension must search/store the child atd+1. Check
64/65cp, depths5/6/12/13, PV depth, the cumulative-budget clamp and stopped verification
unwind. The65cp case must fail on the unchanged search and pass on candidate.
Then full short/race suites, vet and accepted-PGO build on forge8–11/nice10/
2GiB/swap0. SPSA2.5GiB and reader128MiB retain their scopes. No games on forge.

This prepared code lane follows data, capture decision, time management,
SPSA and the earlier search candidates. Before games, nominate the actual
accepted source/net and apply only this patch; check compatibility again
only if source changed. Freeze exact hashes, commands and topology in the
launch manifest before fresh A/A40 and a paired10+0.1 SPRT[0,+6], alpha/beta
.05, min400/cap4000. Rig only; GOMAXPROCS1, Threads1, Hash64, book-off,
NNUE-on, accepted616b1ebe PGO, openings974e4b5a, unchanged9b0002f8 harness.
A/A must contain zero in printed and unrounded paired intervals, with zero
faults. Keep only completedH1 AND both lower95>0 AND zero flag-outs/crashes/
illegal/no-move results. H0/cap shelves without extension; interruption has
no verdict. No automatic games, stacking or code-only adoption.

Forge270961 completes08:26:35 EDT/exit0. The old65cp child-depth witness
fails exactly at6 versus7; all new controls, full short/race suites, vet and
PGO build pass. Local source hashes match the checked receipt. No games
or runtime adoption. [Receipt](2026-10-10-session45-results/singular/receipt.json);
[exact patch](2026-10-10-singular-double.patch).
