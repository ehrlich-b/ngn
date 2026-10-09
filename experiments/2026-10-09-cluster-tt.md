# NGN cluster TT candidate — session 19

Prepared from accepted runtime 5b882e2 / be34532c, source snapshot a49ebfc,
branch `night/cluster-tt`, `output/campaign-cluster-tt/`. One TT package changes;
search, time allocation, evaluation, network, Hash budget and training stay fixed.
No acceptance, new absolute strength claim or automatic promotion.

The October9 [hot-path audit](/Users/ehrlich/repos/ngn-campaign/hot-path-audit.md)
finds no direct copying but flags the inherited single-slot/XOR-key TT layout
and replacement policy. This candidate is written from scratch: four 16-byte
full-key records per 64-byte cluster, a new payload layout, consistent 8-bit
aging and explicit occupancy. Full keys introduce no short-signature false hits.
Public cluster/depth/recency concepts are allowed by the session contract.
Existing API names remain for callers; no third-party source was used.

Payload bits: score0–15, depth16–22, age23–30, PV31, valid move32–58,
bound59–60, occupancy61. Depth is clamped at zero, preserving 0–127;
all legal move bits, signed i16 scores, bounds and PV flags roundtrip.
LowerBound is now enum3. Actual allocated bytes remain within Hash MB.

A record's retention is `4*depth + 2*exact + PV - 8*min(age_distance,31)`;
modular age distance uses eight bits. A matching key accepts a record at
least as valuable, keeps the previous move when a new result has no move,
and refreshes age when rejecting a shallow result. Other keys fill empty
ways, then replace the least valuable way only if the new retention is no
smaller. Synchronized access locks the entire cluster; full-key and payload
are published together. hashfull samples256 evenly spaced clusters and
reports current-generation occupancy per mille. Clear remains idle-only.

CPU verification on WSL13,15 / nice19 follows the accepted clock trajectories. Require baseline failure/candidate pass for four-collider retention,
packing extremes/special moves, key0/zero score occupancy, weakest-victim
selection, depth retention, generation wrap and move retention; existing
receiver isolation/concurrency controls; engine/full short/race and vet;
explicit accepted PGO builds. Source remains uncommitted until a verdict.
Historical single-slot byte-compatibility assertions are replaced by tests
of the new representation; no node-identity claim is made.

The updated October9 plan's item0b(a) explicitly requires the standard
**10+0.1 A/A40 then400-game paired gate** for this provenance replacement.
This supersedes the earlier prospective long-clock SPRT before any games.
The engine/net remain immediate accepted5b882e2/be34532c, Hash64 /Threads1 /
OwnBookfalse; no Hash256, time, search or evaluation change is stacked.
Use974e4b5a openings and unchanged9b0002f8 instrument, concurrency2,
CPU13,15 /nice19 /GOMAXPROCS1. The detached game controller waits for corrected
TT full checks and board CPU checks, then writes an exact game manifest before
A/A40. Require exactly400 games/200 pairs, printed/recomputed penta lower95>0,
zero flags/crashes/illegal/no-move results and intact input pins; otherwise
shelve without extension. No provisional keep, sign-only acceptance, SPRT or
automatic promotion. Factorizer's frozen old-base gate remains pending; if its
net wins before source adoption, re-gate on the new actual accepted net.
Bucket/Hash256 strength work at a longer clock remains a later separate item.

Remaining audit findings concern board API/representation and the five
SEE material values. They were not direct copying findings and are disclosed
in NOTICE; changing them would be separate candidates, with separate evidence.
The new TT is a candidate remedy for the largest design fingerprint, not a
claim that every inherited interface/design choice is eliminated.

Initial CPU pass in `session19-tt/` proves baseline four-way failure and all
new candidate controls pass. The full short suite finds one PV-test fixture
assuming a same-depth non-exact store always overwrites an exact entry. The
new TT intentionally retains the more valuable exact entry. The fixture now
stores the bound at greater depth, keeping the test about PV extraction
rather than TT replacement. Engine code is unchanged by this correction.
Preserve the initial failure. Corrected full checks are queued in
`session19-tt2/` after the already queued clock trajectories, avoiding shared
CPU timing overlap. Source receipt additionally includes `pv_reporting_test.go`.

Corrected full checks complete01:02:31, all tests/vet/PGO pass; local five-file
source equals the reviewed receipt. A/A40 passes6/31/3, no failures. Detached
standard gate664657/664659 starts400 paired games01:23:36 in session19-tt-gate.
[Code review](2026-10-09-session19-results/tt-review.json). No automatic source
adoption; game verdict and factorizer's frozen immediate-base review remain.
