# Matched expanded-data training — session42

Prepared before training. The only treatment is the mapped T80/T76/T74 corpus,
using the exact accepted architecture and its original training recipe. Review
the actual architecture terminal and the complete corpus/scale/coverage receipts
before writing the concrete `nomination.json` and launching. This preparation
does not nominate a net or change the architecture decision rule.

Retain fresh random initialization, seed20261006, H512/KB8/NB8, factorizer off,
58 complete epochs of256M presentations each, batch16384, AdamW LR.001,
weight decay.01, WDL.75/K400, cosine to.1LR, BF16, two CPU feeder threads,
block size262144, and lowest full content-holdout validation loss. Use the
same board hash/modulus1024/split seed20261006 for all sources. Source paths
are sorted by the existing reader before seeded block/row shuffling. No
per-source exposure weights, label remapping, sampling change, width increase,
longer schedule or borrowed checkpoint is added.

The exact T74 sample remains194 per mille/seed2026100903 with the reviewed
conflict exclusion and per-source scale fit, then the original calibrated
score map. The T74 descriptor stays private on the rig; public experiment
records include its byte/hash pin without its token URL. T80/T76 stay in place
on the rig and T74 is read from the immutable forge server. Corpus coverage
reports estimated distinct occupied boards separately from records, with
its approximation and game/symmetry limitations. Review against the planned
2–3× distinct target before launch.

The detached training controller uses rig12,14/nice19/4GiB/swap0 and appends
to the campaign status. Require25.5GiB C: at admission; a silent60s disk check
stops only this trainer's process group below25.25GiB, leaving a checkpoint
reserve above the mandatory25GiB floor. Retain interruptions without an
automatic restart. This is needed at the observed26.166GiB host capacity.
Write exact source, corpus pins, argv, accepted
base/net, machine/environment and concurrent workloads in its training
manifest before the trainer starts. Require all58 epochs/14.848B exposures,
byte-identical best reexport and20 exact integer parity cases. A completed
training stage remains unaccepted until a session reviews it.

Strength admission is a fresh actual-base A/A40 and exactly400 paired games
at10+0.1, GOMAXPROCS1 per engine, Hash64/Threads1/OwnBookfalse/UseNNUEtrue,
openings974e4b5a and the unchanged harness9b0002f8, with taskset/nice19.
Freeze the available topology and exact game manifest before its A/A; never
overlap its engines with another job on those CPUs. Accept only completed
printed AND unrounded paired lower95>0 and zero flag-outs/crashes/illegal/
no-move results, intact inputs. Otherwise shelve without extension. No
automatic games, default-net change, claim or publication follows training.

Session47 preserves trainer13201 unchanged and moves finalization to forge
under the09:00 routing directive. Only waiting parent trial.py13172 is
SIGSTOPped; trainer and disk watchdog remain active. Native training
watcher15998 records `training-terminal-session47.json` at child completion
and retires only that stopped parent, preventing its old rig Go parity
command. The original controller's resulting nonzero exit is an intentional
postprocessor interruption, not a new trainer verdict. Do not resume it.
After full58, copy terminal/history/config/best.pt/best.nnue and original
20 parity records to forge; run the separately staged
[finalizer](2026-10-10-expanded-data-forge-finalize.py) inside the allocation.
Require identical reexport and20 exact integer Go parity cases there before
review/nomination. CPU dependencies retain torch2.7.0 (CPU wheel)/numpy2.5.3;
no corpus, recipe, initialization, checkpoint or selection changes.
Strength admission now uses a fresh forge A/A at the exact game topology;
there is no new rig CPU stage.
