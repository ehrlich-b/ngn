# Factorizer gate on the accepted runtime — session35

Frozen before games, October9. Playing source `ac658e2`, accepted net
`be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a`.
Key order is shelved after its complete gate: unrounded upper95=-0.0980.
No key overlay or other search candidate is included.

The factorizer's full58 epochs/selected57, unchanged recipe, byte-identical
reexport and20 exact integer cases were reviewed in session31. Reuse that
completed evidence, with no retraining or parity replay. Candidate net
`d55a5096f1ad3882a806d2af8d2c699f275f087f8cacd6070cd60b3a41e9f2ce`.
Only EvalFile differs between sides. Build one clean accepted engine with
Go1.25.5/GOAMD64=v3 and explicit PGO616b1ebe; source is the Git archive
ofac658e2. Exact accepted-source full checks and null-frame identity/speed
evidence are inherited. Record binary/input digests, Go/OS and both exact
commands in `game-manifest.json` before the first game.

Fresh A/A40 then exactly400 games/200 reversed opening pairs,10+0.1,
concurrency2, CPUs13,15/nice19/GOMAXPROCS1, Hash64/Threads1/OwnBook=false.
SPRT9b0002f8/openings974e4b5a are reused unchanged. Minimum games1000000
disables early stopping; maxmoves400/gametimeout180/lowpower=false.
Preflight must contain zero in both printed and recomputed paired95% CI,
with zero flags/crashes/illegal/no-move results. Complete gate accepts only
printed AND unrounded lower95>0, zero operational faults and intact pins;
otherwise shelve without extension. Interrupted/invalid runs have no verdict.
The existing independently tested factorizer parser reconciles every raw
game increment, reason, WDL, penta and interval. No automatic promotion.

Run detached in `/home/ehrli/ngn-data/night-20261007/session35-factorizer/`
using [the controller](2026-10-09-factorizer-current.sh). C: admission25.5GiB
reserves0.5GiB beyond the mandatory floor; files grow by substantially less.
SPSA0–11/hopper stay untouched. CPUs12,14 and the GPU are available for a
bounded architecture pilot. No claim, publication or old failed-controller
restart. Controller writes append-only status and terminal receipts.
