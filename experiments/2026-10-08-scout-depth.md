# Later-move scout depth — prospective session 18 strength candidate

Accepted runtime **5b882e2 / be34532c**; source snapshot **65a2a59** has
the same accepted engine plus campaign records. Topic `night/scout-depth`,
worktree `output/campaign-scout-depth/`. One search behavior changes.

The move loop computes `nextDepth` after check, passed-pawn and singular
extensions and the path extension budget. First moves and verification
searches use it. The initial scout for every later move instead uses
`depth - 1 - reduction`, losing both extensions and budget clamps. A scout
that fails low never gets the intended depth. Use `nextDepth - reduction`
for that initial scout. Keep all extension/reduction rules, root search,
parameters, evaluation, net and training unchanged. This is NGN-owned code.

Two integration cases force a checking rook move and a seventh-rank passed
pawn push behind a quiet TT move. A depth-one exact child entry supplies
a fail-low result that covers an ordinary child but cannot cover either
extended child. The accepted code stops at that entry; the candidate must
search a reply and unwind a requested stop with all caller state intact.
Require both cases to fail on the baseline and pass on the candidate.
Then require engine short/race, full short/race and engine vet on WSL.
No correctness classification bypasses the strength gate.

Before games, freeze both PGO-built binaries, net, openings, harness, source
archive, patch and exact commands in `game-manifest.json` and a checksum file.
Use Go1.25.5 / amd64 v3, explicit accepted PGO616b1ebe, WSL2 / Ryzen9800X3D.
All builds/tests use CPU13,15 / nice19 / GOMAXPROCS2. Games use CPU13,15 /
nice19 / GOMAXPROCS1 per engine, concurrency2, **10+0.1**, Hash64, Threads1,
OwnBook=false, accepted NGNN3 KB8 H512 netbe34532c on both sides. The pinned
opening corpus974e4b5a has5000 lines; reversed-color pairs start at its beginning.
The frozen, previously validated SPRT harness9b0002f8 is unchanged.

Run exactly **40 A/A games** with the freshly built baseline on both sides,
requiring the penta interval to include zero and zero operational failures.
Only then run exactly **400 games /200 reversed pairs** candidate versus
immediate accepted baseline. Disable SPRT early stopping with mingames1000000.
Accept only complete printed **and independently recomputed penta lower95>0**,
zero flag-outs, crashes, illegal or no-move outcomes, and intact pre/post inputs.
Otherwise shelve without extension. Incomplete/invalid runs have no verdict.

The detached controller writes `status.jsonl` stage `scout-depth`, with results
under `/home/ehrli/ngn-data/night-20261007/session18d/`. A/A is bounded30min,
gate150min, whole controller195min; no retry or automatic promotion. This
window ends before the factorizer's projected02:24 training completion and
preserves its gate and subsequent batched/sparse CUDA windows. Factorizer,
SPSA and hopper stay untouched. No GPU use, local heavy work or publication.
