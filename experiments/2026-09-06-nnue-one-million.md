# One-million-position training and first games

The larger corpus passed independent conversion and semantic corruption checks, retaining 1,000,015 positions: 801,150 train, 97,294 validation and 101,571 sealed test. Root verified all 3,284 data artifacts. The input remains the pinned Stockfish T80 archive; whole-chain splitting and cross-split model-input quarantine are unchanged. This establishes chain-disjoint partitions, not proof of original-game disjointness.

Training used the existing Chess768→128 shared-FT/SCReLU graph, fixed seed, optimizer, targets and quantization. Eight segments of 16 batches at batch size 16,384 consumed exactly 2,097,152 presentations. The source-derived loader replay traversed the same hash-ordered training BF continuously, visiting 306,298 records twice and 494,852 records three times. All 44 supervised stages passed with no operational failures. Root verified all 11,579 final artifacts, all 18 raw/quantized parity reports (16 FENs each, maximum integer-Go delta 0), and byte-identical raw-derived/quantized NGN files for all 9 candidates.

Validation raw mean loss improved through every segment: .08070519 at initialization, then .07326753, .07115450, .07021664, .06929492, .06841371, .06805370, .06786066, .06722636. Candidate 8 was selected; all 9 were quantization-eligible. Unlike the 100k pilot, this run did not show validation turning upward within its 128 completed updates. Values from the two runs must not be directly compared as scores on the same validation set.

Selected candidate 8 quantization error on validation: mean 5.4384 cp, p99 19.8434 cp, maximum 42.2515 cp. Selection A/B was byte-identical around inaccessible sealed-label mutation, and selection/model/optimizer files were bound before sealed evaluation. Selected sealed original raw/integer losses were .06569321/.06566288; the label-mutated diagnostic changed them to .14272547/.14273716. Sealed loss did not select the checkpoint.

The selected network SHA is 7ac92f670161ca49d42beb2a8a25c36b2250f6f21a0c9e5f6adcf363866183d0 (197,506 bytes). Its four-game smoke against same-code HCE at 10+0.1 again scored 0/4. All 362 plies were independently legal and every game ended naturally by mate, with zero warnings/errors/book signatures and no surviving engines. These games establish operational behavior and give no basis to promote this network. Longer games than the first pilot are not treated as a strength estimate.

Root evidence:
- Data: output/nnue-data-1m-root-review-20260906/receipt.json
- Training: output/nnue-training-1m-root-review-20260906/receipt.json
- Games: output/candidate-smoke-1m-root-review-20260906/receipt.json

Frozen WSL runs:
- /home/ehrli/nnue-public-toolchain-20260906/runs/ngn-data-1m-v1-attempt1
- /home/ehrli/nnue-public-toolchain-20260906/runs/ngn-v0-1m-training-v2-attempt1
- /home/ehrli/repos/ngn-candidate-runner/output/candidate-smoke-1m-20260906-attempt1

The first submitted training request changed after submission because stale documentation was corrected. Root's hash check rejected it before execution. The replacement v2 request records the old reported and retained actual digests and was frozen immutable; root verified all 9,384 exact inputs before authorizing the only 1M training run.

Next: material/calibration diagnostics, independently measured HCE transition fast path, and the frozen 1/2/4/8 HCE SMP matrix. The deployed WSL engine remains 53e4d1b.
