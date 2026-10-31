# Owned-teacher data calibration preparation — campaign session 9

Session starts **14:36:10 EDT**, deadline **16:16:10 EDT, October 8**.
Accepted runtime remains **5b882e2 / be34532c**; the longer H512 net has
no verdict. Prepare an alternative owned-data lane while GPU training and
SPSA continue. No real teacher is nominated and no real datagen, trainer,
strength match, screen or claim is queued by this preparation.

## Question and fixed sample

The existing generator accepts NGNN3 and pins the network actually loaded.
Before scaling gen1 with a stronger teacher, measure throughput, retained
quiet-position distributions, score/outcome calibration, draw/cap frequency
and king-bucket coverage. The historical gen0b sample uses the HCE teacher;
it cannot answer these questions for the accepted KB8 H512 teacher.

Two fixed prospective cells: **5,000** and **25,000** nodes per move, each
two workers and **64 complete games per worker**, **256 games total**.
Seeds **2026100800 / 2026100801** in both cells; seeds do not pair games,
because trajectories and opening rejection may differ. Run cells in that
order, no repeats/extensions. CPU **13 / 15**, one worker pinned per CPU,
nice **19**, GOMAXPROCS=1. Preserve the tuner on 0–11. Each cell has a
**1,800-second** deadline; timeout/failure preserves an invalid partial cell
and launches no next cell. No trainer, claim, match or speed probe may coexist.
The shared hopper is untouched. C: must have >=25 GiB before each cell.

Same settings in both cells: Hash 16 MB; maximum 400 plies; random 8/9-ply
openings; 1,000 verification nodes; opening rejection outside +/-400 cp;
six-ply +/-2,000-cp win streak; twelve-ply +/-8-cp draw streak after ply 80.
The selected immediate teacher and generator binary are identical across
cells. Output is bounded by **3.125 MiB** of records for the entire run;
the controller also imposes a per-file size limit. Normal complete-game
closure is required; an interrupted sidecar cannot pass this calibration.

Declare before launch: require exactly 128 complete games/cell, closed
NGNP1 shards with exact settings/seed/source/binary/net pins, data hashes
and byte/position counts, reconciled game results/terminations, zero dropped
or unscored games, valid records/checksums from NGN's existing `datastat`,
zero retained noisy flags, and unchanged frozen inputs. No calibration
score selects or rejects a net or establishes strength.

Report score/ply/piece/result histograms, duplicate estimates, opening
rejection and termination counts, positions/hour, both perspectives'
king-bucket counts, and `mean((sigmoid(WhiteScore/400)-WhiteResult/2)^2)`.
The latter and score-bin outcome means are **record-weighted descriptive
statistics**; positions within games are correlated. No confidence interval,
paired gain or Elo follows from them. Review the data distribution before
declaring a separate full gen1 data/training recipe and its parity/A/A40/
paired gate400 against the then-immediate accepted base.

## Preparation and historical baseline

Use the clean **d414460** archive and accepted PGO **616b1ebe** from verified
`session9b/`. Reconcile all **230** Go/assembly/test/fixture/profile inputs
and **25** trainer inputs with their accepted test receipts. Reuse the
private common engine/parity/SPRT artifacts only after their exact digests
reconcile. Build separate datagen/datastat executables with explicit archive
commit provenance on WSL; no running source/binary is replaced.

Run existing datagen/NGNP/CLI tests (including deterministic whole-game
generation and NNUE/NGNN3 selection), their short/race controls, and the
new audit rejection/analytical-statistic controls. The engine is unchanged;
its required full short/race checks are inherited only through exact
compilation-input equality to the completed delta receipt.

Analyze the same **14 x 65,536 = 917,504** closed gen0b record prefixes as
the completed [holdout diagnostic](2026-10-08-owned-data-holdout.md).
Bind its existing prefix hashes, verify before/after identity, read only
those prefixes, and report the same descriptive score/outcome/bucket metrics.
This is an HCE baseline sample, not a full-corpus audit or new-teacher result.
GPU excluded; preparation CPU 13,15 / nice 19. All tests and analysis run
on WSL. The Mac edits, reads receipts and performs Git operations only.

Box root: `/home/ehrli/ngn-data/night-20261007/session10/`.
[Preparation controller](2026-10-08-owned-calibration-prepare.sh),
[calibration helper](2026-10-08-owned-calibration.py),
[future run controller](2026-10-08-owned-calibration.sh).
The reviewed-base auditor is the unchanged frozen `session9b/trial.py`,
copied as `base-review.py`; only its read-only nomination/receipt checks
are reused. Its factorizer training command is never executed by this lane.

## Activation after pending evidence

This is an alternative data lane, lower priority than the prepared factorizer
trial. Preserve longer-net **522940**, admission **552631** and CUDA **577875**
in that order. A complete clean longer-net verdict selects `kb8h512-long`
only if its fixed gate passes; otherwise retain `kb8h512`. An incomplete,
failed or invalid gate admits neither. The reused nomination auditor binds
the exact six evidence receipts and immediate base recipe as specified in
[the factorizer activation contract](2026-10-08-factorizer-trial.md).
If another net is accepted before this lane starts, this prepared nomination
contract must be refreshed; do not generate with a stale teacher.

After reviewing all raw evidence, commit/push a separate calibration activation
record and its nomination payload before generation. Use the same nomination
fields as the reviewed-base contract, with this calibration record as
`review_record` and the eventual activation commit as `review_commit`.
Freeze `nomination.json`, the exact copied activation record and
`activation-commit.txt` in `activation-inputs.sha256`. Require the calibration
build receipt and all `calibration-inputs.sha256` entries intact. Then:

```bash
cd /home/ehrli/ngn-data/night-20261007/session10
bash calibration.sh --dry-run
setsid nohup bash calibration.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
```

Observe actual startup once and record the PID/status event. Future results:
`nodes-{5000,25000}/{launch.json,seed-*.ngnp,seed-*.ngnp.json,seed-*.log,receipt.json}`,
`launch-plan.json`, `calibration-receipt.json`, `controller.{exit,done}`,
pre/post hash checks and campaign `status.jsonl`. No automatic gen1, training,
net promotion, strength screen or claim follows calibration.

## Result

Preparation **580479** completed **14:46:24 EDT**, exit zero. Startup was
observed through Linux inotify/pidfd. Existing datagen/NGNP/CLI tests and
their short/race checks pass; isolated datagen **485eb86b** and datastat
**a8de2f81** build cleanly. All **230** accepted engine inputs and **25**
trainer inputs equal local bytes; all **2,019** frozen calibration inputs
reconcile. Common engine/parity/SPRT hashes retain the prior verified pins.
[Build receipt](2026-10-08-session9-results/calibration-build.json).

The new helper passes **22** corrupt-sidecar/noisy-row rejection controls,
an analytical Brier fixture and both king perspectives. Independent review
also exercises the successful synthetic composed plan and **6** changed-
build/activation rejections. The actual dry run refuses absent nomination
without creating a started marker or cell directory; the live process guard
separately refuses the existing trainer. These controls start no real engines.
[Controls](2026-10-08-session9-results/controls.json),
[final reconciliation](2026-10-08-session9-results/final-review.json).

The fixed historical sample has **917,504** records: White loss/draw/win
**331,622 / 273,972 / 311,910**, STM White/Black **458,186 / 459,318**.
Mean score **-11.187 cp**, RMS **543.653 cp**, K400 score/result Brier
**0.10270219**. All eight king buckets appear for both perspectives; White
counts **162419/274303/92484/126616/115382/64089/66015/16196**, Black
**150959/279309/89543/121733/117415/68519/70977/19049**. Prefix hashes
are unchanged; an independent read of raw wire label bytes reproduces the
record/outcome/STM counts and all three score metrics. This supplies no
new-teacher, loss-improvement or strength verdict.
[Sample receipt](2026-10-08-session9-results/baseline-receipt.json).

Frozen helper **588c8414**, controller **b3890f4e**, preparation **ff356c39**,
base-review helper **ed10e851**, prospective box manifest **ccee3cf8**;
input list **71d3a6ca**. Result annotations here are separate from that
immutable prospective box manifest. No real calibration is queued.
Final box snapshot **14:51 EDT**: longer net **51/116** epochs, tuner durable
**350 / 4,200 games**, GPU **60% / 4,197 MiB**, C: **146.5 GiB**. Training,
tuner, admission/CUDA windows and hopper remain untouched.
