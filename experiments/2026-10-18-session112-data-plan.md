# Session112 disk and new-data proposal

**Proposal only. No deletion, training-data download, conversion or bulk
generation is authorized by this document.** The governing root note is
October18 11:25 in `ngn-campaign/plan-2026-10-09.md`. WDL.25 is disarmed;
no further recipe cell on the existing T80/T76/T74 corpus is queued.

Recommendation: choose the T60/T70 data option below, with a fixed storage
budget and unchanged training recipe. First decide whether the old full-T74
mapped corpus may be archived elsewhere or retired. Discarding alternate
checkpoints alone cannot finance a new corpus. Keep the CPU evidence queue
moving while that decision is pending.

## Inventory and protected material

The read-only rig inventory completed at **12:20:21 EDT October18**, pinned
to CPUs13,15/nice19. It inspected metadata for **504,445 files**, including
**2,499 data/model/position/game files**, under `/home/ehrli/ngn-data`.
Unique-inode allocated storage is **241.171GiB**; path-size total is
240.047GiB. Allocation includes filesystem block rounding; neither number
is a promise of host-C recovery. Host C: had **30.368GiB** free at12:15,
against the25GiB floor.

The [data inventory](2026-10-18-session112-results/data-files.tsv.gz) records
every matching data file's path, logical/allocated size, mtime, creating
session inferred from its directory, live references, historical manifest
references and protection reasons. The [compact summary](2026-10-18-session112-results/inventory-summary.json)
contains group totals and large-file rows. The full all-file inventory
(`all-files.tsv.gz`,9.17MB) and full reference lists stay on the rig in
`/home/ehrli/ngn-data/night-20261007/session112/`.

Static path references were read from small manifests/configs, with source
snapshot directories excluded where recognized. The six live-manifest
roots cover the paused tune, its resume receipt, Session109 and S4. A
missing literal reference does not prove that a file is disposable; scripts
can construct paths. No corpus bytes were read or rehashed. Creating session
is a path inference, not a verified filesystem birth record. Files outside
the current campaign and files modified within24h are excluded from cleanup.

| Data | Creating session | Logical GiB | Live job reference | Accepted-net reference / disposition |
|---|---|---:|---|---|
| `night-20261007/session57-data-step2/t74full.bf` | 57; full conversion/copy, Oct11 | 170.778 | None in current tune or timing manifests | **Yes:** accepted Final115 training input; retain unless Bryan approves archive/retirement |
| `night-20261007/session19-data/t76.bf` | 19, Oct9 | None in current tune or timing manifests | **Yes:** accepted Final115 training input; retain |
| T80 `archive-a1/corpus-a1/train-archive.bf` | Predates this campaign; outside inventory root | 70.759 | No current training job | **Yes:** accepted Final115 input; outside deletion scope |
| `night-20261007/session80-training/trained/latest.pt` and final exports | 80 | About0.047 combined | Inference uses the copied net below | **Yes:** original selected Final115 checkpoint/export; retain |
| `night-20261007/session82/net-final115/candidate.nnue` | 82, copied accepted final | 0.00588 | **Yes:** tune, S4 and boundary | **Accepted81215229; retain** |
| `night-20261007/session88/spsa-warm/` | 88, active warm tune | Small | **Yes:** paused1600 checkpoint and native resume | Preserve state, RNG, logs, manifests and binary dependencies |

T74+T76 alone total **222.585GiB**,92.3% of this root's allocated storage.
They are canonical inputs, not duplicate reconstructions. The old raw T74
binpack was already removed by Session111; it offers no further recovery.
No mapped T75 corpus exists under the inventoried root. Forge stays closed.

## Exact small cleanup proposal

The [six-path proposal](2026-10-18-session112-results/cleanup-proposal.json)
offers only the unselected `best.pt` and `best.nnue` in each of:

| Directory under `night-20261007/` | Creating session | Bytes proposed | Why these two files are candidates |
|---|---|---:|---|
| `session84-training174/trained/` | 84 | 44,207,278 | Final174 selected; cell shelved |
| `session87/mixture/volume-control/trained/` | 87 | 44,194,158 | Final115 selected; cell shelved |
| `session87/mixture/moderated-t74/trained/` | 87 | 44,194,158 | Final115 selected; cell shelved |

Total **132,595,594 bytes /0.12349GiB logical**, with exact allocated bytes
in the proposal JSON. These are unselected snapshots, not asserted byte
duplicates. All are older than24h; no live literal reference was found.
Retain each cell's `latest.pt`, `latest.nnue`, result, manifest, history,
parity and game evidence. Do not delete selected `best` checkpoints from
older experiments that actually used best-validation selection.

The largest other group, `session17f/` (3.537GiB logical), is mostly the
**actively reused Go build cache**, with writes through today. It is not
an abandoned data reconstruction and is excluded. Recent rebalanced/WDL.50
artifacts and S4 outputs are excluded too. No file has been deleted here.

The only single in-scope data retirement large enough to fund either large
option is **`session57-data-step2/t74full.bf`:183,371,624,416 bytes**.
This is a separate Bryan decision: archive the exact file to an approved
destination, or accept losing immediate byte-for-byte retraining access to
the accepted net's old dataset. Preserve its existing hash/recipe/source
mapping and accepted checkpoint. Public raw-source pins make reconstruction
possible in principle, but do not guarantee future availability or eliminate
the conversion cost. T76/T80 need not be removed.

Deletion within WSL may not reclaim host C: until trimming/compaction.
Session111's raw-file removal and online trim did not restore enough host
capacity. Do not count projected recovery as free space, lower admission,
or restart WSL during a running gate. Any offline compaction needs its own
approved natural boundary; the exact recovery procedure is outside this plan.

## Three genuinely new-data options

The first two sources are pinned in the existing campaign's
[source inventory](2026-10-09-data-volume.md). Their byte sizes and digests
below reuse that receipt; no source payload was fetched or reverified here.
The [original official-stockfish data catalog](https://huggingface.co/datasets/official-stockfish/master-binpacks/tree/1e095a758c630bc58d0b6dac4da44fcd38ac89c2)
is the data source, not an engine/trainer-code import. Lc0 explicitly permits
reuse of its training data under the published data licenses in its
[open-data statement](https://lczero.org/blog/2021/06/the-importance-of-open-data/).
No third-party weights or implementation would enter NGN.

| Option | Input and fixed proposed output cap | Peak host-C admission budget | Expected value and cost |
|---|---|---|---|
| **A: T60/T70 mixed-source sample, recommended** | `T60T70wIsRightFarseer.binpack`,32,981,564,118bytes (30.716GiB), SHA256 `8c329b988c02bdf4ccc13b3cc627f69531b8b6bb1719f17f775f49476d045c70`; up to1B eligible mapped32-byte records (29.802GiB) | **100GiB** before download:25floor+30.716raw+29.802mapped+2scratch+10training/headroom+2margin, rounded up | Unseen run distribution with the existing conversion/scoring route. Less new decoder work than native Lc0 chunks. Mixed-source score calibration and duplication may reduce value; no Elo promise. Multi-hour conversion needs approval. |
| **B: fixed T75 sample** | `farseerT75.binpack`,46,983,203,020bytes (43.757GiB), SHA256 `f1907be568d4a48297dd28c2060c53b07ee4170e4e9392faa525d6d697948441`; same1B/29.802GiB mapped cap | **113GiB** before download using the same budget terms | Genuinely absent from accepted training; compatible with the previous source workflow. Larger transfer and same-family overlap make it less attractive thanA. The old inaccessible forge conversion supplies no usable rig bytes. |
| **C: fresh NGN self-play** | Accepted fe92ced/81215229/final14 teacher; fixed5000nodes/move and existing1000-node opening verification; first10M eligible records (0.298GiB), new recorded seeds | **38GiB** before bulk generation/training:25floor+1shards/sidecars+10headroom+2margin | No download; targets the engine's own search distribution. Current H512 generation rate is unmeasured and CPU time competes with gates. First a separately approved bounded10-minute calibration, then freeze shard/game count and runtime budget. Weak-teacher feedback and tiny scale are risks. |

Admission budgets are conservative proposed caps, not measurements. The
existing scatter/gather may need more than2GiB of scratch at some sampling
rates: a bounded calibration must establish the limit before a bulk run.
Do not launch with an assumed scratch cap. Choose a deterministic sampling
fraction from source statistics before conversion; do not truncate on score
or keep trying fractions after training results. Retained counts are records,
not distinct boards. Measure occupied-board overlap with T80/T76 before a
training nomination, using the established bounded hash-sample approach.

OptionC deliberately uses the existing **node-budget** generator. It has no
fixed-depth CLI; calling this a fixed-depth plan would be inaccurate. The
historical HCE pilot's throughput is not a forecast for current H512 NNUE.
No generator or calibration is launched in this session.

If Bryan selectsA/B and approves full-T74 retirement, a future data trial
would use retained T80+T76 plus the nominated new source, recording that the
old T74 source was replaced. This changes the data distribution; it is not
an isolated raw-volume experiment. Keep KB8/H512, WDL.75, batch16384,
optimizer, initialization seed,115x256M exposure schedule and final115
selection fixed to the accepted recipe. Board-disjoint validation spans all
sources. Finalize by identical reexport/20 integer parity, then fixed1000
against the immediate accepted source/net/vector, printed and exact
lower95>0/zero faults. A winner requires400 at60+0.6/point>0 before foreign
games. Freeze the actual corpus/compute/holdout manifest before training;
do not start another recipe series on the old corpus.

## Decision needed from Bryan

Select **A, B or C**, and approve a storage route. ForA/B the concrete
large space candidate is the **170.778GiB full-T74 mapped file**, which is
still an accepted-net training input; decide archive destination versus
retirement explicitly. Alternatively provide enough real host-C capacity
to meet the chosen admission without retiring it. The six-path0.12349GiB
cleanup is optional and cannot solve the capacity problem.

Until then: GPU idle; no replacement cell, download or deletion. S4's
existing timing finishes naturally, native warm SPSA resumes under335882,
then final-vector games precede the four checked search SPRTs. No pending
data decision holds those CPU verdicts.
