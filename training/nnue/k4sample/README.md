# NGN K4 archive sampler

This Rust package is the bounded-memory source sampler for the owned K4 NNUE
run. It decodes the pinned Stockfish T80 binpack, assigns complete encoded
chains to a deterministic 97/1/1/1 split, and identifies positions by the exact
`ngn-k4-input-v1` architecture input consumed by the Go labeler and evaluator.

The pipeline is deliberately staged:

1. `ngnk4scan` performs one complete archive scan. It writes sorted fixed-width
   key runs and chain metadata only at complete-chain boundaries. Every chunk
   has an immutable JSON receipt, so an interrupted scan can validate its
   finished chunks, replay the source to the last boundary, and continue. Each
   attempt stops at the first complete-chain boundary after the frozen four-hour
   cap.
2. `ngnk4select` externally merges key runs, quarantines whole chains for
   cross-split or historical-holdout conflicts, chooses unique K4 inputs by
   deterministic chain priority, and emits the pilot/main label shards.
3. `ngnk4sampleverify` independently replays the source and verifies the final
   selection, split isolation, historical exclusions, identities and receipts.

All three stages are implemented. The verifier independently reimplements the
split, priorities, eligibility predicate, K4 key and position ID; reproduces
conflicts, quarantine, owner choice and chain-prefix selection; checks both
owner orderings as the same cryptographic multiset; and replays the source while
matching every selected record to the emitted label shards.

The production run is additionally gated outside this package on at least
60 GiB free on the Windows host volume and must stop before 25 GiB. The pinned
offline Rust dependencies live in the existing WSL toolchain; no new download
or duplicate source archive is required.

`run_wsl_stage.sh` enforces those physical-volume gates, confines the command to
two low-priority CPUs, and supervises the process group with a one-minute disk
check. Use it for every production scan, selection and verification command.
