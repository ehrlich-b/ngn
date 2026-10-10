# Full NGNN4 training stage, with games held for the key decision

Session38 resumes the original session37 full trial's **unstarted training**
stage after C: recovers to28.325GiB and SPSA moves to forge. Original1660
capacity-blocked terminal records remain untouched. No training stage, pilot,
speed sample or completed evidence is repeated.

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

Results live in `night-20261007/session38-ngnn4-training/`, with exact training
manifest, logs/checkpoints/receipts and append-only status lines. Expect4.57h
from the reviewed pilot projection, then about2h for the later fixed gate.
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

This initial unbounded attempt2637 was stopped00:40:06 after its clean file
cache pushed C: below25GiB. It completed **zero epochs**, with no checkpoint;
its logs and interruption receipt are preserved. Only the immutable corpus's
clean cache was advised away read-only; no bytes or unrelated jobs changed.
The active continuation is [the bounded stage](2026-10-10-ngnn4-bounded.md),
wrapper2791/trainer2804, starting00:41:23 with a4GiB hard job memory cap.
Use that directory; this original controller must not be relaunched.
