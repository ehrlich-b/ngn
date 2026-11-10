# Hundredths tuning comparisons

These two recipes are prepared comparisons with unchanged shipped defaults.
They have no match results and make no strength claim. Validate them without
starting games or acquiring the CPU lock:

```
python3 scripts/match_queue.py run --dry-run --jobs experiments/match-queue/tuner-jobs
```

The separate directory keeps the standing queue's two existing control jobs
intact. Selecting or activating these recipes is a separate operator action.
Both roles use engine source `fcfcb048` (the full hash is pinned in each JSON),
the same H768 network, Threads=1, Hash=64 and OwnBook=false. Prepare that source
as the declared binary before any future execution; the runner records the
actual binary hashes. Both comparisons use 10+0.1, SPRT [0,6], at least 400
games and a cap of 10,000 games. No result automatically adopts settings.

The first recipe isolates LMRDivisor=260 (2.60) against 200 (2.00).
The second uses the 2026-11-09T07:50:02Z iteration-1100 checkpoint at
`experiments/burn-1104/spsa/final-vector.json`, SHA-256
`9f5dd547abe98a0eaae5209bbe980ab3dd9da8f471adc04ba6ca33b0fc4d2c13`.
All unchanged-unit parameters use its recorded rounded values. For the three
rescaled parameters, round the original theta times 100: LMRBase=0,
LMRDivisor=259 (theta 2.5921312311575764), NullMoveDepthDivisor=569
(theta 5.688783080507901). T1104CorrMore is forced to 0. The base uses
the compiled defaults on the same engine source and network.
