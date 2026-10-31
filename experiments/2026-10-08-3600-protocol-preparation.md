# Stronger top anchor for the 3600 protocol — 2026-10-08

**Preparation only: v3 instrument and artifact receipts are not yet frozen.** The campaign
authorizes this work; its current accepted-net screen remains the unchanged historical v2 instrument.
The instrument below must be implemented, tested, artifact-bound and prospectively frozen before games.

Berserk **12 64-bit, 1CPU** is the proposed stronger top anchor: fixed CCRL Blitz label **3713 (+9/−9)**,
**4,807 games**, list computed October 3, 2026. Source:
[CCRL exact version](https://computerchess.org.uk/404/cgi/engine_details.cgi?eng=Berserk+12+64-bit&print=Details+%28text%29),
[complete Blitz list](https://computerchess.org.uk/404/rating_list_all.html).
The unadorned 64-bit row is the one-thread entry; the distinct 8CPU row is excluded.
The [official release](https://github.com/jhonnold/berserk/releases/tag/12) was published
2023-10-26; tag `12` pins commit **94d48b044bbad8144f7fce7a2848ac888812a1fd**.

Build from that source on WSL, outside all NGN checkouts, at
`/home/ehrli/ngn-data/claim-v3-20261008/anchors/berserk12/`. Its code and weights are used exclusively
as a match opponent; no NGN code, trainer or net derives from them. Use the release's pinned network
submodule, require the network digest prefix `fb675dad41b4`, and save full source/submodule/binary/net
hashes. `make basic` compiles the native Linux binary without running its benchmark or cleanup target.
The detached [build script](2026-10-08-build-top-anchor.sh) waits for the existing exclusive screen
to end, then builds on CPUs 13,15 / nice 19. A UCI/readiness probe from `/tmp` confirms the embedded
network can start without adjacent files. Stability/orientation games are deferred until no trainer
is live; they do not select or replace the anchor by score.

Frozen web evidence on the box anchor root:

| File | SHA-256 |
|---|---|
| `ccrl-version.html` | `8a4ab1806d7be8f2595c56507b91145c28e719a87a2297a02db8145046ad99a3` |
| `release.json` | `d6c25290eaff6f53ca51d3027b6ef1f6fd2fd7964b56d392133a60223f599a6c` |
| `tag.json` | `f1edc494c1bbf2c28b4c3a4c4d46fb61ee2ec34937d4432ffdc1c974ad2700c2` |

## Proposed prospective revision

Retain exactly four anchors and the v2 statistics/sample size: Maelstrom 3.3.0 **3317**,
Counter 5.5 **3359**, Viridithas 10.0.0 **3557**, Berserk 12 **3713**. Drop Counter 3.8
prospectively because its 2994 label is remote from the target; retain every other fixed label and
artifact. This adds a distinct stronger engine family, with a 156-point upper extension.
No existing v2 game belongs to the new sample. Anchor choice is based on the published label and
artifact availability; no Berserk orientation game has occurred.

Keep 120+1, eight games total, 160 per anchor / 80 intact reversed opening pairs, indices 0–79,
the same 400-ply cap, zero permitted exclusions, full legal/clock/terminal audits, and unchanged
smoothed-score/inverse-variance primary formula. Preserve the 10,000 whole-opening bootstrap
with seed 3000 from v2. Goal passes only when primary pooled lower 95% bound is **strictly >3600**
and raw anchor scores lie strictly above and below 50%. No extensions, optional stopping,
post-hoc replacement or previous-game pooling.

For a strictly retargeted screen, use 10+0.1, 80 per anchor / 40 pairs, and require its primary lower
bound **>3600** plus bracketing and all audits before the 120+1 claim. The current v2 tool's
`ScreenEligible`/`ClaimEligible` flags instead test **>3000** and must never be reported as 3600
eligibility. This strict screen rule is conservative about time-control transfer; any later change
requires a prospective protocol declared before its games, not a reinterpretation of this session's
v2 screen.

Remaining work before a complete frozen v3 protocol:

1. Verify the anchor build receipt and freeze its supported settings (Threads 1, Hash 64,
   book/ponder off where advertised, empty tablebase paths, all other defaults recorded).
2. Run the same fixed four-game 10+0.1 and 120+1 operational stability samples as v2, excluded
   from claim analysis. Record a separate development orientation if useful; never pool it.
3. Change the owned claimtool to bind a new schema, exactly this anchor pool and threshold 3600;
   regression-test strict equality/nextafter boundaries, wrong labels/pins and counts, incomplete
   pairs, exclusions, bracketing, confirmation identity, legal terminals and clock failures.
   Run all required short/race tests on WSL. Preserve the frozen v2 tool for historical analysis.
4. Nominate the immediate accepted engine/net after its game gate; freeze its exact clean source,
   build, net and provenance. Commit the final complete v3 protocol before any v3 screen game.

## Session 2 build review — 10:57 EDT

The build completed **10:46:36 EDT**, source commit and network pin match the prospective record.
[Build receipt](2026-10-08-session2-results/berserk-build-receipt.json),
[UCI transcript](2026-10-08-session2-results/berserk-uci.txt). Binary SHA-256:
`7b9b5898a781cbe0757caef0b7281df5cf5029d17e67ed3df4824f0eb1c0125d`;
embedded network SHA-256 `fb675dad41b45f72a030379f1022412704a68554f703ccb407474043640bafc7`,
network submodule commit `dae8f890cecc27e66a7b8c30b0c5fbc9dcd5a5c1`. The probe from `/tmp`
passed `uciok`/`readyok`, exited zero, and emitted no stderr. Its source/net stay outside NGN.

Supported settings frozen for the operational samples and prospective v3 pool: **Threads 1,
Hash 64, SyzygyPath <empty>, MultiPV 1, Ponder false, UCI_ShowWDL true, UCI_Chess960 false,
MoveOverhead 300, Contempt 0, EvalFile <empty>**. There is no advertised book switch. The
existing option-freezing code preserves the advertised MoveOverhead default (no space in this
option's name), rather than sending NGN's separate `Move Overhead` option. Empty EvalFile selects
the embedded, binary-bound opponent network. A second from-`/tmp` probe sends the **entire** frozen
set in the claimtool's configuration order and passes readiness/exit with empty stderr:
[option receipt](2026-10-08-session2-results/berserk-option-receipt.json),
[transcript](2026-10-08-session2-results/berserk-options-uci.txt). No stability or orientation games
have occurred.

The [v3 instrument](2026-10-08-v3-instrument.md) now pins this exact executable, replaces
Counter 3.8 with Berserk 12, and binds 3600 in a new schema. Full admission remains pending;
the live longer-H512 trainer prevents exclusive operational games tonight until it finishes.
