# Retained named-engine games

Reproduce on the data host: `taskset -c 13,15 nice -n 10 python3 scripts/claim_report.py --output ../evidence/claim-report.json`. The JSON contains per-phase medians, end clocks, termination counts and every loss, including null for no comparable centipawn drop. Phase, score and material definitions are recorded in that output.

The screen has 60/150/110 and frozen pooled performance 3582.3957; the stopped confirmation has 94/216/58 across three anchors. The partial confirmation is descriptive and has no verdict or pooled estimate.

| Anchor | Screen W/D/L | Screen performance | Partial confirmation W/D/L |
| --- | --- | ---: | --- |
| Berserk 12 | 1/27/52 | 3483.6189 | no completed games |
| Black Marlin 9.0 | 13/43/24 | 3575.5241 | 9/28/11 |
| Koivisto 6.0 | 30/39/11 | 3651.0433 | 57/83/20 |
| akimbo 1.0.0 | 16/41/23 | 3583.8996 | 28/105/27 |

Berserk screen losses are the largest concentration to inspect: 52 of 80 games. The top two exact phase/material clusters have two games each; the third is one of many tied singletons. These are descriptive leads, not explanations or evidence of causation:

- middlegame, Q1R2B1N2P7/q1r2b1n2p7: `screen/Berserk 12-06-black`, `screen/Berserk 12-26-black`.
- middlegame, Q1R2B2N2P7/q1r2b2n2p7: `screen/Berserk 12-01-black`, `screen/Berserk 12-17-white`.
- endgame, Q0R0B0N1P1/q0r0b1n0p2: `screen/Berserk 12-34-white`.

Compare the paired games in the two repeated clusters first when selecting search and net investigations. The score drop is measured between consecutive candidate searches, so it locates a change in the engine’s own assessment, not the move that caused a loss. The score is omitted when the final UCI info reports mate; no further strength estimate is inferred.
