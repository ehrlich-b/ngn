# Loader-proved compact accumulators — session46

The accepted be34532c NGNN3 net has an exact conservative bound of
[-19521,+9988] across all buckets and lanes. For each square, independently
choose empty or any of twelve pieces, and sum those min/max contributions
with the bias. This covers every occupancy, including malformed 64-piece
boards; king/perspective transforms only permute square terms. No sample
of positions, empirical overflow assumption or retraining is needed.
[Calculation](2026-10-10-session46-results/acc-bound.json).

Implement an original int16 accumulator stack and king-refresh cache only
for NGNN3 models whose loader proves this bound and whose output weights
meet the existing [-128,128] packed-output bound. Other models retain the
int32 path. Modular word adds/subtracts preserve final exact values even
when intermediate updates wrap; saturating arithmetic is forbidden.
Frames retain null aliasing and real-move ownership through growth/reset.
Quiet/capture updates pair both perspectives; refresh and castle paths
retain their original feature/delta semantics. Output uses the same exact
clamp, square, wide accumulation and centipawn truncation.

Prepare on isolated night/compact-accumulator, initially56b35ef; incorporate
only accepted capture integration before its source-combination checks and
speed nomination if capture passes. No search/time/net overlay. Require
loader-bound boundary/fallback controls, guarded unaligned/exact-alias
kernel comparisons against independent int64/modular arithmetic, all
NGNN3 transition/oracle/cache/null/worker controls on AVX2 and purego,
full short/race/vet and explicit accepted-PGO builds on forge8–11/nice10/
2GiB/swap0. No strength games on forge.

Before speed evidence, freeze the actual accepted base and one exact
compact patch/binaries/profile/net/commands/topology in a manifest. On
free rig CPUs, GOMAXPROCS1/nice19/Hash16: 36 exact identities (NGNN3 and
NGNN2 depth14, HCE depth11), warmup, exactly40 balanced AB/BA blocks,
six positions and3seconds per engine/position. Geometric log-ratio
Student-t interval, df39/critical2.0226909200367606. Keep only lower95ratio>1
and both order strata>1. Otherwise shelve without extension. An incomplete
sample has no verdict. Freeze the playing base through the earlier time
gate; this candidate may be checked now but cannot enter play without
its own completed speed verdict. No Elo inference or net/default promotion.

Capture passes and is integratedde148b9. The compact topic is advanced to
that actual accepted source before full checks. Forge297092 completes
08:55:31 EDT/exit0: normal/purego bound/kernel/oracle/cache/null controls,
full short/race/vet and both explicit-PGO UCI/probe builds pass. All eight
local source hashes match the checked receipt. Exact runtime patch is
tracked; isolated runtime remains uncommitted pending the rig speed rule.
Binaries live at /work/lanes/ngn-s46-compact. Rig speed remains unlaunched
while time occupies13,15 and provenance owns0–11.
[Code review](2026-10-10-session46-results/compact/code-review.json).

Session47 follows the newer CPU-routing directive: the rig sample never
starts. Its first and only timing sample runs on forgeCPU4/nice10, host
pinningON, while unchanged SPSA uses8–11. All36 identities and exactly40
raw balanced blocks reconcile. **KEEP +4.77%[+2.54%,+7.04%] NPS**, strata
1.055775/1.039595, zero probe stderr. Same exact eight source files and
95f7f599 patch as the completed session46 checks; no source/check replay.
Committed topic852c3b5 and integrated9bfbae4 on night/night after the rig
time sample's policy interruption leaves its source unnominated for a
future forge gate. No Elo inference or net/default change.
[Raw review](2026-10-10-session47-results/compact/review.json).
