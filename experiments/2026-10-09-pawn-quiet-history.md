# Pawn-structure quiet ordering — prospective session20 candidate

Topic `night/pawn-history`, source `f97f55c`, immediate accepted runtime
`5b882e2 / be34532c`. Implements merged plan item7 from NGN's own contracts.
No third-party source or net is used. No games, adoption or source commit
before the declared verdict. TT, board, SEE, time and training changes are
excluded from this candidate.

The existing quiet ordering histories depend on moving piece/destination
and preceding moves, but do not distinguish pawn structures. Add a worker-owned
512-slot pawn-context table of colored piece/destination int16 values:
**851,968 bytes per worker**. Reuse the existing board-only pawn hash masked
to512 slots; collisions intentionally share an entry. Evaluation correction
entries stay separate. NewGame clears the table with the other histories.

At the three existing completed-node history sites, learn from a quiet winner
only when its score exceeds the original alpha and it occurs in the completed
quiet list. Reward that winner and penalize the completed quiets before it.
Use the existing NGN H1 depth-dependent bounded response toward ±8192. Captures,
en passant, promotions and castles produce no pawn-history signal. Updates
use the restored parent pawn structure, including when the winner is a pawn
move. Stopped nodes do not update this table.

Only ordinary quiet ordering consumes half the stored entry. TT, capture,
promotion, castle, killer and counter priority rules remain as before.
The existing GetHistoryScore, quiet-pruning and LMR consumers receive no
new term. This is one ordering mechanism, with no evaluation or reduction
policy change and no assumed Elo gain.

Code verification follows the queued SEE checks, on WSL CPU12,14 /nice19:
prove the baseline root-learning/context-ordering witness fails, then the
candidate passes. Check completed examples, fail-low/unsearched/special-move
exclusion, bounds, context/worker isolation, reset, and interrupted-root
exclusion. Require engine/full short/race, vet, and accepted-PGO builds.
No existing test contract is weakened to accommodate the new behavior.

After code checks, measure cost with the existing speedprobe on accepted
be34532c, Hash16, CPU12/GOMAXPROCS1/nice19: warm both sides, then exactly
12 balanced AB/BA blocks, six positions×1second per side. Retain raw records,
aggregate paired NPS ratios and the two order strata. This sample measures
resource cost under the concurrent Rodent/TT workload; changed search paths
prevent interpreting it as a pure-speed result. It gives no Elo or acceptance
verdict, and is not extended after observing its result.

Future games wait for higher-priority provenance work and the factorizer
verdict. Replay only this change onto the actual immediate accepted source/net
and write the exact manifest first. Fresh A/A40 at10+0.1, then paired powered
SPRT[0,+6], minimum400/cap4000, using974e4b5a openings, Hash64/Threads1,
GOMAXPROCS1 and an assigned disjoint affinity. Accept only terminal H1,
recomputed penta lower95>0, complete pairs, zero flags/crashes/illegal/no-move
results and intact inputs. Otherwise shelve, with no extension or provisional
keep. This prospective small-effect rule uses merged plan item6; it does not
change the already-running TT/SEE/net fixed gates.

Results live in box `night-20261007/session20-pawn-history/`. CPU verification
and the fixed resource sample can run detached; no strength games follow
automatically and the accepted engine/net stay unchanged.
