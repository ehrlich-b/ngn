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
