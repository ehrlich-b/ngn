# T17 + completed T8 vector, rebuilt on accepted qcap

Prepared and staged; **not launched**. This continues the previously authorized
[interrupted confirmation](2026-09-04-t8-confirmation.md), with its fixed
[24,000-game vector](2026-08-04-t8-spsa-result.md). No new tuning or heuristic
is included. No speculative source or playing defaults changed on main.

The current `r0905texelv2` match must finish and receive its original verdict
first. These exact artifacts can be tested only if qcap remains the accepted
immediate base. If the fitted evaluator is accepted, rebuild the T17/vector on
that new base, verify it and update artifact identities before launching.
Neither candidate is an absolute2800 rating confirmation.

## Scope and artifacts

Isolated source `output/t8-qcap-confirm-20260905`, parent `1387fd6`. Its diff
changes only `engine/search.go` and the existing T17 regression in
`engine/reentrancy_test.go`. The search change seeds the improving reference,
handles its unknown sentinel and applies the exact nine converged defaults plus
their UCI registry defaults. All qcap/draw/castling and offline changes are
inherited. The evaluation model and coefficients remain the accepted baseline.

Raw artifact directory `output/recovery-2026-09-04/`:

- `t8-qcap-candidate.diff` SHA-256
  `d3c20857c7bfa308eb9f11471a0b6d4d5e31bdeacd78097c4c7266ac8cdfc62c`.
- Mac `build/ngn_20260905_t8qcap`:
  `0a42c12d287269251946f4bcbd016a497f200965a445b9806c22d57a982b4345`.
- Windows `build/ngn_20260905_t8qcap.exe`:
  `43b2a2c93d9a773b0a08d8017d611edb7b39d63efe071cb7366c1ac4c11f92fc`.
- Accepted Windows base `ngn_20260904_qcap.exe`:
  `bc84fe0d298a32ff644775b2892ee43356bd94f9e22b6a527dc99cdbb29556f2`.

Go1.26.2, Mac arm64 and cross-compiled Windows amd64/v3. Root reviewed the
diff and logs; full short/race/vet pass. Two fresh-process fixed-depth runs
agree exactly on score/nodes/PV/bestmove: Kiwipete d12=426957/−90/e2a6,
middlegame d12=227604/+6/d4c5, rook ending d16=853016/+103/b4f4. These differ
from the base and establish repeatability only. All14 UCI defaults match the
vector; black EP evasion, forbidden castling, fifty-move mate and clean quit
smoke checks pass on Mac. Root also ran the independent SF oracle:2293 positions,
48926 depth-2 root divides and128 internal search roots pass with state restoration.
Logs are `t8-qcap-{short,race,vet,oracle}.txt`,
`t8-qcap-identity.json`, and `t8-qcap-release-smoke.txt`.

Root pushed the Windows executable to the native SPRT directory and verified
its remote SHA-256. It is staged alongside the installed engine; no default
artifact was replaced and no game process for this candidate was started.

## Frozen game rule for these qcap-based artifacts

Retain the original interaction confirmation rule: paired SPRT **[−3,+3]**,
alpha/beta .05, minimum200 games, maximum1600. Accept only completed H1 and
zero operational failures/flags. H0 rejects; capped inconclusive shelves. No
extension or provisional keep on score. Do not pool the interrupted279 games.

Native Windows Ryzen9800X3D, concurrency8, default affinity, lowpower=false,
10+0.1. Harness `sprt_20260904.exe`:
`63ef86d9b986fdb2c260782149af259e278845d89595d010df3970e6698cc9ed`.
Opening file `sprt_openings.txt`,5000 lines:
`974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222`.

```text
BOXSPRT_BIN=sprt_20260904.exe scripts/boxsprt.sh launch r0905t8qcap -- -new '.\ngn_20260905_t8qcap.exe' -base '.\ngn_20260904_qcap.exe' -tc 10+0.1 -concurrency 8 -lowpower=false -openings sprt_openings.txt -elo0 -3 -elo1 3 -alpha 0.05 -beta 0.05 -maxgames 1600 -mingames 200 -resignscore 900 -resignplies 5 -drawscore 10 -drawplies 10 -drawminplies 80
```

Launch only after
root review of the preceding completed match, verification of immediate base,
all remote hashes and idle native chess workers. Run the native candidate UCI
smoke then, before games; it is not run alongside the active timed match.
The completed same-qcap A/A
`r0905qbaseaa` used precisely these settings and passed; reuse is valid only
while baseline/harness/machine configuration remains unchanged. A new accepted
base needs its corresponding A/A control. No autonomous handoff currently
launches this candidate.
