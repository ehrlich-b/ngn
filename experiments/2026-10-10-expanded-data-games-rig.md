# Expanded-data strength gate — optional rig lane, session49

Preparation only, no net nomination or games. The13:05 plan chooses whichever
lane is free first after review. Retain the unchanged forge runner for c2;
this stage supplies the same fixed experiment for rig0–11/c6/nice19.

Stage `session49-data-games` under the rig campaign root. Use the identical
accepted9bfbae4 PGO binary on both sides, basebe34532c, originalsprt9b0002f8,
openings974e4b5a and fixed-gate parser. Its previous source/build checks are
already reviewed; no rebuild. Full58/14.848B training, original minimum
holdout checkpoint, byte-identical reexport and20 Go integer parity remain
mandatory on forge. A session reviews the receipt before nomination.

After review, copy the completed forge `training-receipt.json`,
`postprocess.done/exit` and `trained/best.nnue` into this stage's
`expanded-data/`. Create `nomination.json` with actual source/build/profile/
base-net hashes, reviewed receipt hash/review path, machine/Go/environment,
CPU mask0–11, c6, memory cap4GiB/swap0, host_pinning:not-applicable and
concurrent workloads. Launch only when rig SEE has reached terminal (or
has been interrupted without verdict); never overlap these two game jobs.
Training12,14/GPU has priority under4GiB. Keep WSL near12GiB total, C:>=25GiB
and hopper untouched. Forge SPSA is on a separate machine and need not pause.

Use a detached scope with AllowedCPUs0–11/CPUQuota1200%/MemoryMax4G/
MemorySwapMax0/nice19. Same GOMAXPROCS1/Threads1/Hash64/OwnBookfalse/
UseNNUEtrue, TC10+0.1. Write concrete argv/hashes before games. Fresh A/A40
must pass printed AND unrounded paired intervals containing zero and zero
faults; then exactly400 games/200 reversed pairs. A/A1h/gate4h deadlines.
Keep only complete printed AND unrounded lower95>0, zero flag-outs,
crashes, illegal/no-move results and intact inputs; otherwise shelve.
No extension, cross-machine pooling, automatic adoption/default-net change
or claim. Operational logs still require terminal session review.

The rig runner differs from the prepared forge runner only in root/input
receipt location, machine, nice level and status stage. Its statistical
parser, counts and decision rule are unchanged. Scope/topology are pinned
in the nomination. Never launch both lane preparations for the same verdict.
