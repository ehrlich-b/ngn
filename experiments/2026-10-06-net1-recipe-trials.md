# Net1 recipe trials — 2026-10-06

All four trials finished on WSL from `63f88f623b60dc2c8f282d35b3626f6d6cbf53c3`.
NGN code, trainer, NGNP data and weights only; no third-party training scaffolding.
Frozen training prefixes: **35,965,837 positions**; two disjoint closed holdout
shards: **1,895,755 positions**. No new data or training was launched for this report.

Losses below belong to each exported **best epoch**, selected by that trial's
native validation loss. Lambda is the teacher-score fraction of the target;
`val .75` and `val result` are common metrics, so the native `.50` loss is not
directly comparable with a `.75` loss.

| Trial | Architecture | H | Lambda | Epochs | Best epoch | Native val | Val .75 | Val result | Wall seconds | Best NNUE SHA-256 | Parity |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| a | NGNN2, 8 output buckets | 256 | .75 | 32 | 31 | .00768961 | .00768961 | .10015643 | 378.67 | `6a203ffe128541ef422f9344f2906bffbebea6d93931bb1f50e27f44bf2beff6` | 20/20 exact, AVX2; reexport identical |
| b | NGNN2, 8 output buckets | 512 | .75 | 32 | 22 | .00789188 | .00789188 | .10014111 | 578.46 | `f039de18ff0f292d7a69f66ed871abde5d9ae2525c7bf2c924aa2473b96bf63e` | 20/20 exact, AVX2; reexport identical |
| c | NGNN3, 8 king + 8 output buckets | 256 | .75 | 32 | 6 | .00828839 | .00828839 | .10104736 | 681.62 | `9b5ee669ea95bd3d1fc7a668bc574c79d9eb78e5fbd84a985977ee8794d53527` | 20/20 exact, AVX2; reexport identical |
| d | NGNN2, 8 output buckets | 256 | .50 | 32 | 24 | .02543870 | .00979332 | .09593867 | 365.14 | `0b24b1f8ab7a3a99dbea4866a2c9abf0921a4b2e09d601607843dc9115bcc585` | 20/20 exact, AVX2; reexport identical |

Common recipe: seed 20261006, batch 16384, BF16 on RTX 5080, AdamW LR .001,
weight decay .01, cosine decay to .10 of initial LR, full frozen prefix each
epoch (2196 updates), two CPU loader threads. Trial d repeats the architecture
with the lowest common `.75` validation loss using lambda .50. Total training
process wall time: **2003.89 s**; excludes snapshot hashing and export parity.

Trial a has the lowest common `.75` loss. Trial c's best epoch is early and its
loss rises to .00837826 by epoch 32 while training loss falls. Trial d improves
result loss over a, but playing strength is decided by the fixed
[net gates](2026-10-06-net1-gates.md).

Receipts: `/home/ehrli/ngn-data/trials-20261006/` (`results.json`, `snapshot.json`,
`snapshot-hashes.json`, `record-counts.json`, `completed.json`, each trial's
`history.json`, `parity.json`, `parity.log`, `best.pt`, `best.nnue`). All four
network digests were rechecked against the result and parity receipts. Snapshot
SHA-256: `a2b00b085fafea1d9fe060dc2a959d943e68cd9a9a9d9a41ffc03beec52f57fd`.
