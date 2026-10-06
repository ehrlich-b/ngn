# NGNN2 material output buckets — 2026-10-05

Extended the NGN-owned NGNN1 implementation from `880b6ff9` in
`own/nnue2`. No other engine, trainer, format, or old NNUE source was read or
reused. The Mac was used only for edits/Git; all execution used the isolated
WSL worktree `/home/ehrli/wt/own-nnue2-20261005`, Go 1.25.5 with
`GOAMD64=v3`, and the existing Python/PyTorch/NumPy interpreter. CPU limits:
`GOMAXPROCS=4`, `go -p 4`; GPU smoke used two CPU threads. No other processes
or worktrees were changed.

NGNN2 header: `NGN2`, uint32 version/H/NB/QA/QB/SCALE
`2/H/NB/255/64/400`, then W1 int16 [768,H], B1 int16 [H], O int16 [NB,2H],
OB int32 [NB], IEEE CRC32 over all preceding bytes. NB is 1/2/4/8. Exact
integer selection is `min(NB-1, (popcount(all pieces)-2)*NB/32)`; both kings
count, division is integer, and malformed populations below two map to zero.
For NB=8: 2..5 pieces -> 0, 6..9 -> 1, ..., 30..32 -> 7. Shared transformer,
SCReLU, quantization, and both truncating divisions are unchanged. Go uses
current occupancy and passes the selected row to the existing AVX2 kernels;
accumulator transitions need no new state. A loader-derived bound across all
output rows retains the safe wide-kernel fallback for extreme int16 weights.

`--buckets 1` retains NGNN1 checkpoint shapes, float arithmetic, and default
export; `--format 2` opts into NGNN2 even with NB=1. GPU decoding computes
buckets directly from occupancy, without transferring CPU counts. Export,
reference evaluation, fixture generation, UCI, and `datagen -net` accept both
formats; invalid headers, lengths, constants, CRCs, and bucket counts fail
closed. Existing public Go APIs are preserved. Existing NGNN1 fixture bytes
and incremental/refresh tests are unchanged.

Validation on WSL:

```bash
export GOAMD64=v3 GOMAXPROCS=4 OMP_NUM_THREADS=4 MKL_NUM_THREADS=4 OPENBLAS_NUM_THREADS=4
GO=/usr/local/go/bin/go
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
"$GO" vet -p 4 ./...
"$GO" test -p 4 -short ./...
"$GO" test -p 4 -short -race ./engine ./internal/datagen ./internal/ngnp
"$PY" -m unittest discover -s trainer/tests -v
```

All checks passed: full short engine suite 11.544 s, full race engine suite
85.549 s, final trainer suite **24 tests in 15.293 s**, including CUDA
decode/bucket checks and one-bucket format-2 CLI export. The final short suite
and vet also passed; the added four-worker NGNN2 race check passed (0.24 s).
The 20-position trained-export parity check also passed with portable race
kernels. Tests cover all
bucket boundaries/STM orderings, NB=1 exact equivalence, selected-row gradients,
float/integer agreement, uninterrupted/resumed NB=8 weights, extreme arithmetic,
corruption, capture/en-passant/promotion boundary crossings, undo/null moves,
and datagen search/provenance. Exact fixture parity: 64 NGNN1 and 64 NGNN2
positions, with portable Go race kernels as well as AVX2. NGNN2 H=32/NB=8
fixture: 50,304 bytes; SHA256
`b08c8c2c88188fd3885d4fed4b95232c3da0fd733b7fea0711ecc7d78bb25992`.

Before/after depth-10 search used the same NGNN1 H=32 fixture, Threads=1,
Hash=16 MiB, OwnBook=false, and a new game for each position. Score, node count,
PV, and best move matched exactly: start position +41 cp / 451,723 nodes /
`d2d4`; middlegame +17 cp / 2,348,051 nodes / `c3d5`; pawn ending -38 cp /
8,312 nodes / `e4f4`. The full PV receipts are in the run directory below.

Pilot smoke: source `/home/ehrli/ngn-data/pilot/*/*.ngnp`, 1,328,267 raw and
accepted records, mapped read-only. A 10M-position calibration took 3.698 s
(617 updates, loss 0.0124317). The bounded main smoke uses H=256/NB=8,
batch=16,384, bf16, LR=0.001, fixed schedule, one epoch with 400M position
visits cycling the same pilot shards, and a 240-second timeout. This is a
plumbing check, not 400M distinct records or evidence of Elo improvement.
Main smoke: **121.974 s**, 24,694 updates, mean loss **0.003447802**,
end-to-end **3,279,396 positions/s** (steady **3,302,095/s**). Total GPU
training time including calibration: **125.672 s**, below the ten-minute cap.
Checkpoint CLI re-export was byte-identical. Go AVX2 matched all **20 pilot
positions exactly**, spanning all eight buckets. Export SHA256:
`64c2cbf06a91d8bfb0254f6f8cdb79420c7656dced1d973abff89f7252d19433`.
SHA256 checks confirmed every pilot input shard unchanged. Run artifacts
occupied **13 MiB**, including both binaries and both training checkpoints.

Small local run artifacts/logs/checkpoints are retained under
`/home/ehrli/ngn-data/own-nnue2-20261005`; only the random fixture pair and
this compact note are committed. No long training or strength match was run.
