# Search parameter exploration

These ranges are exploratory; defaults remain the shipped values. Perturbation at the end of an SPSA campaign is max(0.5, (max - min) / 20); earlier iterations scale it by (iterations / iteration)^0.101.

| Parameter | Default | Min | Max | End perturbation |
|---|---:|---:|---:|---:|
| LMRBase | 0 | 0 | 3 | 0.5 |
| LMRDivisor | 2 | 1 | 6 | 0.5 |
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
| NullMoveDepthDivisor | 6 | 1 | 12 | 0.55 |
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

LMRBase is added in plies; LMRDivisor divides the logarithmic product. Time percentages are in hundredths. Capture SEE has a fixed threshold and null-move pruning has no evaluation reduction term in this version.
