# Three-million-position corpus with fixed original holdouts

The bounded expansion is independently accepted as data, not as a trained-model or strength result. It retained 3,000,048 positions after whole-chain cross-split quarantine: 2,406,580 training, 295,751 expanded validation and 297,717 expanded sealed positions. It decoded 5,091,261 source positions in 91,529 complete chains, quarantining 267 chains containing 7,099 eligible records. The first complete accepted chain overshot the 3M total target by 48; the 6M decoded cap retained 908,739 records of headroom.

The original validation BF (97,294 positions) and original sealed BF (101,571 positions) are unchanged byte-for-byte. Future training uses these original holdouts. The expanded holdout files are audit-only.

The fixed-holdout audit checked every retained new training position against all eligible model-input keys from the old validation and sealed chains, including keys from previously quarantined chains. Intersections were zero against 97,408 validation keys and 101,623 sealed keys. It checked the immutable old prefix of 30,158 chains and 1,692,787 source/conversion records and complete-chain retention. Root verified that the audit's successful independent verifier receipt names the actual expanded manifest and frozen verifier executable.

All eight supervised stages completed with exit zero and no survivors. Peak sampled process RSS was 5,816,832 KiB in verification, below the 12 GiB bound. Three jointly modified BF/sidecar semantic corruptions were rejected by independent source reconstruction. The repaired synthetic suite passed one positive case, four real rejection cases and four always-success-stub guards. The previous v1 false-pass harness remains preserved; it never ran a real decode.

Root rehashed all 3,410 final artifacts totaling 19,076,341,611 bytes, independently checked the verifier command and holdout bindings, and rehashed the original holdout bytes. The foreground wrapper actually exited zero after final hashing; the intentionally idle SSH shell is not a surviving job.

Run: /home/ehrli/nnue-public-toolchain-20260906/runs/ngn-data-3m-fixed-holdout-v2-attempt1.

- Expanded manifest SHA: 09b410afb8d443172c87861f4881a8a1158be009f6f759699b8931525bd25244.
- Training BF: 77,010,560 bytes, SHA 8dd4c839076795219998d9651ea3498446422d90540345731be00e9d576eb72d.
- Final artifact manifest SHA: b6ec2e2d181c6947d0111e0ec9784e1b7968e94b2fabeef0fa7795180d4c5923.
- Fixed-holdout audit SHA: 0aab98487cfaee6e558096a5199d06e82e5c9adaf2840262c8cd41f9e53fca22.
- Root receipt: output/nnue-data-3m-root-review-20260906/receipt.json, SHA a23f790a099c27cd1ccfba1e31b69f62484f3cb399a91dc65a972df44d118e42.

The next training proposal changes only the training BF/count from the accepted 1M 512-update experiment, preserving the exact old validation set and all graph/optimizer/target/quantization/selection rules. It has source review acceptance but no training authorization yet. The eight-thread operational game control owns the current exclusive compute window.
