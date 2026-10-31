# Broader Leela-derived corpus — session19

The merged October9 plan ranks unique data ahead of another wider/longer net
or an architecture trial. This session prepares data and an unchanged recipe;
training waits for reviewed immediate-base nomination and complete corpus
coverage. Accepted runtime stays5b882e2 / be34532c.

Source inventory: [official-stockfish/master-binpacks](https://huggingface.co/datasets/official-stockfish/master-binpacks/tree/1e095a758c630bc58d0b6dac4da44fcd38ac89c2),
pinned revision1e095a758c630bc58d0b6dac4da44fcd38ac89c2. The existing inventory
is `data-farseer-20260928/tree.json` on WSL. All are data, not nets or imported
engine/trainer code. T60/T70 is a mixed source and is deferred in favor of
T76/T74 for this first volume experiment. The [dataset guidance](https://github.com/official-stockfish/nnue-pytorch/wiki/Training-datasets)
identifies Leela-derived sources as useful, while cautioning that label quality
alone does not predict trained strength. ODbL terms remain those in
`LICENSES/TRAINING-DATA.txt`; no distribution or publication is authorized.

| Archive | Bytes | SHA-256 | State/use |
|---|---:|---|---|
| T80 August2022 | 10,809,713,086 | 0d22957b8d4f0f8e6f2913be7b744b2dab5178c3563e0f916312d0d94c28b92b | Existing accepted2.374B-record mapped corpus retained |
| farseerT76 | 6,141,321,549 | 255956f2d28d78e5eb050aa542d768b0d8e0667ea09bb250effc83706f26da54 | New campaign copy downloaded00:11; full eligible sample planned |
| farseerT74 | 20,144,023,865 | cebf6e5aa62a0df447f3748c90a542ded13105c873a1a819d7f71bdf041ca8cb | Downloading; deterministic sample sized from its complete statistics |
| farseerT75 | 46,983,203,020 | f1907be568d4a48297dd28c2060c53b07ee4170e4e9392faa525d6d697948441 | Deferred for C: budget |
| T60T70wIsRightFarseer | 32,981,564,118 | 8c329b988c02bdf4ccc13b3cc627f69531b8b6bb1719f17f775f49476d045c70 | Deferred mixed-source follow-up |

The external NGN archive helper already supports these exact byte/hash pins,
chain splits, quiet/not-in-check filtering, conflict-key exclusion,
key-stable keep-per-mille sampling and source score/result bins. It remains
outside this engine repository, with its previously disclosed sfbinpack decoder;
none of that tooling is copied into NGN's engine or trainer. No foreign net
or source is used in the new runtime candidates.

Sequence in `session19-data/`, CPU13,15 / nice19 / workers2:

1. Accepted KB8 twelve-probe120+1 spending trace, then compile/test the small
   space-bounded gather patch. Ordinary/consuming gathers must produce identical
   bytes, with both signs, rounding ties, sentinels, input scales and empty buckets.
   **PASS:** two Rust tests and28 mapped records/16 dropped sentinels;
   byte hash6d0fe1623929e97db55fc306c2c842f5d385b725c772f27404750a748d1e571c.
2. T80 statistics-only scatter, keep0. It supplies the reference score/result
   bins for the existing per-source outcome-scale fit. No existing corpus is
   rewritten. `preparation.done` releases the TT/time CPU checks.
3. Preserve factorizer and both CUDA windows; wait for sparse CUDA651274's
   actual completion marker and the compressed downloads.
4. Full T76 scatter/gather; fit its raw scale from score/result bins relative
   to T80, then apply the same T80 SF18-fitted64-knot map. This transfers the
   calibrated T80 map; it is **not a fresh per-source SF18 calibration**.
   Capture source sentinel rates, bins, scale and residual before training.
5. Statistics-only T74 pass; choose keep-per-mille from the eligible count
   before training so combined new mapped/scatter data fit an85GiB budget.
   Scatter that fixed sample and map with its own fitted raw scale.
6. Estimate distinct occupied boards across T80/T76/T74 with a fixed65,536
   bottom-hash sample. Read all rows once; retain only small hash samples.
   `coverage.json` records record volume, estimated distinct volume/ratio and
   approximate uncertainty. This counts occupied boards, not game groups or
   mirrored-input classes. Review whether the2–3× unique-position target was
   reached; do not rename a record count as a unique count.

Consuming gather flushes and syncs the mapped output before removing each
**new campaign** input bucket, retaining identical conversion/shuffle bytes.
After a complete source gather its **new campaign** downloaded copy is unlinked;
the full input byte/hash receipt and corpus manifest persist. It never consumes
old T80 buckets or deletes pre-campaign data. Peak planned new scatter/mapped
storage85GiB plus compressed sources24.5GiB fits the startup138.9GiB free.
A silent tool-side guard checks C: every60s (a concrete disk safety need) and
stops only this controller's descendants below32GiB, leaving margin above25GiB.
No other job, hopper, old data or live artifact is touched.

The first download's WSL PowerShell interop call failed after T76 finished.
Its log/exit/done are retained as `download-initial.*`; corrected continuation
uses `df /mnt/c`, skips the completed-size T76 copy, and downloads only T74.
The converter verifies each source's exact hash once at use. This is a data
transfer continuation, not an extended strength/training experiment.

Prospective **data-only** training: original accepted trainer sourcea49ebfc;
NGNN3 KB8/H512/NB8, batch16384, LR.001,58×256M presentations, WDL.75/K400,
cosine→.1LR, decay.01, BF16, seed20261006, two CPU feeder threads, keep-best-only.
Use the same content hash/modulus1024/split seed20261006 across all three files;
identical occupied boards across sources remain in one partition. Holdout
selection remains lowest native validation loss on the resulting data holdout.
No batched/sparse numerical change, hidden layer, width or longer schedule
is included. If factorizer is accepted first, retain its accepted setting and
freeze that revised immediate net/source before launch; otherwise factorizer off.

Once corpus/base review is complete, write exact input hashes/trainer argv and
game manifest **before launch**. Train from random initialization, not a foreign
or borrowed checkpoint. Require58 contiguous full epochs, byte-identical
reexport and20 exact integer parity cases. Fresh A/A40 then fixed400 paired
10+0.1 games against the actual immediate accepted base; CPU0–11 / nice19 /
concurrency8 / GOMAXPROCS1 / Hash64 / Threads1 / OwnBook=false. Complete penta
lower95>0, zero flags/crashes/illegal/no-move results and intact inputs accept;
otherwise shelve without extension. No loss metric supplies an Elo verdict.
No training or automatic promotion is currently queued.

Launch handles: corpus dispatcher655799; T74 download656694 (initial completed
transfer655634 failed at its next capacity read); C: guard656725;
coverage656801. All waits use native pidfds/file notifications; there are no
assistant polling loops. Corpus `controller.done/exit`, per-source manifests,
`sample-plan.json`, `scales.json`, and `coverage.json` are the next data receipts.

Session19 wrap: both downloads complete, T80 stats/consume-gather controls pass.
The new Rodent priority holds factorizer parity/games; admission666724 therefore
runs the unchanged queued CUDA cost windows on12,14 directly after full training.
Data dispatcher655799 is held by666748 until TT's13,15 game window ends, then
continues its existing sparse-CUDA/download dependency and corpus/coverage chain.
No actual trainer is queued. C:111.5GiB free at01:22; guard656725 remains active.

Session21: T76 is complete55.63GB/1.738B mapped rows; T74 scatter ended143
on the32GiB space guard, and coverage failed its dependency. Removed only
13.15GB of incomplete campaign T74 buckets. HostC:10.96GiB did not recover
through WSL trim. Continuation675046 waits natively for provenance675044,
then70GiB free before the same194-per-mille T74 sample/score mapping and
distinct-coverage scan. No trainer follows automatically.
[Continuation and disk receipt](2026-10-09-session21.md).
