# Continuation history at four and six plies

Candidate: T1104ContPlies=1 versus the shipped default 0, based on
6d3168b8c5f15532cc7a352d05465a77ab50f361. Verdict: **DEFERRED**.
Nomination: send this candidate to the rig pre-screen; advance to the rig's
composite gate only after the prescribed NOMINATE result. No strength claim.

Implementation is original NGN code from the stated longer-continuation idea.
Plies two, four and six all concern the current side, so the existing follow-up
table is shared. Near histories keep full weight; each farther context reads at
half weight and learns at half the capped rate. This includes the production
completed-winner policy, cutoff rewards, loser penalties, history pruning and
LMR. No new worker table is allocated. Empty/null contexts are skipped.

## Local verification

- Both speed probes were built from cmd/speedprobe with go build -trimpath -buildvcs=false.
  VCS stamping is disabled for reproducible binary hashes across the commit.
  identity.json records CPU 3, depth 10 and 12 equal searches per evaluator
  (HCE and the required NGN network); option 0 is the unmodified default.
- go test -p 2 -short ./engine -run ContPlies covers the spin registry, shallow
  and null stack contexts, half reads, both update policies, winner/loser labels,
  option-off behavior and a completed search with distant ancestors.
- smoke.txt is one paired two-game harness smoke using the same executable on
  both sides, only the candidate option enabled, at 40,000 nodes per move:

```sh
build/sprt -lowpower=false -new build/ngn -newoption T1104ContPlies=1 -base build/ngn -nodes 40000 -concurrency 1 -elo0 0 -elo1 5 -mingames 2 -maxgames 2 -maxmoves 20 -pgn ../evidence/smoke.pgn
```

The smoke uses built-in openings and a 20-ply cap to check plumbing quickly.
Its two capped draws and displayed estimates establish no playing strength.
The nomination's main statistics are null/zero because the pre-screen has not run.

Headless decision: follow the explicit default-off implementation and defer the
long pre-screen as required; no interactive input is needed or requested.
Host policy: hostname forge is matched by exact equality; no kernel or device
checks are part of this change. Only the requested ordinary Go checks run.

## Required rig procedure (verbatim)

Pre-screen with the option as the only difference: 'mkdir -p scratch; GOAMD64=v3 go build -trimpath -o build/ngn .; go build -o build/sprt ./cmd/sprt; build/sprt -lowpower=false -new build/ngn -base build/ngn -genbook 2000 -genplies 6 -balancecp 100 -balancedepth 8 -seed 1104 -out scratch/book.txt; nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104ContPlies=1 -base build/ngn -nodes 40000 -openings scratch/book.txt -concurrency 4 -elo0 0 -elo1 5 -mingames 1000 -maxgames 4000 -maxmoves 300 -pgn scratch/games.pgn | tee evidence/prescreen.txt'. Stop at the tool's own verdict or at maxgames, never on a peek; this takes about 2 hours. It is a filter at 40,000 nodes per move, not a 10+0.1 result and not an Elo claim. Verdict: NOMINATE if the tool prints H1 or the Penta Elo lower bound is above 0; otherwise SHELVED. Either is a complete result, with no parameter retries. Write evidence/nomination.json with option, verdict, penta Elo, interval, games, W/D/L and pentanomial counts. On SHELVED the branch carries no engine change and the patch stays at evidence/candidate.patch. Registered gates pass with the option off. Do not run the pre-screen here. A review reads the commit, so these files must be in it; write KEEP and final.md before finishing.
