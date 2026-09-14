# Optimized Rodent NGN versus external Counter 5.5

Status: frozen protocol independently accepted and released for one run.

The old external rating evidence predates the accepted Counter output speedup,
Rodent integration and its +96-Elo internal gate, and the subsequent 2.43x Rodent
search speedup. One fresh external comparison is now useful for prioritization.
This is not a repeat of the old broad calibration campaign, and no own-network
training or new search heuristic is part of it.

## Fixed proposed protocol

- Latest accepted NGN `52ee629`, `rodent-v1.1-anand`, exact existing Anand weights.
- Opponent: the exact Counter 5.5 release artifact from the accepted historical
  one-thread cell, with its supported options and actual NNUE path verified.
- Fresh 100-game current-NGN/current-NGN A/A, then 400 games / 200 reversed-color
  opening pairs. 10+0.1, concurrency four, one thread each, Hash128, no book and
  no adjudication; preserve the established affinity and interference controls.
  NGN Move Overhead is 100 throughout preflight and both phases; Counter has no
  such option and retains its native time policy.
- Freeze build/source, executable/model/opening hashes, supported options and
  the minimal reused harness before release. Do not apply NGN-only diagnostic
  expectations to the external engine or silently weaken its checks.
- Paired 100,000-resample bootstrap, seed 2026091201. A/A interval must include
  zero. Complete legal/terminal/operational/process checks are required.
- Fixed cap: no score-based extension or rerun. Report the NGN-relative WDL,
  pentanomial and interval irrespective of sign. This is a checkpoint, not a
  keep/shelve gate for already accepted pure-speed code.

## Interpretation boundary

The result will measure the remaining gap to this exact external engine under
these local settings. It cannot establish a formal list rating or simply be
added to prior internal gains. Historical Counter rating labels are conditional
orientation only: clock, machine, opponent pool and artifact correspondence
remain separate uncertainties. Absolute 3300 is unverified until adequate fresh
configuration-matched evidence supports that claim.

All execution stays on the authorized WSL host. No installed binary, default
backend or external release contract changes are authorized by this checkpoint.

## Frozen release

Latest NGN source is `52ee629b316f89373513cf9ef3be819d4555c3f3`, tree
`b7dfd988f8c3f4ee1d8a4a46d68bd2764ed2241a`; binary SHA-256
`b73491e7cfbe927b28215a87d32102a275f6c7cff1cd06ac4d2bf79299ce1bfd`.
The exact external Counter binary is
`6c48fb52934d49d3774633e32f0b4fb4796b1e2c63925f24167ef0f0c0d761c8`,
release v1.55.0, source `63c487ca724c620f71c129d62129c6fb9109c872`.
It uses embedded NNUE with release AVX assembly and native Threads1/Hash128/
ExperimentSettings=false. No external EvalFile or overhead option is invented.

Reviewed driver SHA-256:
`74640c18d6c98c28bd8110e0515e243af8b4f35fa0d09c063af84efc424bd838`.
Held manifest SHA-256:
`1fe22dcd81ed3b6ee522eeedc217c8c31ec0370e7f7e867348333b2ae7c38785`.
Static checking passed all 21 frozen inputs and both executable hashes. Release
changes only the manifest's release-state field. Target is one immutable
`/home/ehrli/repos/ngn-external-counter-checkpoint-gate-20260913/run-001`.

The NGN phase is strict. The external phase retains the previously verified
Counter-specific fifty-move/threefold PV-warning allowlist; all other warnings
remain fatal. Exact startup, embedded-network and option evidence are checked,
with process attribution supplied by the independent process witness. Both
crash-log parsers handle valid interleaved startup records without relaxing
record counts, timestamps or paths.
