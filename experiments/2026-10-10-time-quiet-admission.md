# Immediate iteration admission on the quiet-speed base — session43

The existing admission candidate changes one condition: read the clock at
each new-depth boundary, retaining the 1024-call throttle inside an iteration.
The accepted-net 120+1 trace and prior mechanism/full checks are already
reviewed; do not repeat them. This is a time heuristic requiring real games.

Actual immediate base is **56b35ef/be34532c**, including the reviewed quiet
perspective speed change. Apply only the existing admission patch and its
four tests. Source remains uncommitted in `night/time-quiet-admission` until
a complete strength verdict; the tracked patch preserves it.

The detached rig controller waits natively for capture9929 to release13,15,
then runs the four controls, full short/race/vet and explicit-PGO builds on
this new source combination. It writes checked source/binary/net/opening/
harness hashes, exact argv, OS/toolchain and concurrent jobs before games.
Fresh A/A40 at60+0.6 must contain zero in both printed and unrounded paired
95% intervals, with zero faults. Only then paired SPRT[0,+6], alpha/beta.05,
minimum400/cap4000 starts on the same rig13,15/concurrency2/nice19 topology.
GOMAXPROCS1, Hash64/Threads1/OwnBookfalse/UseNNUEtrue, original616b1ebe PGO,
974e4b5a openings and9b0002f8 harness. The smaller topology is declared before
any games, replacing the prospective8-game topology; no other sample is pooled.

Accept only completedH1 AND printed/unrounded paired lower95>0 AND zero
flags/crashes/illegal/no-move results AND intact inputs. H0 or cap shelves
without extension. No automatic adoption, completed-sample replay, net change,
clock-policy stack or10+0.1 verdict. The scope caps memory at2500MiB/swap0;
4GiB training and3GiB provenance scopes remain below the12GiB rig budget.

The base stays frozen while this gate runs. Expanded-data training is independent;
its next game manifest must nominate the then-accepted runtime/net and free
CPUs after review. Cap4000 on two games can take several days; actual SPRT
completion may be much earlier. The controller has a96-hour gate deadline;
an interrupted deadline is no verdict and never authorizes a restart.
The queued host copy retains its earlier40–60-hour estimate; this corrects
the estimate without changing any frozen executable input. Terminal output appears in
`session43-time-admission/{controller.exit,controller.done,gate-summary.json}`.
