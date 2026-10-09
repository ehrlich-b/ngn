# NGN board representation and key-order evidence

Plan0b(b) requires removing inherited board names/representations with perft,
nodecheck and fixed-depth identities. No foreign board source was opened for
this work. Sourcea49ebfc, immediate runtime5b882e2/be34532c.

**Representation refactor PASS / KEEP ON TOPIC8a565d1**, branch
`night/board-layout`, `output/campaign-board-layout/`, pushed to private origin.
One explicit `Position.Side` replaces both turn flags; castling/check/EP state
is independent. Make/unmake, null moves and copies preserve Side separately.
Move bits now hold flags0–3, promotion4–6, mover7–10, captured11–14,
source15–20 and destination21–26. Legal move codes fit27bits and both TT APIs.
Color-indexed occupancy replaces separate white/black occupancy fields.
Square-major `[64][13]` key storage replaces piece-major `[12][64]` storage.
Board getters, private piece parser/EP-victim/hash helpers, king-weight constant
and dark-square mask now use NGN names; all engine/tool callers are adapted.

The first compile catches an old experiment executable's PositionTag reference;
its type reference is updated. The first full suite catches a symmetry fixture
that copied metadata without the newly separate Side. Correcting the fixture
changes no production engine code; original failure remains `session19-board/`.
Corrected `session19-board2/` ends **01:12:40 EDT**, all engine/full short/race,
vet and accepted-PGO builds pass. Six public perft references pass on both sides.
All12 depth14 node/score/best/PV samples and3 canonical UCI nodechecks are exact
matches. The canonical nodecheck uses Hash64/Threads1/explicitbe34532c and
depth12/12/16 at Kiwipete/middlegame/rook ending; fresh immediate-base counts,
no stale September baselines or macOS taskpolicy. All computation13,15/nice19.
Local source equals the full tested source receipt before committing8a565d1.
[Full receipt](2026-10-09-session19-results/board-layout-review.json).

**Key assignment is intentionally unchanged in that refactor.** It therefore
removes names/layout fingerprints without claiming the complete requested new
draw order. It is retained on topic until the prior TT/provenance order is
resolved; no accepted-line/main merge, speed gain or Elo claim.

**Complete own key-order variant FAILS NEUTRAL ADMISSION.** Separate topic
`night/board-key-order`, `output/campaign-board-key-order/`, remains uncommitted.
It includes the tested representation refactor and independently seeds
`NGN board keys v1`, draws EP/castling metadata first, then square-major
occupants in descending square/piece order, then the side key. No legacy `*ZC`
names or piece-major random-draw order remains. Own seed-generated keys only.

Corrected `session19-board-key-order2/` ends **01:12:48**, hash/EP/perft checks
pass, but **all12 depth14 search samples differ**. New Zobrist keys change TT
indices/collisions and thus search; perft correctness does not establish node
identity. No neutral adoption or strength games follow. Local hash source equals
the tested key-order receipt. Failed prerequisite-only first attempt remains
`session19-board-key-order/`; no key-order sample was repeated or extended.
[Differences and receipt](2026-10-09-session19-results/board-key-order-review.json).

**Needs Bryan:** allow a separate standard A/A40/gate400 behavior verdict for the
complete new key order, or retain key assignment while choosing another neutral
design. The requested neutral rule cannot admit this measured variant. Until
then, keep both topics separate and make no claim that all inherited design
fingerprints are removed. No publication or disclosure is sent.

Session21 integrates the tested representation as39eb128 after accepted
provisional TTb49ab5a. Combined compatibility checks are queued, followed by
a separately authorized own-key A/A40/gate400 using Bryan's upper95>0
provenance rule. The prior Needs Bryan key-gate question is resolved by the
updated plan; no key adoption yet. [Current manifest](2026-10-09-session21.md).
