# Full NGNN4 training stage, with games held for the key decision

Session38 resumes the original session37 full trial's **unstarted training**
stage after C: recovers to28.325GiB and SPSA moves to forge. Original1660
capacity-blocked terminal records remain untouched. The initial unbounded attempt completed zero epochs; no completed pilot,
speed sample or training evidence is repeated.

Use the same verified trainer2acf834 and frozen58x256M H512/L16/KB8/NB8,
dense/BF16/batch16384, factorizer-off recipe in
[the original manifest](2026-10-09-ngnn4-full.md): sameT80 data/content split,
holdout, seed, loss/optimizer/decay and best full-holdout selection. Fresh
weights/RNG/output, no pilot initialization. The original driver differs only
in output directory and annotating that the later game base is pending the
current plan's higher-priority key confirmation.

Run GPU training on rig12,14/nice19, with memory snapshot and C:>=25.5GiB.
SPSA/CPU tests stay on forge; rig has14GiB available before launch, hopper
untouched. After all58 epochs: byte-identical best reexport,20 exact integer
Python/Go cases and the already frozen four-block actual-trained cost report.
These remain training/parity/cost evidence; no game or adoption follows
automatically.

Results live in `night-20261007/session38-ngnn4-bounded/`, with exact training
manifest, logs/checkpoints/receipts and append-only status lines. The original pilot projects4.57GPUh; the4GiB cache cap may change wall time.
Nominal completion is about05:15–05:30 EDT October10, then parity/cost.
Reestimate from complete epochs; the later fixed gate takes about2h.
The higher-priority key gate uses13,15 and finishes before architecture games.
Review its completed result first. If it passes, integrate its provisional
source and rebase/check the architecture engine on that actual accepted base
before freezing architecture binaries/games. Trainer/weights are unaffected
by a Zobrist initializer change; do not retrain or replay the cost admission.
If it shelves, useac658e2. No time/SPSA vector or net adoption during training.

Only after the key disposition and full training/parity review: a new actual
source/binary/net manifest, fresh A/A40 then exactly400 paired10+0.1 architecture
games by the original lower95>0/zero-fault rule. No automatic merge, extension,
claim, publication, external message or Mac computation.

Resource correction before any completed epoch: run this same full recipe
in its own user-systemd scope, MemoryMax4GiB/MemorySwapMax0/CPUQuota200%,
AllowedCPUs12,14/nice19. The unbounded attempt2637 completed zero epochs
and is preserved; only its corpus clean cache was advised away read-only.
The limit bounds file cache as well as RSS, preserving the hostC: floor.

Detached wrapper **2791** starts00:41:23 EDT October10; trainer **2804**.
Scope`ngn-gpu-s38.scope` is active with the limits above. At00:46 C: has
30,414,458,880bytes/28.326GiB, GPU61%/3409MiB, scope memory4.23GB.
The cap includes file cache; original2637 receipts remain in
`session38-ngnn4-training/`. Key wrapper2847 uses its own2GiB scope13,15.
The controller appends terminal/status records and stops after training,
reexport/parity/cost; it never starts games. Do not launch duplicate jobs.
