# Optimized Rodent versus Counter: strength gate

Status: **released, then held by prelaunch CPU interference; zero games started**.
Rodent's accepted output speedup in `1bac0e9` does not establish a strength gain.
This experiment compares two explicit evaluator configurations of the same
accepted engine binary when the host can meet its frozen load contract.

Planned fixed protocol: fresh 100-game Counter/Counter A/A, then 400 paired
Rodent/Counter games, 10+0.1, concurrency 4, one thread per engine, Hash128,
OwnBook false, CPUs 0/2/4/6, and the first 200 frozen openings with color reversal.
The candidate is optimized `rodent-v1.1-anand`; the control is `counter-5.5`.
All other settings and the executable must match. The final manifest must pin
the actual binary, both existing networks, opening corpus and harness sources
before release. The actual freeze and attempted release are recorded below.

Keep the candidate configuration only if the predeclared paired 95% lower bound
is positive and all operational/legal/terminal audits pass. Otherwise shelve at
the fixed cap, with no extension. Do not reinterpret this as absolute Elo or pool
it with historical external cells. No own-network training or extra calibration.

## Harness preparation

The current UCI preflight had an obsolete HCE/NGN-v1-only backend allowlist and
diagnostic regex. The narrow repair admits the exact Counter and Rodent names
while retaining advertised-option membership checks and exact selected-backend
diagnostic equality. Tests prove both names and reject a mismatched diagnostic.
The generic HCE-vs-NGN manifest schema remains untouched; the established fixed
game driver does not use it.

The complete current candidate-match Python suite with these two file changes
passes on WSL: `python3 -m unittest discover -s tests -v`, 29 tests in 3.304s.
The [raw log](2026-09-13-rodent-anand-artifacts/strength-preflight/harness-tests.txt)
has SHA-256 `1398dddd29ddbd35f41cc4b7b53321e3c659834c30c03922b77c479ff2434644`.
An earlier narrower/stale discovery is not the acceptance evidence. No engines
or games were run for this harness-unit check. The previously repaired startup-log
auditor is retained, and the future game run still requires its fresh A/A.

## Frozen release and prelaunch hold

Root reconciled the actual clean build receipt, commit/tree and shared binary,
and independent source review accepted the orientation, bootstrap and runtime
selection/witness contracts. Both configured backend preflights pass. Final source
is `762377947c0907d1ce6be74cb71e01521610bd76`, tree
`59587882012a4bd0a64fdac2678a0b0cee01c143`; both roles use the same CGO-disabled
Linux/amd64 v3 binary, SHA-256
`5547e3481f63789df59f0ab5d059212bb6c61e2780c898f93ee7afb82c73dbf8`.

The released manifest is SHA-256
`0331be68c61106fc5858737e6d6f180071c5b2984a9df9bc2d0e0e7b7a2edd80`.
All fourteen frozen sources and the engine hash passed the held check. The actual
launch then failed closed before creating `run-001`: its ten-second prelaunch
sample measured 1.2472 non-owned CPU cores, above the declared maximum of one.
The leading process was an unrelated Python validation job; root confirmed it
was still live after a five-minute idle wait. It was not stopped or reprioritized.
An earlier WSL service timeout cleared on read-only checks; WSL was not restarted.

No games, A/A result, relative Elo or adoption decision exist. Preserve the
rejected prelaunch sample. Once load clears, a fresh prelaunch attempt may use
the unchanged contract only after confirming no run exists; do not overwrite
the original rejection evidence or start concurrent controllers.

The [status and exact artifact map](2026-09-13-rodent-anand-artifacts/strength-preflight/status.json)
and adjacent release/build files preserve everything needed to resume. WSL gate
directory: `/home/ehrli/repos/ngn-rodent-counter-strength-gate-20260913`.
