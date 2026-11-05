# Fractional LMR rig pre-screen

Status: DEFERRED. No strength verdict is available; only the two-game harness smoke is run here. The composite gate nomination awaits the rig pre-screen below.

## Required rig procedure (verbatim)

Commit experiments/burn-1104/lmr-stat/ with identity.json, the log of one two-game fixed-node harness smoke, nomination.json with verdict DEFERRED, and PRESCREEN.md holding this requirement verbatim for a later rig deep-run: Pre-screen with the option as the only difference: 'mkdir -p scratch; GOAMD64=v3 go build -trimpath -o build/ngn .; go build -o build/sprt ./cmd/sprt; build/sprt -lowpower=false -new build/ngn -base build/ngn -genbook 2000 -genplies 6 -balancecp 100 -balancedepth 8 -seed 1104 -out scratch/book.txt; nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104LMRFractional=1 -base build/ngn -nodes 40000 -openings scratch/book.txt -concurrency 4 -elo0 0 -elo1 5 -mingames 1000 -maxgames 4000 -maxmoves 300 -pgn scratch/games.pgn | tee evidence/prescreen.txt'. Stop at the tool's own verdict or at maxgames, never on a peek; this takes about 2 hours. It is a filter at 40,000 nodes per move, not a 10+0.1 result and not an Elo claim. Verdict: NOMINATE if the tool prints H1 or the Penta Elo lower bound is above 0; otherwise SHELVED. Either is a complete result, with no parameter retries. Write evidence/nomination.json with option, verdict, penta Elo, interval, games, W/D/L and pentanomial counts. On SHELVED the branch carries no engine change and the patch stays at evidence/candidate.patch. Registered gates pass with the option off. Do not run the pre-screen here. A review reads the commit, so these files must be in it; write KEEP and final.md before finishing.

## Candidate construction

This is an original implementation of the requested fractional-reduction and combined-history idea, using only NGN code and the task description; no other engine source was consulted or copied. The base is NGN’s existing logarithmic formula stored at 1024 units per ply. Main, continuation and follow-up histories have equal weight through GetHistoryScore; the continuous correction uses the existing HistoryLMRDivisor and HistoryLMRCap. PV, improving, TT-PV and killer offsets retain their existing one-ply magnitudes. The starting head has no cut-node LMR offset; the candidate adds one ply at cut nodes only when enabled. Root quiet reductions also use the fractional helper, omitting the unavailable follow-up ancestor. Clamp to [0, depth-2] before nearest-ply rounding (half up). The option-off reduction code is retained.

The smoke uses 1,000 nodes per move and a 20-ply cap to check UCI option transport and legal color-swapped games; it is neither the 40,000-node pre-screen nor an Elo measurement.

The committed smoke transcript drops trailing whitespace from the raw harness output so it satisfies the repository whitespace gate; game results and text are unchanged.
