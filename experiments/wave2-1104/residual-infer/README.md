# Residual inference proof

Run `./experiments/wave2-1104/residual-infer/generate.py --check` to verify the
396,860-byte handcrafted H32 fixture. It contains bounded, nonzero small integer
weights in both heads, eight king contexts and all eight output rows. The Go
int64 oracle is separate from production activation and output helpers.
`fixture.json` supplies activation edges, hand-calculated arithmetic cases and
four legal special-move starting positions.

Run `./experiments/wave2-1104/residual-infer/run.py fragment --verbose` from the
repository root to reproduce `result.txt`. This runs only
`GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run '^TestWave2ResidualEval' -timeout 15s`
(with verbose output), scrubs credential variables, limits CPU affinity to two
cores and configures two CPU tensor threads. Temporary files and further proof
logs belong outside the checkout. `result.txt` is the required committed copy
of the captured proof output; it is not a speed sample.

The tests compare compact and wide scores to a straightforward int64 oracle for
both STM and every output bucket, signed truncation, clipped inputs/bottleneck,
separate branch rounding and opposite signs at the final clamp. Loader-derived
bounds cover int16 endpoints, one-step overflow and large-output wide fallback.
Each storage path runs four 128-ply walks with both zero and nonzero residuals,
including castling, en passant, capture promotion, king bucket/mirror refresh,
null aliases, stack growth, undo and reset. The actual embedded H768 NGNN3 bytes
are wrapped in a zero projection with nonzero bottleneck weights and compared
exactly to the original reference. UCI selection, backend reporting, residual-only
reload identity, private workers and corrupt reload fallback are also checked.

`run.py vet` runs the brief's vet gate. `run.py short` runs the required short
suite only on a suitable host: forge hosts are blocked with exit 75 because the
brief prohibits kernel tests and that suite includes unguarded kernel tests.
This is an outstanding gate, not a passing substitute. Default play and embedded
weights stay unchanged. There is no speed KEEP or Elo claim.
