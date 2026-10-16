# Threads8 claim instrument v4 preparation

The merged plan's October16 09:45 note records Bryan's choice:
**Threads8/Hash256, CCRL Blitz 8CPU labels**. This prospective instrument
inherits v3's estimator, openings, clocks, counts, audits and decision rules.
It does not nominate a candidate or authorize a sample with missing inputs.

The initial frozen pool is Ethereal13.00 **3581**, akimbo1.0.0 **3614**,
BlackMarlin9.0 **3623**, Berserk12 **3741**. The raw October16 full-list HTML
is copied to `/home/ehrli/ngn-data/claim-v4-20261016/ccrl-blitz-all-2026-10-16.html`;
`labels.json` beside it preserves its digest and exact rows, including the
8CPU names and1695/3229/3390/2918 game counts. No games informed this pool.

**Ethereal is not admitted.** The official
[v13.00 release](https://github.com/AndyGrant/Ethereal/releases/tag/v13.00)
distinguishes its free Standard HCE edition from commercial NNUE. Building
the official source tag2687b3834179fc3ebf52f0f13b96499f4f6977e9 with
`make pext` produces the free HCE binary. It emits `NNUE is disabled for this
binary` when its advertised UseNNUE default is selected. The CCRL row is
classified commercial. The free executable cannot carry3581. Preserve the
failed smoke and free build as unadmitted; do not use it in a claim or infer
a replacement label from games. The source build and failed smoke live
outside NGN source/net directories. No licensed NNUE artifact was found in
the campaign's existing opponent storage.

The v4 source therefore names an absent `ethereal-13.00-nnue` artifact and an
explicitly unresolved digest. Normal artifact checks prevent a live receipt
or dry admission until that input is resolved. Bryan must supply the correct
licensed artifact or approve a prospective pool change. Velvet6.0.0 has a
verified open-source 8CPU row3590±16/965games in the same HTML and is the
proposed replacement; it has not been added to the frozen pool or played.

The other three anchors pass unscored Threads8/Hash256 operational smokes
from `/tmp`, on CPUs13,15/nice19. Each advertises the required thread/hash
range, accepts its full frozen defaults with book/ponder off where offered,
returns two legal start-position moves after newgame, and exits0 without
engine errors. Akimbo and BlackMarlin are official Linux release binaries;
Berserk reuses the accepted original build. Their embedded nets are bound
by the binary digests; no net-file option is advertised by the first two.
The exact transcripts/options/digests are in the adjacent session92 results.

v4 settings:

- Schema `NGN-CLAIM-4`; Threads8/Hash256 for every engine, NGN GOMAXPROCS8.
- Concurrency1; both players inherit physical cores0,2,4,6,8,10,12,14,
  nice19. CPUs13,15 admit preparation/dry modes only.
- Exclusive rig for live receipts and games: no trainer, sprt, spsa or
  datagen; no GPU compute process. Hopper and keepalive stay untouched.
- Screen10+0.1,40reversed pairs per anchor,320games; confirmation120+1,
  80pairs per anchor,640games. Owned opening SHA974e4b5a…3222,
  indices0–39/0–79,400-ply cap, complete history and all original audits.
- Each stage needs primary pooled lower95 strictly greater than3600 and
  raw scores strictly bracketing50%. Confirmation follows only the exact
  passing screen identity. No exclusions, extensions, optional stopping,
  score-based pool changes, or prior-game pooling. Bootstrap unchanged.

Regression fixtures cover the new schema/pool/labels/resources, scheduler
override, advertised ranges, effective Threads8, the strict3600 boundary,
fixed counts and same-identity confirmation. Engine behavior is unchanged.

Full short and full short race suites, focused claim fixtures and vet PASS
at09:48:16 EDT. The final unresolved-anchor declaration passes the focused
suite afterward. Source is committed locally as5f54c9c. An exact clean
snapshot of that commit is reconstructed on WSL by archive, index/tree and
commit bytes, without pull/fetch; tree93bbe4d8 matches. Built runner
`a2fb2dd09ab70074ca12dc2b36ec62bf1f7de16f2b8f8a05d7bac745a5ef2016`
is at `claim-v4-20261016/tooling/frozen/claimtool`. The three operational dry
modes PASS. Screen and confirmation dry modes correctly stop on unresolved
Ethereal; they create no receipt or game directory. These use the existing
v3 development-only identity as a fixture and nominate no candidate.
[Build and dry-mode receipt](2026-10-16-session92-results/v4/build-receipt.json).

The October16 09:45 order is: finish preparation; after both v4 readiness
and the moderated GPU cell's natural terminal, pause warm SPSA at its next
verified checkpoint; gate volume-control and moderated-T74 by fixed1000
against81215229; transfer any winning claim candidate at60+0.6/400games,
point>0; then the v4 screen and automatic conditional confirmation. No
separate T8-versusT4 width run. A screen failure returns to SPSA with
`-resume` and GPU cells. Every prospective attempt, including a failed
screen, gets its own complete immutable receipt before games.

The current warm controller treats any stopped child as failure and has no
pause/resume state. Before the planned pause, preserve the checkpoint and
controller disposition, then resume the same binary/recipe with `-resume`;
do not invoke its existing fresh-start command on a populated output file.
No tune or gate was stopped during this preparation session.
