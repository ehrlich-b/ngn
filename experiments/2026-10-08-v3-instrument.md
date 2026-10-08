# 3600 claim instrument v3 — session 2, 2026-10-08

Session started **10:42:08 EDT**, deadline **12:22:08 EDT**. This is instrument preparation,
not a frozen claim attempt. No v3 screen or 120+1 claim may launch with an incomplete receipt.
The [campaign log](2026-10-08-3600-campaign.md) owns the current candidate and queue; the
[prospective pool preparation](2026-10-08-3600-protocol-preparation.md) owns the pool design.

## Instrument change

The previous claimtool hardcodes >3000 in both screen and claim eligibility. The new
`NGN-CLAIM-3` schema binds **ClaimThreshold=3600** in each plan and analysis. Strict equality
at 3600 fails. Old v2 receipts cannot pass v3 validation; their frozen executable/source remain
at `claim-run-20261007/frozen/claimtool` and `/home/ehrli/wt/claim-run-20261007`.

The exact new pool is Maelstrom 3.3.0 **3317**, Counter 5.5 **3359**, Viridithas 10.0.0 **3557**,
Berserk 12 **3713**. Every executable path/digest and label is required. Counter 3.8 is dropped
prospectively. Berserk remains a measurement opponent outside every NGN source/net directory.

Unchanged: 10+0.1 screen, 120+1 confirmation, eight simultaneous games, 40/80 reversed opening
pairs per anchor (320/640 total games), CPU 0–15 / nice 19, GOMAXPROCS=1 per engine, 400-ply cap
after natural terminals, zero exclusions, full independent legal/clock/terminal audits, original
smoothed-score inverse-variance primary formula, and 10,000 whole-opening bootstrap draws / seed
3000. Raw anchor scores must lie strictly above and below 50%. Both samples require their primary
lower 95% bound >3600. No older games, extensions, optional stopping, or score-based anchor replacement.

Confirmation recomputes the persisted screen analysis before checking candidate binary/net/source/
provenance, tool, anchor artifacts/dependencies, selected options, and opening identity. The frozen
opening slice has the same digest in separate attempt directories; its paths necessarily differ.
The queue wrapper also requires the v3 schema, bound threshold, strict lower bound, and unchanged
post-run hashes before advancing. The historical v2 flags continue to mean >3000 only.

## Validation plan and run

No engine implementation changes. Regression fixtures cover old schema/threshold rejection,
3600 equality and adjacent floating-point values, sub-3600 v2-pass values, counts, pairs, exclusions,
bracketing, all four labels/executable pins, stability/tooling pool selection, and screen-to-confirmation
identity. Existing audits cover illegal moves, mislabeled terminals, repetition/cap precedence and clocks.

The [detached validation controller](2026-10-08-v3-tool-tests.sh), PID **541322**, waits for final
source pins and termination of the existing PGO controller **534228**, checking silently every five
minutes. It then proves the old threshold fails the new regression, runs narrow claimtool tests,
required engine short/race tests, and full short/race suites on WSL CPUs 13,15 / nice 19.
Tests use an isolated source archive plus hash-bound patch at
`/home/ehrli/ngn-data/night-20261007/session2/`; no running job's source/binary is overwritten.

**All tests passed at 11:04:59 EDT**, including the required engine short/race and full short/race
suites. The unchanged tool admits the synthetic lower bound 3500 and fails the new witness; v3
passes the threshold and identity regression suite. Exact
[test receipt](2026-10-08-session2-results/test-receipt.json). Pre/post input hashes match; source
archive SHA-256 `ac8d4134a37150057c22383f41fc61afbe4ce75723aa44718299e2aa33b2bff5`,
candidate patch `93b0b75e93af0132df536df400c0eb7bf9fcd54b3ad374364fe20ef22c81d9f7`.
Local checks are limited to formatting, `bash -n`, receipt reconciliation and `git diff --check`.

The [clean-source build procedure](2026-10-08-v3-freeze-build.sh) verified all **226** compilation
inputs against that tested tree, carried the independently accepted `default.pgo`, built the new
tool and NGN binary on WSL, and saved dry plans for the four operational/screen modes. Exact
[build receipt](2026-10-08-session2-results/v3-build-receipt.json) and
[screen dry plan](2026-10-08-session2-results/v3-screen-dry-run.json). Clean source commit
**376671f842aad7841cd2c743c834c61f702eb0ae** at
`/home/ehrli/ngn-data/claim-v3-20261008/tooling/source`.

Frozen tool **4cdf39aa…**, NGN **f36af8be…**, development net **be34532c…** under `tooling/frozen/`.
The NGN build metadata confirms automatic PGO, Go 1.25.5, amd64/v3 and `vcs.modified=false`.
The development identity uses the immediate accepted `be34532c`; it is not a final claim nomination.
All four dry modes validate source/build/provenance/anchor pins and owned openings without engine
searches, live clock probes or game creation. The separate full-option Berserk readiness probe passes.

Build-launch custody: PID **543269** never began work and is terminal; preserve its empty
`session2/build.log` and PID file. Observed replacement **543309** compiled and dry-checked once,
then its final check incorrectly matched historical `receipt.json` files in the copied source tree.
`session2/build2.log` retains that failure. Corrected the check to cover only the new output directories,
verified the existing artifacts, and wrote the completion receipt **11:15:34 EDT**, without rebuilding,
launching games, or altering any frozen artifact. The corrected procedure is committed for reuse in a
new directory; do not rerun it over the existing `tooling/` root. Crash-handler log entries here are
initialization messages from the dry tool invocations, not engine crashes, and are not frozen inputs.

## Remaining admission before v3 games

1. Finish the stronger anchor's build receipt and freeze its advertised options. Operational stability
   is a fixed four games at each of 10+0.1 and 120+1 (indices 0–1, both colors), excluded from claim
   analysis; run only while no trainer, datagen or tuner/match process exists.
2. The clean committed tool/build binding is complete. Exercise a fresh tooling sample and the
   existing bounded A/A clock control before v3 games; keep the artifact-bound source intact.
3. Nominate the immediate accepted engine/net after the pending longer-H512 gate. Freeze its source,
   binary, net, provenance, supported options, openings, runner and audit identities.
4. Commit a complete prospective v3 protocol/attempt manifest before any v3 screen game. A passing
   v2 screen supplies development orientation only. No 120+1 claim is queued by this session.
