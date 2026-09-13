# Existing Rodent Anand network: full-refresh compatibility

Active objective remains **raise the rating to 3300**, using existing networks.
This inactive evaluator compatibility slice is necessary implementation work,
not an accepted playing-strength improvement or completion of that objective.

## Admission

Immediate source base: `270c73563139996d03f3c34b2840015b8e130df9`.
Root read the [exact-network contract](2026-09-13-rodent-anand-integration-contract.md)
and an independent Sol reviewer confirmed the released binary's practical
full-refresh raw oracle. Sol is authorized to implement Slice A only:
strict immutable loader, portable full refresh, exact raw output and release
material scaling, with deterministic exact-release corpus verification.

The network remains an external file, SHA-256
`5f7480b56538e9e64ee02fb5b4842bf0394ee44b707dae3a63ec7f902abd0afb`.
The raw oracle is the exact testers executable, SHA-256
`9c68d7b39dc933eff5fd2da88bd4be1b0d009f6d2add7485c121cfc0d1308531`.
Tagged source is supporting mechanism evidence, not a reproducible-release claim.
In particular, the released Anand settings use scale 192 rather than the tag's
default 400. Preserve the release's asymmetric material formula and absence of
pre-100-ply rule-50 attenuation as compatibility; do not quietly fix either.

## Boundary

- Standalone `rodenteval` package; no engine/UCI selection or default change.
- No incremental context or AVX2 in this first slice.
- No model embedding, redistribution, training or deployment.
- Synthetic malformed-input/arithmetic/mapping tests plus a frozen 50–100 FEN
  corpus generated from the exact released executable, without score selection.
- Distinguish release-observed raw output from final static values derived using
  its independently inspected machine-code material formula.
- Stop on unexplained mismatch; never fit a scale to make the corpus pass.
- Incremental parity, realistic speed and paired-clock strength remain required
  before a future playing-backend adoption. Rodent's rating does not transfer
  automatically with its weights.

## Status

Slice A implementation and independent oracle generation are in progress on
WSL. No behavior candidate is admitted, no game run has begun, and 3300 remains
unverified.
