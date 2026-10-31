# Separate helper-policy preparation

Astra's ranked second and third policy candidates are prepared separately on
f1d0a03/a4d90b41/final14. This fits the admitted three-candidate S queue;
none is integrated or automatically gated. After each prior verdict, rebase
only the next single policy on the immediately selected source and freeze its
own Threads4/Hash256/c1 A/A40/fixed400 rule before games. Do not combine them,
reuse the consensus sample/control, or infer Elo from their check receipts.

Root ties: preserve all existing root ordering scores; keep tactical slots;
rotate only quiet moves within exactly equal-score groups by helper index.
The existing previous-completed-best hoist still runs afterward. Primary path
keeps its original ordering call. Helper private history and interior ordering
unchanged. Focused tests cover permutation, score bands, tactical positions,
determinism, cold helper separation and no history writes.

Depth schedules: all workers search depths1–4. Helpers1/2 thereafter keep
even/odd depth sequences; helper3 keeps all depths (repeat this assignment for
larger widths). Primary keeps every depth. Clamp the last iteration to the
requested maximum even if its parity differs; guaranteed loop progress.
Focused tests exhaust widths/limits1–100 and verify the exact depth sequences.
Cancellation, shared TT, aspiration, histories and root-result ownership remain
existing policies. This differs from the old depth-one startup staggering.

Each gets engine short, full short race at GOMAXPROCS2/4, vet, PGO build and
36 fixed-depth14 Threads1 move/score/node/PV identities on rig CPUs13/15.
They serialize under one detached prep unit. The first consensus S controller
waits natively for that unit's terminal before using CPUs12/14's physical cores;
a prep failure frees the cores but leaves that topic unchecked and unadopted.
No CPU overlap inside the S control/gate, no GPU treatment or foreign artifacts.
