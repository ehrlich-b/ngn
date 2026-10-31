# TT/key pressure diagnosis — session51

Requested by the current plan15:15 amendment, diagnostic only. Factorial
comparison of archived TT-only sourcea49ebfc: original single-slot reference
versus original own four-way patche8440ba3, crossed with original key assignment
versus NGN's accepted current assignment. The new assignment is generated
from NGN's own algorithm into the archived board's feature arrays; numeric
piece codes and hash feature meanings are unchanged. Board/SEE/time/search/
net are identical across cells. Use originalbe34532c, Hash64, one worker.
No legacy implementation is copied into a playing branch or committed;
only wrappers around the frozen cache operations are added in forge staging.

Eight frozen positions from the existing key diagnosis, iterative depth20,
report cumulative counters/actual occupancy at completed depths16 and20.
TT probes/hits, store occupancy, matching/missing-key stores, unchanged
stores, new insertions, evictions and same-key writes. Full-table bucket
histograms and allocated records distinguish cluster occupancy from density.
Miss-store refusal means a store for a key absent before the call left the
bucket unchanged; it is not assumed to be a bug. This is cold-root pressure,
not a full game trajectory. Depth is not standardized work across TT designs.

Raw operations remain byte-for-byte unchanged apart from method renaming.
A bounded controlled fixture checks counter reconciliation and exact depth10
nodes/score/bestmove with wrappers disabled/enabled for each TT design.
Then four fixed cells, no sampling/seed/depth selection after results.
Run source compilation/controls/probes only on forge5–7/2GiB/swap0/nice10,
GOMAXPROCS1, host_pinningON, while SPSA uses8–11/2.5GiB/two games.
Each cell has30min timeout; any incomplete cell stays incomplete, no
automatic rerun or lower-depth substitute. Detached controller writes
per-depth JSONL immediately and native stage status. No games, strength
verdict, adoption, rollback or certification from these measurements.
