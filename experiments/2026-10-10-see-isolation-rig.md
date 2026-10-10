# SEE-only isolation on the rig — session49

Plan amendments at11:50 and13:05 authorize rig games again. The queued
forge session48 waiter is policy-interrupted before A/A or candidate games;
zero games, no verdict, no pooling. SPSA remains running on forge.

Use the same archived checked SEE-only pair39eb128 versus the exact table
patch integrated asf8686be, acceptedbe34532c, originalsprt9b0002f8 and
openings974e4b5a. No rebuild or repeated source checks. The historical
reference is measurement only; no derived code is restored or adopted.

Fresh independent rig sample: CPUs0–11, c6, nice19, GOMAXPROCS1 per engine,
Hash64/Threads1/book-off/NNUE-on, scope6GiB/swap0. Concurrent training
13201 stays on12,14/GPU under its4GiB scope; hopper1170 stays untouched.
Keep WSL near12GiB total and C: at least25GiB. Forge pinning has no bearing
on this rig sample. Record concrete machine/OS/Go/argv/input pins before
launch in `session49-see-isolation/game-manifest.json`.

Unchanged diagnostic rule: fresh A/A40 at10+0.1, printed and unrounded
paired intervals contain zero, zero operational faults and intact inputs;
then exactly1000 games/500 reversed pairs at10+0.1. A/A deadline1h,
diagnosis8h. Complete counts, zero flags/crashes/illegal/no-move results
and intact pins required. Printed AND unrounded upper95<0 implicates SEE;
lower95>0 is positive evidence; overlap zero is unresolved. No automatic
adoption, certification or rollback. A confirmed loss calls for original
values redesigned from scratch. No extension or rescue games.

The expanded-data gate uses whichever lane is free first after full58,
reexport20-parity and session review. It may use forge c2 with its own
fresh A/A (SPSA paused at a durable checkpoint), or rig after this diagnosis.
No pooling between machines; an interrupted run has no verdict.
