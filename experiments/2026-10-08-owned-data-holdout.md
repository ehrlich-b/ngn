# Owned self-play data preparation — campaign session 6

Session starts **13:26:33 EDT**, deadline **15:06:33 EDT**. Training-tool
preparation only. Accepted engine/net remain **df52dfc / be34532c**, pending
the separately declared delta speed experiment. Existing trainer/controller
sources, binaries and recipes are untouched. No new net, match or GPU job
is launched by this work.

## Missing behavior and scoped change

The NGNP branch of `trainer/train.py` checks that validation and training
paths are disjoint. Two different shards can contain the same occupied
board, so this check alone cannot provide a board-disjoint validation set.
The next owned-self-play experiment needs such a set for checkpoint selection.
This is an optional capability, not a retrospective claim about prior gates.

Add `--ngnp-holdout-modulus N` and `--ngnp-split-seed SEED`, reusing NGN's
existing `ngn-board-mix64-v1` occupied-board hash. N must be a power of two
>=2; zero retains the existing default. Hashing excludes labels, STM, ply,
flags and inactive padding; the same occupied board with either STM stays
together. Flag/ply filtering applies to both partitions. This is not a
game-disjoint or symmetry-equivalence holdout. No claim of those stronger
properties is made.

Read-only maps and bounded blocks are shared; no corpus conversion/copy,
full-corpus index or new training dependency is introduced. Exact eligible
and partition counts, mapped prefix lengths, seed/modulus/hash revision
are stored in config/checkpoints and compared on resume. Frozen record-count
prefixes continue to ignore later appends. Source bytes inside a prefix
must remain immutable under the existing data contract.

## Validation declared before execution

WSL only, isolated `night-20261007/session7/source/`, Go unchanged; Python
3.12 / existing NumPy/PyTorch interpreter. CPU **13,15**, nice **19**, CPU
training smoke only; GPU remains with the longer-H512 run. Require:

- Cross-shard duplicate boards with differing labels/STM remain disjoint;
  filters, deterministic shuffle and all partial batches are preserved.
- Entirely held-out blocks are retained; inactive packed padding is ignored;
  both partitions are nonempty; malformed filtered rows are still rejected.
- Frozen-prefix appends/partial tails cannot change the split receipt or rows.
- One-epoch CPU smoke and identical resume succeed; changed seed/modulus and
  competing shard validation fail. Existing default and archive tests pass.
- Full existing trainer suite passes, including quantization/export/parity
  fixtures and prior archive/NGNN2/NGNN3 behavior. Reconcile tested source
  hashes with the committed patch. No playing-strength acceptance is implied.

## Completed preparation

Detached validation controller **576805** completed **13:33:10 EDT** with
exit zero. Six new controls pass in **10.965 s**; the complete trainer suite
has **47 passes / 3 skips** in **49.887 s**. The three existing CUDA decoder
checks were skipped with `CUDA_VISIBLE_DEVICES=''`, preserving the running
GPU job. This change introduces no GPU decoding operation. The CPU smoke
trains only its tiny synthetic fixture; no strength candidate was trained.

The fixed read-only [sample diagnostic](2026-10-08-owned-data-sample.py)
uses the first **65,536** records from each of fourteen closed gen0b shards:
**917,504** total records. Dividing those shards into seven training/seven
validation files gives **1,830 shared occupied boards** despite different
paths. The new content split gives **916,593 training / 911 holdout** rows,
**889,889 / 889 unique occupied boards**, and **zero shared boards**. Both
split counts cover every eligible row. Prefix hashes and file sizes are
unchanged; this is a bounded diagnostic, not a full-corpus audit or Elo result.

Exact [test receipt](2026-10-08-session6-results/holdout-tests.json),
[source reconciliation](2026-10-08-session6-results/holdout-source.json),
and [data sample](2026-10-08-session6-results/holdout-data-sample.json).
All **24** trainer/fixture inputs match local bytes; both frozen input checks
match. AppleDouble `._` packaging metadata is separately identified and
excluded from Python inputs. Baseline archive SHA-256 **7908ddb1** and tested
overlay **8d44f0ef** remain in `night-20261007/session7/`. The committed
[validation controller](2026-10-08-owned-data-tests.sh) records exact commands,
CPU affinity, interpreter, GPU exclusion and terminal status. No required
engine test was repeated because no Go, assembly, net or runtime behavior
changed; the pending delta controller already passed its required engine and
full short/race suites against this unchanged engine base.

No further behavior candidate is admitted while the longer net's verdict
is pending. After its gate and the exclusive v3 admission window, nominate
the immediate accepted teacher, then declare a bounded datagen calibration
and separate data/training gate before launch. Owned NGNP training can now
select a validation checkpoint without exact-board leakage across shards.
