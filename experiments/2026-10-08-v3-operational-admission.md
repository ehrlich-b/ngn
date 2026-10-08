# Prospective v3 operational admission — session 3

This completes the remaining instrument checks from the
[v3 record](2026-10-08-v3-instrument.md), using its already frozen development
control. These **12 games** are excluded from every strength screen and claim.
They do not nominate the longer net, select an anchor by score, or authorize a
320-game screen or 640-game claim. The complete final claim protocol remains
pending the immediate accepted engine/net verdict.

Run exactly these stages in order, stopping on the first operational failure:

| Mode | Opponent | TC | Games | Opening indices |
|---|---|---|---:|---|
| tooling-check | Counter 5.5 | 1+0.02 | 4, serial | 0–1, both colors |
| stability-screen | Berserk 12 | 10+0.1 | 4 | 0–1, both colors |
| stability-confirmation | Berserk 12 | 120+1 | 4 | 0–1, both colors |

The existing tool performs a fresh bounded A/A stop-response clock control at
each stage's declared concurrency and clocks (concurrency 1 for tooling,
8 for stability). It writes and seals the complete live receipt before each
game, then independently audits every opening, ply, clock and terminal,
reconciles both color pairs and rechecks every input hash. Require four admitted
games and two pairs per stage, zero exclusions, no flags/crashes/illegal/no-move
results, clean hashes, and the exact advertised settings. Scores have no
admission role. No replay, extension or automatic retry after failure.

Frozen development control and tool:

- Source `/home/ehrli/ngn-data/claim-v3-20261008/tooling/source`, clean commit
  `376671f842aad7841cd2c743c834c61f702eb0ae`.
- NGN `f36af8be0f0d3c86a4e097ca1520f131801ffa0c20ab107966a30d2bc910efb6`,
  accepted NGNN3 KB8 H512 net
  `be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a`.
- Tool `4cdf39aa3ae964ccbc7b0c7b7a65a4fe71f32992ee35a8326c1f7c0163d41e7c`;
  `tooling/build-receipt.json` SHA-256
  `b3ceb67ed2c4781465e264d19936022d7759e944de38db3cb8691afc6d41f04a`;
  development identity SHA-256
  `2509127a6eefedde73b7f2cbc648dfed96d1be6c75c1f2a8d66736e0c3aabf31`.
- Counter 5.5 and Berserk 12 use the executable pins and advertised settings
  in the frozen tool and [anchor record](2026-10-08-3600-protocol-preparation.md).
  Berserk binary `7b9b5898`, embedded net `fb675dad41b4`; the admission controller
  additionally checks its complete selected option map against the prior probe.
- The full owned opening digest is `974e4b5a`; each stage freezes the first-80
  slice before using only indices 0–1. Go 1.25.5, GOAMD64=v3, Linux WSL2,
  Ryzen 9800X3D. Frozen build argv and training/data/export/parity provenance
  are in `tooling/frozen/` and revalidated by every live receipt.

The detached [controller](2026-10-08-v3-operational-admission.sh) waits on the
**existing controller 522940** with Linux pidfd, a single native completion
wait bounded at 14 hours. There are no assistant polling loops or periodic
status lines. It requires session1's terminal completion event and post-gate
hash receipt, then drains only the current campaign tuner group. Require no
trainer, datagen, tuner, match or speed probe before live work; the claimtool
retains its own exclusive-rig checks. Leave hopper untouched. Read C: space
through its mounted filesystem and require >=25 GiB before games. Use CPUs
0–15, nice 19, GOMAXPROCS=1 per engine. Restore a tuner paused by this controller
on success or failure; a previously completed tuner is not restarted.

Output root: `/home/ehrli/ngn-data/claim-v3-20261008/admission-session3/`;
`run-{tooling-check,stability-screen,stability-confirmation}/` contains receipts,
raw games, audit, analysis and post-run hashes. The root `admission-receipt.json`
summarizes admission. Campaign events append to `night-20261007/status.jsonl`.
Dry plans use separate `dry-*` directories and launch no engines. Commit this
record and controller before queuing the live controller. No new GPU trainer
should start until admission is complete or failed; its short exclusive window
is required by the measurement instrument. No production or publication occurs.

**Queued**, **11:38:23 EDT**, controller **552631**. The exact record/controller
were committed and pushed as **862afa1** before launch. Linux pidfd support was
proved against the live existing controller; the
[waiter receipt](2026-10-08-session3-results/admission-waiter-ready.json) binds
522940 and its exact command. No v3 games have begun. Estimated trigger around
**22:15 EDT**, with results roughly **22:30 EDT**, conditional on training/gate
completion. A failed dependency or expired wait halts admission without retry.

All three fresh dry plans **PASS**, including exact mode/anchor/count/concurrency/
clock fields and absence of game directories or pre-run receipts. `bash -n`
passes. Frozen queued controller SHA-256
`c306d1d0eeade42ab6985291d46dc8b0dcadf4e6a7569355c416788ddb9eb8f1`;
original prospective manifest
`c0b25cdd60b385e987fdfca10aa631b42c4e8e5dcfd0b1b5d736a467457c9c19`.
Their immutable box copies and the originating commit are hash-checked before
the wait, before games and after completion. This result annotation is separate
from the original prospective box manifest. The source/tool/development identity
stay at their original pins even if a later net or speed change is accepted.
