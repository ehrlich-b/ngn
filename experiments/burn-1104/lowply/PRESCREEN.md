# Low-ply history rig handoff

Status: DEFERRED. Only a two-game harness smoke was run on this host.
The smoke is not a strength estimate. Nominate this option for the rig composite gate
only after the specified pre-screen; the candidate remains default off.

Pre-screen with the option as the only difference: 'mkdir -p scratch; GOAMD64=v3 go build -trimpath -o build/ngn .; go build -o build/sprt ./cmd/sprt; build/sprt -lowpower=false -new build/ngn -base build/ngn -genbook 2000 -genplies 6 -balancecp 100 -balancedepth 8 -seed 1104 -out scratch/book.txt; nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104LowPly=1 -base build/ngn -nodes 40000 -openings scratch/book.txt -concurrency 4 -elo0 0 -elo1 5 -mingames 1000 -maxgames 4000 -maxmoves 300 -pgn scratch/games.pgn | tee evidence/prescreen.txt'. Stop at the tool's own verdict or at maxgames, never on a peek; this takes about 2 hours. It is a filter at 40,000 nodes per move, not a 10+0.1 result and not an Elo claim. Verdict: NOMINATE if the tool prints H1 or the Penta Elo lower bound is above 0; otherwise SHELVED. Either is a complete result, with no parameter retries. Write evidence/nomination.json with option, verdict, penta Elo, interval, games, W/D/L and pentanomial counts. On SHELVED the branch carries no engine change and the patch stays at evidence/candidate.patch. Registered gates pass with the option off. Do not run the pre-screen here. A review reads the commit, so these files must be in it; write KEEP and final.md before finishing.

Implementation provenance: independently written from the low-ply history concept
in the task specification, using NGN move packing, worker ownership and its existing
gravity update. No other engine source was consulted or copied. The table has
five ply slices, each keyed by from/to, and uses a bounded depth-squared bonus
only for ordinary quiet beta cutoffs. Captures, en passant, promotions and castling
retain their existing scoring paths. No low-ply value enters pruning or reductions.
The per-worker allocation is 160 KiB on 64-bit hosts and is reset with quiet history.

The smoke uses 1,000 nodes per move and a 40-ply cap to exercise option routing,
node budgeting and paired games quickly. Its draw results are not used for nomination.
The committed identity compares the starting head at depth 10 on CPU 3 across
HCE and the already supplied NNUE network, using both search drivers.
