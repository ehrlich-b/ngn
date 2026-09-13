# Optimized Rodent versus Counter: strength gate

Status: **preparing, not released**. Rodent's accepted output speedup in
`1bac0e9` does not establish a strength gain. This experiment will compare two
explicit evaluator configurations of the same accepted engine binary.

Planned fixed protocol: fresh 100-game Counter/Counter A/A, then 400 paired
Rodent/Counter games, 10+0.1, concurrency 4, one thread per engine, Hash128,
OwnBook false, CPUs 0/2/4/6, and the first 200 frozen openings with color reversal.
The candidate is optimized `rodent-v1.1-anand`; the control is `counter-5.5`.
All other settings and the executable must match. The final manifest must pin
the actual binary, both existing networks, opening corpus and harness sources
before release. No games are authorized by this preparation record alone.

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
