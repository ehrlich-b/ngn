# Combined provenance certification — session39

Frozen before games. The reviewed key follow-up is provisionally accepted as
`6c0fc26/be34532c`: 400 games, -6.0807 [-25.8905,+13.6895], zero faults,
under Bryan's provenance upper95>0 exception. The original failed key sample
stays shelved. The three provisional replacements now need their combined
CLAUDE.md batch certification against pre-replacement `5b882e2/be34532c`.

Compare the current accepted runtime (own four-way TT, own board/key layout,
own SEE values, and its separately accepted null-frame speed improvement)
against that pre-batch runtime. This certifies the integrated package; it
does not attribute an Elo gain to individual replacements or to the speed
change. Both use the identical accepted net and original PGO profile.
No time-policy, tuned vector, factorizer or NGNN4 overlay is included.

Use the rig, real-clock **60+0.6**, CPUs0-11/nice19, concurrency6,
GOMAXPROCS1, Hash64/Threads1/OwnBook=false/UseNNUE=true, openings974e4b5a,
unchanged sprt9b0002f8, maxmoves400, gametimeout900. A fresh A/A40 at this
exact topology must have printed and recomputed paired95% intervals
containing zero and no operational faults. Then SPRT[0,+6], alpha/beta.05,
minimum400 games, cap4000. The cap is fixed and is not extended.

Certification requires a completed H1 verdict, final unrounded paired
lower95>0, zero flags/crashes/illegal/no-move/watchdog/voided results, and
intact pinned inputs. H0 or an inconclusive cap requires batch investigation
and bisect; neither licenses a provisional keep on sign. Record any games
left outside a completed pair when the pinned harness drains after its stop.
The paired statistic alone decides. No automatic merge, rollback or net
promotion follows; a session must review the terminal evidence.

The scope has MemoryMax3GiB, MemorySwapMax0 and CPUQuota1200%. It may overlap
the unchanged4GiB GPU training on12,14 and the later2GiB architecture gate
on13,15. These workloads and resource limits are recorded before the A/A.
The hopper remains untouched. Keep C:>=25GiB; require25.5GiB at admission.
The fixed400-game architecture gate retains its own immediate accepted base
and net; its evidence is not pooled with this certification.

Results live at `night-20261007/session39-provenance-batch/`. After checked
forge source/binary transfer, write the exact commands, source/net/binary/
opening/harness/profile hashes, machine/OS/Go/environment and checks receipt
in `game-manifest.json`, then detach the controller. A/A should take about
15 minutes; minimum400 about2-3 hours; cap4000 may take20-30 hours. Observe
terminal/status notifications or inspect once next session, not a poll loop.
