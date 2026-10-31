# Frozen SPSA engine live-option audit — campaign session 11

Session starts **15:17:40 EDT**, deadline **16:57:40 EDT, October 8**.
Accepted source/net remain **5b882e2 / be34532c**. The longer-net verdict,
v3 operational admission and factorizer CUDA check are pending. SPSA's
synthetic self-test checks update math, not the search-option circuit.
This bounded audit checks that circuit on its actual frozen engine.

Before execution: pin the current engine/tuner binaries, wrapper, frozen
search/UCI/default-net/driver sources and embedded accepted net. Reconcile
all 14 registry names/defaults/ranges against the actual advertised UCI
options. Record executable search references for every parameter.

Run exactly **30 configurations / 180 searches**: unchanged defaults,
each of the 14 knobs separately at both declared endpoints, then unchanged
defaults again. Six existing owned speedprobe positions; cold game state
per position, fresh process per configuration. **Depth 14 / 1,000,000-node**
caps, 30-second protocol deadline, whole controller bounded at 30 minutes.
Use the frozen tuning engine's actual defaults: embedded `be34532c`,
UseNNUE true, OwnBook false, Threads 1 and Hash 128. Never substitute
the later speed engine or a mid-run theta. Require initial/final defaults
to match depth/nodes/score/bestmove/PV exactly, clean exits and stderr, and
unchanged input hashes. Endpoint differences provide concrete per-knob
search witnesses. Report an unwitnessed knob as a diagnostic limitation;
do not infer that it is dead, extend the sample or change the tune.

WSL only, CPUs **13/15**, at most one engine per CPU, nice **19**,
GOMAXPROCS=1. No games, timing gain, tune convergence or Elo verdict.
Existing trainer, tuner, their frozen source and queued controllers remain
untouched. The shared hopper remains untouched. Commit/push this manifest,
[controller](2026-10-08-spsa-live-audit.sh) and
[helper](2026-10-08-spsa-live-audit.py) before executing. New root
`/home/ehrli/ngn-data/night-20261007/session11/`; immutable audit inputs,
raw per-configuration UCI streams, `registry.json`, `source-pins.json`,
`audit-receipt.json` and terminal markers stay there. Launch once:

```bash
cd /home/ehrli/ngn-data/night-20261007/session11
setsid nohup timeout 1800 bash controller.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
```

This verifies a prerequisite of the already-running SPSA lane. Only a
converged vector after >=20k retained games and its fresh A/A40/gate400
against the immediate accepted base can establish a tuning gain. If the
longer net wins, this unfinished old-base tune cannot transfer as evidence.

## Completed audit

Manifest/controller/helper **a57f452** committed and pushed before execution.
Detached wrapper **587464** started **15:24:52 EDT**, completed **15:25:22 EDT**,
exit zero. Actual tuning engine **f09b76b7** and tuner **67cf935d** retain their
original bytes. No builds, training, games or timing measurements were launched.
All **180 searches / 30 configurations** completed. All 14 advertised options
match the frozen registry; all 14 have executable search references and concrete
endpoint search witnesses. Default-before/default-after records match exactly.

| Parameter | Differing endpoint/position records out of 12 |
|---|---:|
| NullMoveR | 10 |
| NullMoveMinDepth | 10 |
| LMRFullDepth | 6 |
| LMRMoveThreshold | 12 |
| FutilityMargin | 12 |
| FutilityMaxDepth | 6 |
| ReverseFutilityMargin | 12 |
| DeltaMargin | 12 |
| SingularDepth | 12 |
| SingularMargin | 12 |
| ExtensionBudget | 5 |
| AspInit | 12 |
| AspMult | 12 |
| LMPBase | 12 |

The [independent review](2026-10-08-spsa-live-review.sh) reparses all raw streams
using token fields rather than the probe's regular expression, reconciles every
depth/node/score/PV/bestmove and witness, verifies command/CPU pins, clean exits,
empty stderr, all frozen hashes and repeated defaults. It passes **15:26:35 EDT**.
[Review and one witness per knob](2026-10-08-session11-results/final-review.json),
[source pins](2026-10-08-session11-results/source-pins.json),
[registry/read lines](2026-10-08-session11-results/registry.json),
[launch custody](2026-10-08-session11-results/launch-observed.json).
Full transcripts and witnesses remain in box `session11/`.

The existing training, admission and CUDA hash lists all pass, with
[receipts](2026-10-08-session11-results/dependency-check.json). The first read-only
admission hash check used the wrong working directory and rejected its three
relative paths. Correct directory-specific checks passed before launching any
engine; initial check output is preserved. No queued script or frozen input changed.

Source inspection also bounds the resume claim: this driver reloads theta and
iteration but restarts its seeded perturbation/opening RNG. It does not reproduce
an uninterrupted optimizer trajectory. Leave the current frozen driver intact;
bind RNG and recipe identity before a future fresh tune. Current retained games
remain optimizer work, with no strength claim before the final paired gate.

[Final box snapshot](2026-10-08-session11-results/box-state.json), **15:27 EDT**:
longer H512 **58/116** epochs, latest **284.95 s**, GPU **69% / 4,197 MiB**;
SPSA durable **375 / 4,500 games** at **15:01:04 EDT**. Trainer **541007** on
12,14, tuner **577984** on 0–11, dependency controllers **522940 / 552631 / 577875**
and hopper **1168** are live and untouched. C: free **146.5 GiB**.
No accepted net/search change, final nomination, screen, claim or next trainer.
