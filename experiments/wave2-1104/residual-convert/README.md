# Exact NGNN3 to NGNN6 lift

`trainer.residual_export` preserves the NGN-owned KB8/B8 source's W1,
B1, linear weights, linear biases and 64-byte king map without quantizing
again. H32 and H768 are the two reserved NGNN6 widths. Its default PCG64
seed is 20261104: draw int64 bottleneck weights uniformly in [-8,8], then
int64 biases in [-1024,1024], and cast to int16/int32. Both projection
weights and projection biases start at zero. This yields the exact original
integer function for every input; the bottleneck is deterministic and nonzero.
The exporter also accepts explicit integer head arrays via `residual_bytes`
and validates their shapes, representations and full affine bounds.

The independent `trainer.residual_refeval` parser and NumPy int64 evaluator
implement the binary specification in `docs/ngnn6.md`. They share no parser,
feature code, arithmetic helpers or output kernels with the exporter, legacy
reference or Go engine. Position packing/FEN parsing uses `trainer.ngnp`.
All divisions truncate toward zero, and the branches are added before the
final score clamp. A later trained nonzero model must pass this same oracle;
equal zero-residual scores imply no loss or strength result.

The committed H32 packet is generated entirely from NGN-owned seeded integer
arrays. It contains NGNN3, its deterministic zero lift, and a separate seeded
nonzero residual. `records.ngnp` contains 32 records: eight material buckets
times two horizontal mirrors times both turns. `parity.json` records wire
hashes/sizes, accumulator values, branch scores and special transitions.
Every nonzero fixture score differs from its NGNN3 score, so treating NGNN6
as NGNN3 fails the parity test. Go checks compact and wide accumulators,
refresh and incremental frames, captures, both castles, both promotions,
both en passant directions, null/undo, and king bucket/mirror crossings.

With NumPy installed, run from the repository root:

```sh
python3 -m unittest trainer.tests.test_residual_export
GOMAXPROCS=2 go test -p 1 -count=1 ./engine -run '^TestWave2ResidualParity' -timeout 15s
python3 experiments/wave2-1104/residual-convert/generate.py --check
```

The executable runner pins affinity to two available CPUs, sets CPU tensor
threads and GOMAXPROCS to two, scrubs credential variables, runs subprocesses
at nice 10 and reports native wait4 resource measurements. It uses shared
Go caches and an external temporary directory. For a cached, isolated NumPy
installation, set `PYTHONPATH` to that installation's directory.

```sh
python3 experiments/wave2-1104/residual-convert/run.py python
python3 experiments/wave2-1104/residual-convert/run.py parity
python3 experiments/wave2-1104/residual-convert/run.py h768 --output-dir ../scratch/residual-convert
```

The H768 command converts `engine/nets/ngn-h768-2dcb6d93.nnue` once into
external scratch, verifies its pinned hash, and reuses the verified scratch
file on subsequent invocations. It compares all 32 records against the
independent reference, legacy reference, and Go compact/wide evaluation.
It then runs four production depth-four paired searches from
`search-fens.json`, fresh 1-MiB hash tables and one search thread, checking
score, best move, PV and node count exactly. The Go test skips this explicit
scratch input in the ordinary short suite; the H768 command supplies it.
`h768-receipt.json` contains the measured formats, hashes, payload sizes,
integer scores and search results; each subsequent run must match it.
`result.txt` is captured execution output from the Python tests and complete
Go fragment, including H768. No large converted model or embedded default
changes are included. No speed, loss or Elo claim follows from these checks.

To convert another owned source explicitly:

```sh
python3 -m trainer.residual_export owned.ngn3 scratch.ngn6 --receipt scratch-receipt.json
python3 -m trainer.residual_refeval scratch.ngn6 '7k/8/8/8/8/8/8/K7 w - - 0 1'
```
