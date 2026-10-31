# NGNN3 king buckets — 2026-10-06

Implemented from NGN's own NGNN1/2 at `ce89b4f9` on `own/nnue3`.
No third-party engine/trainer, old NNUE branch, data or weights were read or
reused. Mac: edits and Git only. Execution: isolated WSL worktree
`/home/ehrli/wt/own-nnue3-20261006`, Go 1.25.5, GOAMD64=v3,
GOMAXPROCS<=4, build/test/vet -p 2; Python CPU threads=2. The ongoing
14-worker gen0 datagen job was left running. No matches.

NGNN3 uses the prescribed 96-byte header: `NGN3`, little-endian uint32
version/H/KB/NB/QA/QB/SCALE = 3/H/KB/NB/255/64/400, 64 uint8 king-map
entries; W1 int16 [KB*768,H], B1 int16 [H], O int16 [NB,2H],
OB int32 [NB], IEEE CRC32. KB=1..8, NB=1/2/4/8, H=16..2048 in
multiples of 16; invalid headers/maps/lengths/CRCs fail closed before use.
The default KB=8 map, after Black's vertical flip and own-king horizontal
mirroring, maps half-board files a,b/c,d at ranks 1 and 2 to 0/1 and 2/3;
rank 3 to 4, rank 4 to 5, ranks 5–6 to 6, ranks 7–8 to 7. Input index:
`bucket*768 + rel*384 + T*64 + square`, rel=own/opponent 0/1,
T=pawn..king 0..5. The map lives in the network and checkpoint; custom
JSON maps need no inference code changes. No factorizer.

Each worker owns a lazy (perspective,bucket,mirror) cache with the last
accumulator and piece masks. A changed own-king context refreshes only
that perspective by subtracting/adding the bitboard differences; the other
perspective uses fused incremental rows. Cache snapshots are independent
of stack undo and survive reset with the same immutable network. Existing
AVX2/scalar arithmetic, output buckets, APIs and NGNN1/2 layouts remain.
The trainer constructs bucketed inputs on GPU, checks both kings once at
shard validation, saves the map, and rejects changed maps on resume.

WSL validation: all **35 trainer tests** passed (40.144 s), including CUDA
feature equality for every king square and a custom KB=4 map, symmetry,
selected-row gradients, resume equality and CLI re-export. Full short Go
suite and vet passed; race checks passed for engine (137.258 s), datagen
and NGNP. NGNN3 tests include **2,929 king moves / 2,198 bucket or mirror
transitions**, undo/cache revisits, both castles, promotions, captures,
en passant, null moves, resets, four workers and model reloads. Extreme
int16 weights and H through 2048 match the independent int64 oracle.
The owned H=32/KB=8/NB=8 fixture is **394,436 bytes / 320 exact cases**:
SHA256 `2678a899e7a1fa44cf7e846bbd727149c32d63c1c7296f1b05593a34ebab7427`.
It passes with AVX2 and portable race kernels; NGNN1/2 fixtures are unchanged.

Before/after search identity used six owned positions, iterative and fixed
depth 7, H=256, single thread, cold 16 MiB hash. NGNN1 (NB=1) and NGNN2
(NB=8) each matched all 12 baseline cases exactly in score/nodes/PV/best
move/depth. For format overhead measurement, generated symmetric shared
input rows and tiled them into eight king buckets: NGNN3 then matched all
12 NGNN2 cases exactly. These tied weights isolate inference cost from
changes in search topology; they are separate from the trained smoke net.

Pilot smoke: `/home/ehrli/ngn-data/pilot/*/*.ngnp`, **1,328,267** raw and
accepted records, read-only. H=256/KB=8/NB=8, batch=8192, bf16,
AdamW/LR=0.001, no decay; 128M visits cycle the same pilot data rather than
representing distinct positions. Main smoke: **109.427 s**, 15,708 updates,
loss **0.002353170**, **1,169,729 positions/s** (steady 1,181,225/s).
Calibration used 4M visits in 5.151 s; total GPU training **114.578 s**,
below the ten-minute bound. CLI re-export was byte-identical. All **20**
selected pilot cases, spanning all eight material buckets, matched Go
AVX2 and portable race kernels exactly. Trained export SHA256:
`1548bd4962ebe9d7a5d9334720202d888f2062682ef9eddf5b82c9ac347ac097`.
Every pilot shard's SHA256 was unchanged. This is a plumbing test, with no
strength estimate. Artifacts before timing occupied **57 MiB**, below 2 GiB.

NPS: Threads=1 / GOMAXPROCS=1, CPU 0 pinned, cold 16 MiB hash, two
warmups then **24 paired, interleaved 3-second trials** (four repetitions of
six positions, alternating pair order). H=256, NGNN2 NB=8 versus NGNN3
KB=8/NB=8. Mean NPS: **1,286,837 versus 1,151,061**. Geometric paired
NGNN3/NGNN2 ratio: **0.89544** (median 0.89084), about **10.5% lower NPS**.
Position ratios 0–5: 0.945, 0.848, 0.982, 0.844, 0.873, 0.890. All
14 datagen workers were active; load averages were 16.52 before / 16.28
after. Absolute rates include that shared load. The ratio measures the tied
input-row networks described above, independently of trained-net quality.
Full per-trial receipts are in `nps.json` in the retained run directory.

Reproduction (WSL only, with the existing general-purpose Python environment):

```bash
export GOAMD64=v3 GOMAXPROCS=4 OMP_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 MKL_NUM_THREADS=2
GO=/usr/local/go/bin/go
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
"$GO" vet -p 2 ./...
"$GO" test -p 2 -short ./...
"$GO" test -p 2 -short -race ./engine ./internal/datagen ./internal/ngnp
"$PY" -m unittest discover -s trainer/tests -v
# Use trainer/README.md for the format, map, training and fixture commands.
RUN=/home/ehrli/ngn-data/own-nnue3-20261006
"$GO" build -p 2 -o "$RUN/probe" ./cmd/speedprobe
"$PY" -m scripts.ngnn3_probe prepare --root "$RUN"
"$PY" -m scripts.ngnn3_probe paired --root "$RUN" --binary "$RUN/probe" --cpu 0 --repeats 4 --duration 3s
# scripts.ngnn3_pilot_parity selects the 20 exported pilot cases.
```

Small logs, checkpoints, binaries and full search/timing receipts are retained
at `/home/ehrli/ngn-data/own-nnue3-20261006`. Only owned fixture files,
implementation/tests, reusable validation scripts and this note are committed.
