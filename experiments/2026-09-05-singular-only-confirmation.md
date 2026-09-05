# Coherent child depth with singular extensions only

Candidate built and root-reviewed, **not launched or accepted**. This is the single candidate
selected from the [fixed four-arm diagnostic](2026-09-05-depth-policy.md).
It is a search heuristic experiment, not an automatic correctness keep.

## Behavior and prerequisite

Remove positive check and passed-pawn extensions. Preserve singular positive
and negative extensions, the extension budget, all legality/terminal/stop rules,
evaluation, ordering and the other pruning/reduction policies. Initial nonfirst
scouts use the already-computed child depth minus the reduction; remove the old
reclamp that discarded extensions. The diagnostic D variant combined these
changes explicitly. It produced six changed choices without a negative SF
child-score change on the23 frozen cases, but that selected sample does not
establish strength. Canonical tree cost increased on two of three roots.

The isolated candidate must contain no diagnostic switches, per-node logging,
extra counters or UCI changes. Require full node/score/PV/bestmove identity with
the D overlay on all three canonical roots and all23 full-history400k-node cases,
Hash64/Threads1, plus full short/race/vet and the independent legal-move oracle.

The current classical-fit verdict and T17+T8 confirmation come first. This
qcap-based candidate can launch as built only if qcap remains the accepted
immediate base. An accepted intervening change requires rebasing/rebuilding,
repeat validation and replacement frozen artifact identities before launch.
Do not silently test this old candidate against a newer base.

## Prospective game rule

Paired SPRT **[0,+10]**, alpha/beta .05, minimum200 games, maximum1600 games.
Accept only a completed H1 with zero operational errors and zero clock losses
on either side. H0 rejects; capped inconclusive shelves. No extension, pooling
with other variants, provisional keep on positive sign, or parameter changes
after results begin. No fitted evaluation or quiet-promotion patch is bundled.

Native Windows9800X3D,10+0.1, concurrency8/default affinity, lowpower=false.
Use `sprt_20260904.exe` SHA-256
`63ef86d9b986fdb2c260782149af259e278845d89595d010df3970e6698cc9ed`
and5000-line book `sprt_openings.txt` SHA-256
`974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.
Standard adjudication: resign900cp/5plies, draw10cp/10plies after80plies.
The cap is approximately100 minutes at recent throughput; no other native
Windows/WSL engine CPU work may overlap.

Require a completed A/A on the exact baseline/settings. The completed
`r0905qbaseaa` may be reused only if qcap, harness, book and machine configuration
remain unchanged. Any accepted different base requires its own A/A. Independently
check idle workers, remote hashes and native candidate UCI smoke before launch.
Record the final command/run name and all identities before the first game.

## Frozen isolated artifacts

Source archived from `98968fc823c91a501fc985cfad9cfda6a20b10a6` into
`output/singular-only-candidate-20260905/src`. Root compared every tracked
Go/module file with main: only `engine/search.go` differs. The patch removes
33 lines and adds8. It preserves the D diagnostic's nonnegative scout-depth
floor so a negative remaining-depth value cannot fall into the raw-eval branch.
No diagnostic data structure, switch or UCI edit remains.

- Normalized applicable diff `output/singular-only-candidate-20260905/singular-only-runtime.patch`:
  SHA-256 `efbafd010d1dcb552f6c3f3a401ba215b67000da4acc20c69a914f2497a15fc7`.
  Root `git apply --check` passes without modifying main. The original pathful
  `singular-only.patch` is retained separately.
- Mac `build/ngn_20260905_singularonly`: SHA-256
  `e9542e597ca327ee9bf0a539c906e61d84d187dcd779efd740c680374d581641`.
- Windows `build/ngn_20260905_singularonly.exe`: SHA-256
  `0a4a8ec0db97e321ec2e6e82cd73975cec98226416f22f627992d9961594740e`.
  Go1.26.2, Windows/amd64/v3, CGO disabled; root inspected build metadata.
- `plain-D-identity.json`: SHA-256
  `c7bee2ce98a60655e9ec1a5553f10168a11d68e757f2de0c1873811ca86b7d75`.
  All26 node/score/full-PV/bestmove comparisons pass; root checked the assertions
  and raw records. Logs in the same isolated directory show full short/race,
  vet and independent oracle passes. The oracle ran rather than skipped:
  2,293 positions,48,926 depth-2 root divides,128 raw search roots, all matching
  with state restoration.

The Windows binary is built locally, not pushed or installed. Immediate accepted
base, A/A basis, native smoke, remote hashes and run name remain pending the prior
completed game verdicts. No game launch is authorized by an artifact name alone.

The candidate's self-play gate cannot establish the absolute2800 goal. An
accepted engine gain proceeds to the [prospective external confirmation](2026-09-05-2800-pin-plan.md).
