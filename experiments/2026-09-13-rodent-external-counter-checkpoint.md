# Optimized Rodent NGN versus external Counter 5.5

Status: no accepted result. The released attempt is preserved. A clean `run-003`
package is hash-checked and held pending independent release plus an exclusive
host window.

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

## Recovered execution disposition

The September 14 execution did not produce an accepted checkpoint result.

- `run-001` stopped before any game because the first driver revision expected
  NGN's startup log in the wrong directory. The directory is preserved. No
  engine or match result was involved.
- `run-002` used the corrected role layout. Its A/A phase completed 100 games / 50
  pairs, scored exactly 50%, and passed its legal match audit. The candidate
  supervisor then completed normally after 3,359.073 seconds; fastchess exited
  zero, its process witness recorded 478,456 observations across the exact two
  frozen binary hashes on CPUs 0, 2, 4 and 6, found no violations, and found no
  survivors.
- The candidate timing evidence nevertheless failed the frozen interference
  rule. Across 672 five-second samples, 509 exceeded one non-owned CPU core and
  the peak was 2.1869 cores. Separate `python` and `opencode` work dominated the
  load. This is sustained interference, not a marginal sample.
- The driver rejected the phase before candidate legal/result audit or paired
  Elo calculation. The candidate PGN is frozen but must not be scored, audited
  for a rating result, pooled, extended or used to choose whether to rerun. No
  external result or rating increment is accepted from either directory.

The exact `run-002` evidence remains on the authorized host at
`/home/ehrli/repos/ngn-external-counter-checkpoint-gate-20260913/run-002`.
Key SHA-256s are:

- frozen manifest: `c2b707223e173d84e26ac5ad3ecb1f3fdbc44cd04759635cb3dacf9ee956c52f`;
- A/A match audit: `fea84ee74c0cd565f4294dea004534092a0a9dd819b938c1ad7a0779275e83b8`;
- candidate CPU receipt: `d1526e80427f0ec77507ed5fade8c6c77163830527d5c4017bd6df9d1caedab9`;
- candidate supervisor: `6e5bd45597b950b2815451819f3ffc5ae8dd83c0aa33bc06ddb6c54702081d5a`;
- candidate process witness: `d27d94bc91cf835b469ff90284cae73422c53f8426f9fa6bf8fd2394e2031299`;
- frozen, unaccepted candidate PGN: `026956e2f95f397f5c60e6aab974cb5a2d6b91588d1f3759aee7bc5ea5116bcd`.

One operational replacement remains scientifically admissible because the
rejection was declared without reading or computing the candidate score. It is
not released yet. A replacement must use a fresh run directory, byte-identical
engines/models/openings and the same fixed 100+400 protocol, preserve both
failed attempts, and receive a fresh static/release review. Most importantly,
the host must be reserved for the whole A/A and candidate interval: a short
quiet preflight is insufficient when another scheduler can start CPU work
mid-match. Do not launch while the WSL hopper or other sustained jobs are active,
and do not weaken the one-core interference rule to make a run pass.

## Held operational replacement

On September 18, `run-003` was prepared without executing either engine or
reading any candidate result file. The held package is at
`/home/ehrli/repos/ngn-external-counter-checkpoint-20260913/output/external-counter-checkpoint-20260913/held-v3`.
Its driver is byte-identical to the reviewed v2 driver, all 21 frozen inputs and
both engine binaries passed the driver's held static check, and the intended
immutable run root does not exist.

The manifest comparison passes after removing exactly two declared differences:
the fresh `run-003` root replaces `run-002`, and an `operational_replacement`
object records the non-score-conditioned CPU rejection. The protocol, engines,
models, openings, source hashes, options, seeds, caps and claim boundary are
otherwise unchanged.

- held driver SHA-256: `0d204c66d49f930bcba1b09a88a60530ef260f51a659d88e37333b26d15f6344`;
- held manifest SHA-256: `7d4c8cb12ad4616ad108182807bbcc4a2292a1e2a82647e22fb009eb0ed09ce0`;
- held replacement receipt SHA-256: `8355e7c4b78977e41afc7f2c264e7994fb5183a4742c7bf9cdaee3000d252452`;
- prior v2 layout-preflight receipt SHA-256:
  `f01c64c69d3990880e8c4b9dcfcc966a738860191d353e91e81b90a227e1e995`.

This is preparation, not release. `launch_authorized` is false. Remaining gates
are an independent static review that creates the released manifest and a real
exclusive WSL CPU window spanning both phases.
