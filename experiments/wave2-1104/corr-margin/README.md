# Correction-aware reverse futility experiment

`W2CorrMargin` is a UCI spin option, default 0, range 0–1. When enabled,
only the existing reverse-futility margin grows by
`min(128, abs(corrected_static_before_TT - raw_static) / 2)` centipawns.
Division truncates after taking the absolute value. Unknown and mate-range
raw/corrected scores contribute zero; the existing in-check guard remains.
TT refinement still supplies the cutoff score, but never the correction delta.
Correction learning and the other search heuristics retain their inputs.

From the repository root, run:

```sh
python3 experiments/wave2-1104/corr-margin/run.py
```

The executable wrapper runs the named Go fragment at nice 10 with Go -p 1,
GOMAXPROCS=2, tensor thread limits of 2, and affinity to two available CPUs.
It strips credential variables and reports native wait4 resource accounting.
It writes its output to stdout, leaving the checkout unchanged.
`result.log` is the captured fixture output, including measured accounting.

`off-depth4.json` freezes cold Hash=1 MiB, Threads=1, depth-4 H768 results
captured at the manifest baseline before the production change. The four
cases cover the start position, Kiwipete, a rook ending and en passant.
The test compares move, score, nodes and qnodes with the frozen values.
The network is already tracked; the manifest pins its SHA-256.

The direct search fixtures put beta just inside the old RFP cutoff and check
whether search reaches the move loop. Exact and directional TT bounds,
zero learned correction with a distant TT score, both correction signs,
in-check nodes and mate inputs are covered. Formula tests include integer
extremes and cap boundaries. The existing T1104CorrMore option is independent.

Strength status: **DEFERRED**. This is one frozen formula, with no speed or
Elo claim. Default play remains unchanged pending the specified completed rig
gate. No tournament or strength experiment is part of this fixture.
