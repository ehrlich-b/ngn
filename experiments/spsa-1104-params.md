# Search parameter exploration

These ranges are exploratory; defaults remain the shipped values. Perturbation at the end of an SPSA campaign is max(0.5, (max - min) / 20); earlier iterations scale it by (iterations / iteration)^0.101.

| Parameter | Default | Min | Max | End perturbation |
|---|---:|---:|---:|---:|
| LMRBase | 0 | 0 | 300 | 15 |
| LMRDivisor | 200 | 100 | 600 | 25 |
| HistoryLMRDivisor | 2500 | 500 | 6000 | 275 |
| HistoryLMRCap | 2 | 1 | 5 | 0.5 |
| RFPMaxDepth | 7 | 1 | 12 | 0.55 |
| RFPImproving | 1 | 0 | 3 | 0.5 |
| FutilityImproving | 1 | 0 | 3 | 0.5 |
| LMPQuadratic | 1 | 1 | 4 | 0.5 |
| LMPNonImprovingDivisor | 2 | 1 | 5 | 0.5 |
| LMPMaxDepth | 9 | 2 | 14 | 0.6 |
| SEECaptureDepth | 4 | 1 | 8 | 0.5 |
| SEECaptureThreshold | 100 | 0 | 400 | 20 |
| SEEQuietDepth | 4 | 1 | 8 | 0.5 |
| SEEQuietSlope | 80 | 0 | 300 | 15 |
| HistoryPruneDepth | 3 | 1 | 8 | 0.5 |
| HistoryPruneThreshold | 1000 | 100 | 4000 | 195 |
| HistoryPruneMoves | 3 | 0 | 10 | 0.5 |
| NullMoveDepthDivisor | 600 | 100 | 1200 | 55 |
| NullMoveOffset | 1 | 0 | 4 | 0.5 |
| IIRDepth | 4 | 2 | 10 | 0.5 |
| IIRPhase | 8 | 0 | 24 | 1.2 |
| IIDDepth | 4 | 2 | 10 | 0.5 |
| IIDReduction | 2 | 1 | 4 | 0.5 |
| SingularMarginDivisor | 32 | 8 | 96 | 4.4 |
| SingularTTDepthOffset | 3 | 0 | 8 | 0.5 |
| HistoryRateCap | 400 | 64 | 512 | 22.4 |
| HistoryRateScale | 1 | 1 | 4 | 0.5 |
| HistoryBonusCap | 2048 | 512 | 8192 | 384 |
| PawnCorrectionWeight | 3 | 0 | 10 | 0.5 |
| NonPawnCorrectionWeight | 3 | 0 | 10 | 0.5 |
| MinorCorrectionWeight | 3 | 0 | 10 | 0.5 |
| SoftIncrementPercent | 80 | 0 | 150 | 7.5 |
| HardSoftMultiplier | 4 | 1 | 8 | 0.5 |
| HardBankPercent | 30 | 1 | 100 | 4.95 |
| NullMoveR | 4 | 2 | 5 | 0.5 |
| NullMoveMinDepth | 2 | 1 | 6 | 0.5 |
| LMRFullDepth | 3 | 1 | 6 | 0.5 |
| LMRMoveThreshold | 2 | 1 | 8 | 0.5 |
| FutilityMargin | 101 | 50 | 160 | 5.5 |
| FutilityMaxDepth | 8 | 1 | 8 | 0.5 |
| ReverseFutilityMargin | 69 | 40 | 300 | 13 |
| DeltaMargin | 261 | 0 | 400 | 20 |
| SingularDepth | 8 | 4 | 12 | 0.5 |
| SingularMargin | 67 | 16 | 256 | 12 |
| ExtensionBudget | 29 | 4 | 64 | 3 |
| AspInit | 18 | 4 | 40 | 1.8 |
| AspMult | 171 | 110 | 250 | 7 |
| LMPBase | 3 | 2 | 16 | 0.7 |

LMRBase, LMRDivisor and NullMoveDepthDivisor use hundredths: LMRBase=100 adds one ply, LMRDivisor=260 divides the logarithmic product by 2.60, and NullMoveDepthDivisor=600 gives floor(depth / 6.00). Defaults preserve the original search arithmetic. Time percentages are in hundredths. Capture SEE has a fixed threshold and null-move pruning has no evaluation reduction term in this version.

SPSA rejects any selected range with fewer than 20 unit steps (`max - min < 20`), including 0/1 flags. Use `-params LMRBase,LMRDivisor,NullMoveDepthDivisor` to choose a continuous subset; omitting `-params` requests the entire registry and fails while it contains discrete ranges. Fixed `-engineoption` settings are not tuned; compare discrete choices with separate A/B jobs. For a synthetic check without games:

```
go run ./cmd/spsa -params LMRBase,LMRDivisor,NullMoveDepthDivisor -selftest
```

Old checkpoints bind their original units, parameter set and executable hashes. They cannot resume with this tuner. When preparing a new recipe from an old theta, multiply these three theta values by 100 before rounding; never reinterpret an old rounded divisor of 3 as the original fractional theta.
